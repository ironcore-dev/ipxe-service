package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"

	buconfig "github.com/coreos/butane/config"
	"github.com/coreos/butane/config/common"
	k8simages "github.com/onmetal/k8s-image/api/v1alpha1"
	inv "github.com/onmetal/k8s-inventory/api/v1alpha1"
	mreq1 "github.com/onmetal/k8s-machine-requests/api/v1alpha1"
	"github.com/onmetal/machine-operator/app/machine-event-handler/logger"
	netdata "github.com/onmetal/netdata/api/v1"
	"gopkg.in/yaml.v1"

	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	timeoutSecond = 5 * time.Second
)

type httpClient struct {
	*http.Client

	log logger.Logger
}

type event struct {
	UUID    string `json:"uuid"`
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`
}

var (
	requestIPXEDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ipxe_request_duration_seconds",
		Help:    "Histogram for the runtime of a simple ipxe(getChain) function.",
		Buckets: prometheus.LinearBuckets(0.01, 0.05, 10),
	},
		[]string{"mac"},
	)
	requestIGNITIONDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ignition_request_duration_seconds",
		Help:    "Histogram for the runtime of a simple ignition(getIgnition) function.",
		Buckets: prometheus.LinearBuckets(0.01, 0.05, 10),
	},
		[]string{"mac"},
	)
)

func init() {
	prometheus.MustRegister(requestIPXEDuration)
	prometheus.MustRegister(requestIGNITIONDuration)
}

func main() {
	http.HandleFunc("/", ok200)
	http.HandleFunc("/ipxe", getChain)
	http.HandleFunc("/ignition", getIgnition)
	http.Handle("/metrics", promhttp.Handler())
	if err := http.ListenAndServe(":8082", nil); err != nil {
		log.Fatal("Failed to start IPXE Server", err)
		os.Exit(11)
	}
}

func ok200(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "ok\n")
}

type dataconf struct {
	NetdataNS        string `yaml:"netdata-namespace"`
	MachineRequestNS string `yaml:"machine-request-namespace"`
	InventoryNS      string `yaml:"inventory-namespace"`
	K8SImageNS       string `yaml:"k8simage-namespace"`
}

func (c *dataconf) getConf() *dataconf {
	yamlFile, err := ioutil.ReadFile("/etc/ipxe-service/config.yaml")
	if err != nil {
		log.Fatalf("yamlFile.Get err   #%v ", err)
		os.Exit(21)
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}
	log.Printf("Config is #%+v ", c)
	return c
}

func newHttp() *httpClient {
	client := &http.Client{Timeout: timeoutSecond}
	l := logger.New()
	return &httpClient{
		Client: client,
		log:    l,
	}
}

func (h *httpClient) postRequest(requestBody []byte) ([]byte, error) {
	var url string
	if os.Getenv("HANDLER_URL") == "" {
		url = "http://localhost:8088/api/v1/event"
	} else {
		url = os.Getenv("HANDLER_URL")
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}
	token, err := getToken()
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", token)
	resp, err := h.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func getToken() (string, error) {
	if _, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount/token"); os.IsNotExist(err) {
		return "", err
	}
	data, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func getIgnition(w http.ResponseWriter, r *http.Request) {
	var mac string
	timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
		requestIGNITIONDuration.WithLabelValues(mac).Observe(v)
	}))

	defer func() {
		timer.ObserveDuration()
	}()

	mac = getMac(r)
	if mac == "" {
		log.Printf("Not found mac in netdata, %s", " returned 204")
		http.Error(w, "not found netdata", http.StatusNoContent)
	} else {
		uuid := getUUIDbyInventory(mac)
		if uuid == "" {
			log.Printf("Not found inventory uuid for mac %s", mac)
			log.Printf("Render default ignition from configmap %s", mac)
			// read ignition-definition:
			dataIn, err := ioutil.ReadFile("/etc/ipxe-service/ignition-definition.yaml")
			if err != nil {
				log.Printf("Problem with default ignition /etc/ipxe-service/ignition-definition.yaml Error: %+v", err)
			}
			// render by butane to json
			options := common.TranslateBytesOptions{}
			dataOut, _, err := buconfig.TranslateBytes(dataIn, options)
			// return json
			fmt.Fprintf(w, string(dataOut))
		} else {
			ip := getIP(r)
			e := &event{
				UUID:    uuid,
				Reason:  "Ignition",
				Message: fmt.Sprintf("Ignition request for ip %s and  mac %s ", ip, mac),
			}
			h := newHttp()
			requestBody, _ := json.Marshal(e)
			resp, err := h.postRequest(requestBody)
			if err != nil {
				h.log.Info("can't send a request", err)
				fmt.Println(string(resp))
			}
			// TODO render specified ignition
		}
	}
}

func getMac(r *http.Request) string {
	ip := getIP(r)
	log.Printf("Clien's IP from request: %s", ip)
	mac := getMACbyNetdata(ip)
	log.Printf("Client's MAC Address from Netdata: %s", mac)
	if mac == "" {
		log.Printf("Not found client's MAC Address in Netdata for IPv4 (%s): ", ip)
	}
	return mac
}

type pasrseyaml struct {
	BaseUrl string `yaml:"base-url"`
	Kernel  string `yaml:"kernel"`
	Initrd  string `yaml:"initrd"`
}

func (c *pasrseyaml) getIpxeConf() *pasrseyaml {

	yamlFile, err := ioutil.ReadFile("/etc/ipxe-service/ipxe-default.yaml")
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}

	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}
	return c
}

func getIPXEbyK8SImage() {
	var conf dataconf
	conf.getConf()

	if err := k8simage.AddToScheme(scheme.Scheme); err != nil {
		log.Fatal("Unable to add registered types inventory to client scheme: ", err)
		os.Exit(15)
	}

	cl := createClient()

	var k8simagecrds k8simages.ImageList

	err := cl.List(context.Background(), &k8simagecrds, client.InNamespace(conf.K8SImageNS))
	if err != nil {
		log.Fatal("Failed to list K8S-Image crds inventories in namespace default: ", err)
		os.Exit(18)
	}

	//var k8simageTest string
	if len(k8simagecrds.Items) > 0 {
		log.Printf("TEST - %+v", k8simagecrds)
		//k8simageTest = k8simagecrd.Items[0].Spec.Initrd.Url
	}

	//log.Printf("TEST - %+v", k8simageTest)

}

func getChain(w http.ResponseWriter, r *http.Request) {
	var mac string
	timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
		requestIPXEDuration.WithLabelValues(mac).Observe(v)
	}))
	defer func() {
		timer.ObserveDuration()
	}()

	mac = getMac(r)
	ip := getIP(r)
	getIPXEbyK8SImage()

	if mac != "" {
		uuid := getUUIDbyInventory(mac)
		if uuid == "" {
			log.Printf("Not found client's MAC Address (%s) in Inventory: ", mac)
			log.Println("Response the default IPXE ConfigMap ...")

			var c pasrseyaml
			c.getIpxeConf()
			tmpl, err := template.ParseFiles("/etc/ipxe-service/ipxe-template")
			if err != nil {
				log.Println("Couldn't parse IPXE template file ...", err)
			}

			err = tmpl.ExecuteTemplate(w, "ipxe-template", c)
			if err != nil {
				log.Println("Couldn't execute IPXE template file ...", err)
			}

		} else {
			e := &event{
				UUID:    uuid,
				Reason:  "IPXE",
				Message: fmt.Sprintf("IPXE request for ip %s and  mac %s ", ip, mac),
			}
			h := newHttp()
			requestBody, _ := json.Marshal(e)
			resp, err := h.postRequest(requestBody)
			if err != nil {
				h.log.Info("can't send a request", err)
				fmt.Println(string(resp))
			}
			fmt.Fprintf(w, "Generate IPXE config for the client ...\n")
			// TODO render specified ipxe
		}
	}
}

func createClient() client.Client {
	cl, err := client.New(config.GetConfigOrDie(), client.Options{})
	if err != nil {
		log.Fatal("Failed to create a client:", err)
		os.Exit(19)
	}
	return cl
}

func getMachineRequest(w http.ResponseWriter, r *http.Request) {
	var conf dataconf
	conf.getConf()

	if err := mreq1.AddToScheme(scheme.Scheme); err != nil {
		log.Fatal("Unable to add registered types machine request to client scheme:", err)
		os.Exit(12)
	}

	cl := createClient()

	var mreqs mreq1.MachineRequestList
	err := cl.List(context.Background(), &mreqs, client.InNamespace(conf.MachineRequestNS))
	if err != nil {
		log.Fatal("Failed to list machine requests in namespace default:", err)
		os.Exit(14)
	}

	log.Printf("machine requests %+v", mreqs)
}

func getUUIDbyInventory(mac string) string {
	var conf dataconf
	conf.getConf()

	if err := inv.AddToScheme(scheme.Scheme); err != nil {
		log.Fatal("Unable to add registered types inventory to client scheme:", err)
		os.Exit(15)
	}

	cl := createClient()

	mac = strings.ReplaceAll(mac, ":", "")

	var inventory inv.InventoryList
	err := cl.List(context.Background(), &inventory, client.InNamespace(conf.InventoryNS), client.MatchingLabels{"machine.onmetal.de/mac-address-" + mac: ""})
	if err != nil {
		log.Fatal("Failed to list crds inventories in namespace default:", err)
		os.Exit(17)
	}

	var clientUUID string
	if len(inventory.Items) > 0 {
		clientUUID = inventory.Items[0].Spec.System.ID
	}
	log.Printf("search inventories for mac: %+v", clientUUID)

	return clientUUID
}

func getMACbyNetdata(ip string) string {
	var conf dataconf
	conf.getConf()

	if err := netdata.AddToScheme(scheme.Scheme); err != nil {
		log.Fatal("Unable to add registered types netdata to client scheme:", err)
		os.Exit(18)
	}

	cl := createClient()

	var crds netdata.NetdataList
	searchlabel := "ip-" + strings.ReplaceAll(ip, ".", "_")
	log.Printf("Search label %s", searchlabel)

	err := cl.List(context.Background(), &crds, client.InNamespace(conf.NetdataNS), client.MatchingLabels{searchlabel: ""})
	if err != nil {
		log.Fatal("Failed to list crds netdata in namespace default:", err)
		os.Exit(20)
	}

	// TODO:
	// 1. check multi CRDs
	// 2. check does an element exists (CRD)

	var clientMACAddr string
	if len(crds.Items) > 0 {
		clientMACAddr = crds.Items[0].Spec.MACAddress
	}
	return clientMACAddr
}

func getIP(r *http.Request) string {
	forwarded := r.Header.Get("X-FORWARDED-FOR")
	if forwarded != "" {
		return forwarded
	}

	clientIP, _, _ := net.SplitHostPort(r.RemoteAddr)

	log.Printf("Ip is %s", clientIP)
	return clientIP
}
