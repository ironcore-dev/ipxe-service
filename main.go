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
	"gopkg.in/yaml.v1"

	k8simages "github.com/onmetal/k8s-image/api/v1alpha1"
	inv "github.com/onmetal/k8s-inventory/api/v1alpha1"
	minst "github.com/onmetal/k8s-machine-instance/api/v1"
	mreq1 "github.com/onmetal/k8s-machine-requests/api/v1alpha1"
	"github.com/onmetal/machine-operator/app/machine-event-handler/logger"
	netdata "github.com/onmetal/netdata/api/v1"

	"k8s.io/apimachinery/pkg/types"
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

    fmt.Println("iPXE is running ...")

	http.HandleFunc("/", ok200)
	http.HandleFunc("/ipxe", getChain)
	http.HandleFunc("/ignition", getIgnition)
	http.HandleFunc("/-/reload", reloadApp)
	http.Handle("/metrics", promhttp.Handler())
	if err := http.ListenAndServe(":8082", nil); err != nil {
		log.Fatal("Failed to start IPXE Server", err)
		os.Exit(11)
	}
}

func reloadApp(w http.ResponseWriter, r *http.Request) {
	ip := getIP(r)
	if ip == "127.0.0.1" {
		log.Print("Reload server because changed configmap")
		w.Write([]byte("reloaded"))
		go os.Exit(0)
	} else {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("forbidden"))
	}
}

func ok200(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "ok\n")
}

type dataconf struct {
	InstanceNS       string `yaml:"instance-namespace"`
	NetdataNS        string `yaml:"netdata-namespace"`
	MachineRequestNS string `yaml:"machine-request-namespace"`
	InventoryNS      string `yaml:"inventory-namespace"`
	ImageNS          string `yaml:"k8simage-namespace"`
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

func renderDefaultIgnition(mac string, w http.ResponseWriter) {
	log.Printf("Render default Ignition from ConfigMap, mac is %s", mac)
	// read ignition-definition:
	dataIn, err := ioutil.ReadFile("/etc/ipxe-service/ignition-definition.yaml")
	if err != nil {
		log.Printf("Problem with default ignition /etc/ipxe-service/ignition-definition.yaml Error: %+v", err)
	}
	// render by butane to json
	options := common.TranslateBytesOptions{
		Raw:    true,
		Strict: false,
		Pretty: false,
	}
	options.NoResourceAutoCompression = true
	dataOut, _, err := buconfig.TranslateBytes(dataIn, options)
	if err != nil {
		log.Printf("Error in ignition rendering: %+v", err)
	}
	// return json
	fmt.Fprint(w, string(dataOut))
}

func getIgnition(w http.ResponseWriter, r *http.Request) {
	log.Print("TODO make autorestart if configmap was changed")
	var mac string
	timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
		requestIGNITIONDuration.WithLabelValues(mac).Observe(v)
	}))

	defer func() {
		timer.ObserveDuration()
	}()

	mac = getMac(r)
	if mac == "" {
		log.Printf("Not found MAC in Netdata, %s", " returned 204")
		http.Error(w, "Not found netdata", http.StatusNoContent)
	} else {
		uuid := getUUIDbyInventory(mac)
		if uuid == "" {
			log.Printf("Not found inventory UUID for MAC %s", mac)
			renderDefaultIgnition(mac, w)
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

			var (
				instances minst.MachineInstanceList
				mreqs     mreq1.MachineList
			)
			instances = getMachineInstance(uuid)
			if len(instances.Items) == 0 {
				log.Printf("Not found instance with UUID  %s", uuid)
				renderDefaultIgnition(mac, w)
				return
			}
			// TODO handle multiple instances
			machineReqID := instances.Items[0].Spec.MachineRequestID
			mreqs = getMachineRequest(machineReqID)
			if len(mreqs.Items) == 0 {
				log.Printf("Not found machinerequest with ID  %s", machineReqID)
				renderDefaultIgnition(mac, w)
				return
			}
			userData := mreqs.Items[0].Spec.UserData
			log.Printf("UserData: %+v", userData)
			// render by butane to json
			options := common.TranslateBytesOptions{}
			dataOut, _, err := buconfig.TranslateBytes([]byte(userData), options)
			// return json
			fmt.Fprintf(w, string(dataOut))
		}
	}
}

func getMac(r *http.Request) string {
	ip := getIP(r)
	log.Printf("Clien's IP from request: %s", ip)
	mac := getMACbyNetdata(ip)
	log.Printf("Client's MAC Address from Netdata: %s", mac)
	if mac == "" {
		log.Printf("Not found client's MAC Address in Netdata for IPv4: %s", ip)
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

func getIPXEbyK8SImage(w http.ResponseWriter, imageName string) {
	var conf dataconf
	conf.getConf()

	if err := k8simages.AddToScheme(scheme.Scheme); err != nil {
		log.Fatal("Unable to add registered types inventory to client scheme: ", err)
		os.Exit(15)
	}

	cl := createClient()
	k8simagecrd := k8simages.Image{}

	imgNamespacedName := types.NamespacedName{
		Namespace: conf.ImageNS,
		Name:      imageName,
	}

	err := cl.Get(context.Background(), imgNamespacedName, &k8simagecrd)
	if err != nil {
		log.Printf("Failed to get K8S-Image crd in namespace %s: %+v", conf.ImageNS, err)
		os.Exit(18)
	}

	k8simageKernel := k8simagecrd.Spec.Source[0].URL
	k8simageInitrd := k8simagecrd.Spec.Source[1].URL
	k8simageRootfs := k8simagecrd.Spec.Source[2].URL
	fmt.Fprintf(w, "#!ipxe\n\nset base-url http://45.86.152.1/ipxe\nkernel %+v\ninitrd %+v\nrootfs %+v\nboot", k8simageKernel, k8simageInitrd, k8simageRootfs)
}

func renderIpxeDefaultConfFile(w http.ResponseWriter) ([]byte, error) {
	var c pasrseyaml
	c.getIpxeConf()
	tmpl, err := template.ParseFiles("/etc/ipxe-service/ipxe-template")
	if err != nil {
		log.Println("Couldn't parse IPXE template file: ", err)
	}

	err = tmpl.ExecuteTemplate(w, "ipxe-template", c)
	if err != nil {
		log.Println("Couldn't execute IPXE template file: ", err)
	}

	return nil, err
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

	if mac != "" {
		uuid := getUUIDbyInventory(mac)
		if uuid == "" {
			log.Printf("Not found client's MAC Address (%s) in Inventory: ", mac)
			log.Println("Response the default IPXE config file ...")

			renderIpxeDefaultConfFile(w)

		} else {
			e := &event{
				UUID:    uuid,
				Reason:  "IPXE",
				Message: fmt.Sprintf("IPXE request for IP %s and  MAC %s ", ip, mac),
			}
			h := newHttp()
			requestBody, _ := json.Marshal(e)
			resp, err := h.postRequest(requestBody)
			if err != nil {
				h.log.Info("Can't send a request", err)
				log.Println(string(resp))
			}
			log.Printf("Generate IPXE config for the client ...\n")
			var (
				instances minst.MachineInstanceList
				mreqs     mreq1.MachineList
			)
			instances = getMachineInstance(uuid)
			if len(instances.Items) == 0 {
				log.Printf("Not found instance with UUID  %s", uuid)
				renderIpxeDefaultConfFile(w)
				return
			}
			// TODO handle multiple instances
			machineReqID := instances.Items[0].Spec.MachineRequestID
			mreqs = getMachineRequest(machineReqID)
			if len(mreqs.Items) == 0 {
				log.Printf("Not found machinerequest with ID  %s", machineReqID)
				renderIpxeDefaultConfFile(w)
				return
			}
			imageName := mreqs.Items[0].Spec.Image.Name
			getIPXEbyK8SImage(w, imageName)
		}
	}
}

func createClient() client.Client {
	cl, err := client.New(config.GetConfigOrDie(), client.Options{})
	if err != nil {
		log.Fatal("Failed to create a client: ", err)
		os.Exit(19)
	}
	return cl
}

func getMachineRequest(uuid string) mreq1.MachineList {
	var conf dataconf
	conf.getConf()

	if err := mreq1.AddToScheme(scheme.Scheme); err != nil {
		log.Fatal("Unable to add registered types machine request to client scheme: ", err)
		os.Exit(12)
	}

	cl := createClient()
	var mreqs mreq1.MachineList
	err := cl.List(context.Background(), &mreqs, client.InNamespace(conf.MachineRequestNS), client.MatchingLabels{"id": uuid})
	if err != nil {
		log.Fatal("Failed to list machine requests in namespace default: ", err)
		os.Exit(14)
	}

	log.Printf("Machine requests %+v:", mreqs)
	return mreqs
}

func getMachineInstance(uuid string) minst.MachineInstanceList {
	var conf dataconf
	conf.getConf()

	if err := minst.AddToScheme(scheme.Scheme); err != nil {
		log.Fatal("Unable to add registered types machine-instance to client scheme: ", err)
		os.Exit(22)
	}

	cl := createClient()
	var minstes minst.MachineInstanceList
	err := cl.List(context.Background(), &minstes, client.InNamespace(conf.InstanceNS), client.MatchingLabels{"id": uuid})
	if err != nil {
		log.Printf("Failed to list machine-instance in namespace %s: %+v", conf.InstanceNS, err)
		os.Exit(24)
	}

	log.Printf("Machine-instances %+v:", minstes)
	return minstes
}

func getUUIDbyInventory(mac string) string {
	var conf dataconf
	conf.getConf()

	if err := inv.AddToScheme(scheme.Scheme); err != nil {
		log.Fatal("Unable to add registered types inventory to client scheme: ", err)
		os.Exit(15)
	}

	cl := createClient()

	mac = strings.ReplaceAll(mac, ":", "")
	var inventory inv.InventoryList
	err := cl.List(context.Background(), &inventory, client.InNamespace(conf.InventoryNS), client.MatchingLabels{mac: ""})
	if err != nil {
		log.Fatal("Failed to list crds inventories in namespace default:", err)
		os.Exit(17)
	}

	var clientUUID string
	if len(inventory.Items) > 0 {
		clientUUID = inventory.Items[0].Spec.System.ID
	}
	log.Printf("Search inventories for MAC: %+v", clientUUID)

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
	searchLabel := netdata.LabelForIP(ip)
	log.Printf("Search label: %s", searchLabel)

	err := cl.List(context.Background(), &crds, client.InNamespace(conf.NetdataNS), client.MatchingLabels{searchLabel: ""})
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

	return clientIP
}
