package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/Masterminds/sprig"
	buconfig "github.com/coreos/butane/config"
	"github.com/coreos/butane/config/common"
	"github.com/gorilla/mux"
	"gopkg.in/yaml.v1"

	ipam "github.com/onmetal/ipam/api/v1alpha1"
	"github.com/onmetal/metal-api-gateway/app/logger"
	inv "github.com/onmetal/metal-api/apis/inventory/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	conf                = getConf()
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

	rtr := mux.NewRouter()
	rtr.HandleFunc("/mac/{mac:[a-z0-9:]+}/boot-mac.ipxe", getMacChain).Methods("GET")
	rtr.HandleFunc("/mac/{mac:[a-z0-9:]+}/ignition/{part:[a-z0-9]+}", getMacIgnition).Methods("GET")
	rtr.HandleFunc("/", ok200).Methods("GET")

	http.Handle("/", rtr)
	http.HandleFunc("/ipxe", getChain)
	http.HandleFunc("/ignition/", getIgnition)
	http.HandleFunc("/ignition", getIgnition)
	http.HandleFunc("/-/reload", reloadApp)
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/cert", getCert)
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
	ConfigmapNS      string `yaml:"configmap-namespace"`
	IpamNS           string `yaml:"ipam-namespace"`
	MachineRequestNS string `yaml:"machine-request-namespace"`
	InventoryNS      string `yaml:"inventory-namespace"`
	ImageNS          string `yaml:"k8simage-namespace"`
}

func getInClusterNamespace() (string, error) {
	ns, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("cannot determine in-cluster namespace: %w", err)
	}
	return string(ns), nil
}

func getConf() dataconf {
	var c dataconf
	yamlFile, err := ioutil.ReadFile("/etc/ipxe-service/config.yaml")
	if err != nil {
		log.Printf("yamlFile.Get err   #%v \n Application will use current namespace for everything\n", err)
		ns, _ := getInClusterNamespace()
		if len(ns) == 0 {
			ns = "default"
		}
		c := dataconf{
			ConfigmapNS:      ns,
			IpamNS:           ns,
			MachineRequestNS: ns,
			InventoryNS:      ns,
			ImageNS:          ns}
		log.Printf("Config is #%+v ", c)
		return c
	}
	err = yaml.Unmarshal(yamlFile, &c)
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

func doesFileExist(fileName string) bool {
	_, error := os.Stat(fileName)

	// check if error is "file not exists"
	if os.IsNotExist(error) {
		return false
	} else {
		return true
	}
}

func renderDefaultIgnition(mac string, w http.ResponseWriter, partKey string) {
	var dataIn []byte
	var err error
	log.Printf("Render default Ignition from Secret , mac is %s", mac)
	if doesFileExist("/etc/ipxe-default-secret/" + partKey) {
		if len(partKey) > 0 {
			dataIn, err = ioutil.ReadFile("/etc/ipxe-default-secret/" + partKey)
		} else {
			if doesFileExist("/etc/ipxe-default-secret/ignition") {
				dataIn, err = ioutil.ReadFile("/etc/ipxe-default-secret/ignition")
			}
		}
	} else {
		log.Printf("Secret ipxe-default not contain %s", partKey)
	}
	if len(dataIn) == 0 {
		log.Printf("Render default Ignition from ConfigMap, mac is %s", mac)
		if doesFileExist("/etc/ipxe-default-cm/" + partKey) {
			if len(partKey) > 0 {
				dataIn, err = ioutil.ReadFile("/etc/ipxe-default-cm/" + partKey)
			} else {
				if doesFileExist("/etc/ipxe-default-cm/ignition") {
					dataIn, err = ioutil.ReadFile("/etc/ipxe-default-cm/ignition")
				}
			}
		}
	}
	if err != nil {
		log.Printf("Error in ignition rendering before butane: %+v", err)
	}
	resData := renderButane(dataIn)
	// return json
	fmt.Fprint(w, resData)
}

func renderButane(dataIn []byte) string {
	// render by butane to json
	options := common.TranslateBytesOptions{
		Raw:    true,
		Strict: false,
		Pretty: false,
	}
	options.NoResourceAutoCompression = true
	dataOut, _, err := buconfig.TranslateBytes(dataIn, options)
	if err != nil {
		log.Printf("\nError in ignition rendering.dataIn is : %+v\n", dataIn)
		log.Printf("Error in ignition rendering: %+v", err)
	}
	return string(dataOut)
}

func getIgnition(w http.ResponseWriter, r *http.Request) {
	// debug data
	log.Printf("\nRequested uri: %+v\n", r.URL.Path)
	log.Printf("\nHeaders: %+v\n", r.Header)

	var mac string
	var partKey string
	var userData string
	timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
		requestIGNITIONDuration.WithLabelValues(mac).Observe(v)
	}))

	defer func() {
		timer.ObserveDuration()
	}()

	if strings.LastIndex(r.URL.Path, "ignition/") >= 0 {
		info := strings.Split(r.URL.Path, "ignition/")
		partKey = info[len(info)-1]
		if len(partKey) > 0 {
			partKey = "ignition-" + partKey
			log.Printf("partKey is, %s", partKey)
		}
	}

	mac = getMac(r)
	if mac == "" {
		log.Printf("Not found MAC in IPAM ips, %s", " returned 204")
		http.Error(w, "Not found ipam ip obj", http.StatusNoContent)
	} else {
		uuid := getUUIDbyInventory(mac)
		if uuid == "" {
			log.Printf("Not found inventory UUID for MAC %s", mac)
			renderDefaultIgnition(mac, w, partKey)
		} else {
			ip := getIP(r)
			postEvent(ip, mac, uuid)
			// get secret
			secret := getSecret(uuid)
			if len(secret.Data) > 0 {
				if len(partKey) > 0 && len(secret.Data[partKey]) > 0 {
					userData = string(secret.Data[partKey])
				} else {
					if len(secret.Data["ignition"]) > 0 {
						userData = string(secret.Data["ignition"])
					}
				}

			}
			// appear here only if secret not contain data
			// get configmap if no secrets
			if len(userData) == 0 {
				cm := getConfigMap(uuid)
				if len(cm.Data) == 0 {
					log.Printf("Not found instance with UUID  %s", uuid)
					renderDefaultIgnition(mac, w, partKey)
					return
				}
				// TODO handle multiple instances
				if len(partKey) > 0 {
					userData = cm.Data[partKey]
				} else {
					userData = cm.Data["ignition"]
				}
			}
			if len(userData) == 0 {
				log.Print("UserData is empty in specific configmap and secret, going to default")
				renderDefaultIgnition(mac, w, partKey)
				return
			} else {
				log.Printf("UserData: %+v", userData)
				userDataByte := []byte(userData)
				userDataJson := renderButane(userDataByte)
				log.Printf("UserDataJson: %s", userDataJson)
				// return json
				fmt.Fprint(w, userDataJson)
				return
			}
		}
	}
}

func postEvent(ip string, mac string, uuid string) {
	e := &event{
		UUID:    uuid,
		Reason:  "Ignition",
		Message: fmt.Sprintf("Ignition request for ip %s and  mac %s ", ip, mac),
	}
	h := newHttp()
	requestBody, _ := json.Marshal(e)
	resp, err := h.postRequest(requestBody)
	if err != nil {
		h.log.Error("can't send a request", err)
		fmt.Println(string(resp))
	}
}

func getMac(r *http.Request) string {
	ip := getIP(r)
	log.Printf("Client's IP from request: %s", ip)
	mac := getMACbyIPAM(ip)
	log.Printf("Client's MAC Address from IPAM: %s", mac)
	if mac == "" {
		log.Printf("SECURITY Error Alert!")
		log.Printf(" Not found client's MAC Address in IPAM for IPv4: %s", ip)
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

func renderIpxeDefaultConfFile(w http.ResponseWriter) ([]byte, error) {
	var ipxeData []byte
	var err error
	ipxeData, err = ioutil.ReadFile("/etc/ipxe-default-secret/ipxe")
	if err != nil {
		ipxeData, err = ioutil.ReadFile("/etc/ipxe-default-cm/ipxe")
		if err != nil {
			log.Printf("Problem with default secret and configmap   #%v ", err)
			log.Fatal("This is critical!!!! default ipxe and ignition should works")
		}
	}
	fmt.Fprintf(w, string(ipxeData))
	return nil, nil
}

func renderIpxeMacConfFile(mac string) ([]byte, error) {
	var ipxeData []byte
	var err error
	ipxeData, err = ioutil.ReadFile("/etc/ipxe-default-secret/boot-mac.ipxe")
	if err != nil {
		ipxeData, err = ioutil.ReadFile("/etc/ipxe-default-cm/boot-mac.ipxe")
		if err != nil {
			log.Printf("Problem with default secret and configmap #%v ", err)
			return nil, err
		}
	}

	type Config struct {
		Mac string
	}
	cfg := Config{Mac: mac}
	tmpl, err := template.New("boot-mac.ipxe").Funcs(sprig.HermeticTxtFuncMap()).Parse(string(ipxeData))
	if err != nil {
		return nil, err
	}
	var ipxe bytes.Buffer
	err = tmpl.Execute(&ipxe, cfg)
	if err != nil {
		return nil, err
	}

	return ipxe.Bytes(), nil
}

func getCert(w http.ResponseWriter, r *http.Request) {
	ns := conf.ConfigmapNS
	cl := createClient()

	configmap := corev1.ConfigMap{}
	cmNameSpace := client.ObjectKey{
		Namespace: ns,
		Name:      "ipxe-service-server-cert",
	}

	err := cl.Get(context.Background(), cmNameSpace, &configmap)

	if err != nil {
		log.Printf("Failed to list configmap in namespace %s: %+v", ns, err)
		os.Exit(24)
	}

	log.Printf("Configmap %+v:", configmap)
	fmt.Fprintf(w, configmap.Data["ca.crt"])
}

func getMacChain(w http.ResponseWriter, r *http.Request) {
	var mac string
	timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
		requestIPXEDuration.WithLabelValues(mac).Observe(v)
	}))
	defer func() {
		timer.ObserveDuration()
	}()

	params := mux.Vars(r)
	mac = params["mac"]

	if mac != "" {
		ip, _ := getLocalLinkIpFromIPAM(mac)
		if ip != "" {
			log.Printf("Response the iPXE config file for mac %s...", mac)
			body, err := renderIpxeMacConfFile(mac)
			if err != nil {
				http.Error(w, "failed to render iPXE config for mac", http.StatusNoContent)
				return
			}
			_, err = w.Write(body)
			if err != nil {
				http.Error(w, "failed to write iPXE config for mac", http.StatusNoContent)
				return
			}
		} else {
			log.Printf("SECURITY Error Alert! Request %#v", r)
			log.Printf("Not found MAC Address in IPAM for local-link: %s", ip)
			http.Error(w, "Mac not found", http.StatusNoContent)
		}
	}
}

func getMacIgnition(w http.ResponseWriter, r *http.Request) {
	var mac string
	var part string
	timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
		requestIGNITIONDuration.WithLabelValues(mac).Observe(v)
	}))

	defer func() {
		timer.ObserveDuration()
	}()

	params := mux.Vars(r)
	mac = params["mac"]
	part = params["part"]
	if part == "" {
		http.Error(w, "no ignition part specified", http.StatusNoContent)
		return
	}
	partKey := fmt.Sprintf("ignition-%s", part)
	if mac != "" {
		ip, uuid := getLocalLinkIpFromIPAM(mac)
		if ip != "" && uuid != "" {
			var dataIn []byte
			var err error
			log.Printf("Render default Ignition from Secret , mac is %s", mac)
			file := filepath.Join("/etc/ipxe-default-secret", partKey)
			if doesFileExist(file) {
				dataIn, err = ioutil.ReadFile(file)
			}
			if len(dataIn) == 0 {
				log.Printf("Render default Ignition from ConfigMap, mac is %s", mac)
				file := filepath.Join("/etc/ipxe-default-cm", partKey)
				if doesFileExist(file) {
					dataIn, err = ioutil.ReadFile(file)
				}
			}
			if err != nil {
				log.Printf("Error in ignition rendering before butane: %s", err)
				http.Error(w, "Error in ignition reading", http.StatusNoContent)
				return
			}

			ctx := context.Background()
			cl := createClient()
			kubeconfigSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("kubeconfig-inventory-%s", uuid),
					Namespace: conf.InventoryNS,
				},
			}
			err = cl.Get(ctx, client.ObjectKeyFromObject(kubeconfigSecret), kubeconfigSecret)
			if err != nil {
				log.Printf("Error getting kubeconfig for inventory: %s", err)
				http.Error(w, "Error in ignition reading", http.StatusNoContent)
				return
			}

			kubeconfig, exists := kubeconfigSecret.Data["kubeconfig"]
			if !exists {
				log.Printf("Error getting kubeconfig data for inventory: %s", err)
				http.Error(w, "Error in ignition reading", http.StatusNoContent)
				return
			}

			type Config struct {
				Mac        string
				Kubeconfig string
			}
			cfg := Config{Mac: mac, Kubeconfig: string(kubeconfig)}
			tmpl, err := template.New("ignition").Funcs(sprig.HermeticTxtFuncMap()).Parse(string(dataIn))
			if err != nil {
				http.Error(w, "Error in ignition template creation", http.StatusNoContent)
				return
			}
			var ignition bytes.Buffer
			err = tmpl.Execute(&ignition, cfg)
			if err != nil {
				http.Error(w, "Error in ignition template rendering", http.StatusNoContent)
				return
			}
			resData := renderButane(ignition.Bytes())

			_, err = w.Write([]byte(resData))
			if err != nil {
				log.Printf("Failed to write ignition for mac: %s err: %s", mac, err)
				http.Error(w, "Failed to write ignition for mac", http.StatusNoContent)
				return
			}
		} else {
			log.Printf("SECURITY Error Alert! Request %#v", r)
			log.Printf("Not found MAC Address in IPAM for local-link: %s", ip)
			http.Error(w, "Mac not found", http.StatusNoContent)
			return
		}
	}
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

	if mac == "" {
		log.Println("Response the default IPXE config file ...")
		renderIpxeDefaultConfFile(w)
	} else {
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

			instance := getConfigMap(uuid)
			if len(instance.Data) == 0 {
				log.Printf("Not found configmap with UUID  %s", uuid)
				renderIpxeDefaultConfFile(w)
				return
			}
			userData := instance.Data["ipxe"]
			fmt.Fprintf(w, userData)
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

func getSecret(uuid string) corev1.Secret {
	cl := createClient()

	secret := corev1.Secret{}
	cmNameSpace := client.ObjectKey{
		Namespace: conf.ConfigmapNS,
		Name:      "ipxe-" + uuid,
	}

	err := cl.Get(context.Background(), cmNameSpace, &secret)

	if err != nil {
		log.Printf("Failed to list secret in namespace %s: %+v", conf.ConfigmapNS, err)
	}

	log.Printf("Secret %+v:", secret)
	return secret
}

func getConfigMap(uuid string) corev1.ConfigMap {
	cl := createClient()

	configmap := corev1.ConfigMap{}
	cmNameSpace := client.ObjectKey{
		Namespace: conf.ConfigmapNS,
		Name:      "ipxe-" + uuid,
	}

	err := cl.Get(context.Background(), cmNameSpace, &configmap)

	if err != nil {
		log.Printf("Failed to list configmap in namespace %s: %+v", conf.ConfigmapNS, err)
	}

	log.Printf("Configmap %+v:", configmap)
	return configmap
}

func getUUIDbyInventory(mac string) string {
	if err := inv.AddToScheme(scheme.Scheme); err != nil {
		log.Fatal("Unable to add registered types inventory to client scheme: ", err)
		os.Exit(15)
	}

	cl := createClient()

	mac = "machine.onmetal.de/mac-address-" + strings.ReplaceAll(mac, ":", "")
	var inventory inv.InventoryList
	err := cl.List(context.Background(), &inventory, client.InNamespace(conf.InventoryNS), client.MatchingLabels{mac: ""})
	if err != nil {
		log.Printf("Failed to list crds inventories in namespace %s: %+v", conf.InventoryNS, err)
		os.Exit(17)
	}

	var clientUUID string
	if len(inventory.Items) > 0 {
		clientUUID = inventory.Items[0].Spec.System.ID
	}
	log.Printf("Found inventories for MAC: %+v", clientUUID)

	return clientUUID
}

func IpVersion(s string) string {
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '.':
			return "ipv4"
		case ':':
			return "ipv6"
		}
	}
	return ""
}

func getMACbyIPAM(ip string) string {
	if err := ipam.AddToScheme(scheme.Scheme); err != nil {
		log.Fatal("Unable to add registered types ipam to client scheme:", err)
		os.Exit(18)
	}

	cl := createClient()

	var crds ipam.IPList

	ipLabel := strings.ReplaceAll(ip, ":", "_")

	log.Printf("Search label: ip == %s, namespace = %s", ipLabel, conf.IpamNS)

	err := cl.List(context.Background(), &crds, client.InNamespace(conf.IpamNS), client.MatchingLabels{"ip": ipLabel})
	if err != nil {
		log.Printf("Failed to list crds ipam in namespace default: %+v", err)
		return ""
	}

	// TODO:
	// 1. check multi CRDs
	// 2. check does an element exists (CRD)

	var clientMACAddr string
	if len(crds.Items) > 0 {
		for k, _ := range crds.Items[0].ObjectMeta.Labels {
			if strings.Contains(k, "mac") {
				clientMACAddr = crds.Items[0].ObjectMeta.Labels["mac"]
				break
			}
		}
	}
	return clientMACAddr
}

func getLocalLinkIpFromIPAM(mac string) (string, string) {
	if err := ipam.AddToScheme(scheme.Scheme); err != nil {
		log.Fatal("Unable to add registered types ipam to client scheme:", err)
	}

	cl := createClient()

	var crds ipam.IPList

	macLabel := strings.ReplaceAll(mac, ":", "")

	log.Printf("Search label: mac == %s, namespace = %s", macLabel, conf.IpamNS)

	err := cl.List(context.Background(), &crds, client.InNamespace(conf.IpamNS), client.MatchingLabels{"mac": macLabel})
	if err != nil {
		log.Printf("Failed to list crds ipam in namespace default: %+v", err)
		return "", ""
	}

	var localLinkIp string
	var machineUUID string
	if len(crds.Items) > 0 {
		for k, _ := range crds.Items[0].ObjectMeta.Labels {
			if strings.Contains(k, "ip") {
				localLinkIp = crds.Items[0].ObjectMeta.Labels["ip"]
				if len(crds.Items[0].ObjectMeta.OwnerReferences) > 0 {
					if strings.HasPrefix(crds.Items[0].ObjectMeta.OwnerReferences[0].Name, "m-") {
						machineUUID = strings.ReplaceAll(crds.Items[0].ObjectMeta.OwnerReferences[0].Name, "m-", "")
					}
				}
				break
			}
		}
	}
	return localLinkIp, machineUUID
}

func FullIPv6(ip net.IP) string {
	dst := make([]byte, hex.EncodedLen(len(ip)))
	_ = hex.Encode(dst, ip)
	return string(dst[0:4]) + ":" +
		string(dst[4:8]) + ":" +
		string(dst[8:12]) + ":" +
		string(dst[12:16]) + ":" +
		string(dst[16:20]) + ":" +
		string(dst[20:24]) + ":" +
		string(dst[24:28]) + ":" +
		string(dst[28:])
}

func getIP(r *http.Request) string {
	clientIP := r.Header.Get("X-FORWARDED-FOR")
	if clientIP == "" {
		clientIP, _, _ = net.SplitHostPort(r.RemoteAddr)
	}

	if IpVersion(clientIP) == "ipv6" {
		netip := net.ParseIP(clientIP)
		return FullIPv6(netip)
	}

	return clientIP
}
