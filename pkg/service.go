package pkg

import (
	"bytes"
	"fmt"
	"github.com/Masterminds/sprig"
	"github.com/gorilla/mux"
	ipamv1alpha1 "github.com/onmetal/ipam/api/v1alpha1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"text/template"
)

type IPXE struct {
	Config    Config
	K8sClient K8sClient
}

func (i IPXE) Start() {
	prometheus.MustRegister(requestIPXEDuration)
	prometheus.MustRegister(requestIGNITIONDuration)

	rtr := mux.NewRouter()
	rtr.HandleFunc("/ipxe", i.getChainDefault).Methods("GET")
	rtr.HandleFunc("/ipxe/{mac:[a-f0-9:]+}/{part:[a-z0-9-]+}", i.getChainByMac).Methods("GET")
	rtr.HandleFunc("/ignition/{mac:[a-f0-9:]+}/{uuid:[a-z0-9-]+}/{part:[a-z0-9-]+}", i.getIgnitionByMacAndUUID).Methods("GET")
	rtr.HandleFunc("/", ok200).Methods("GET")

	http.Handle("/", rtr)
	http.HandleFunc("/-/reload", i.reloadApp)
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/cert", i.getCert)
	if err := http.ListenAndServe(":8082", nil); err != nil {
		log.Fatal("Failed to start IPXE Server", err)
	}
}

func (i IPXE) getChainDefault(w http.ResponseWriter, _ *http.Request) {
	timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
		requestIPXEDuration.WithLabelValues("default").Observe(v)
	}))
	defer func() {
		timer.ObserveDuration()
	}()

	log.Println("Response the default IPXE config file ...")

	data, err := readIpxeConfFile("ipxe")
	if err != nil {
		http.Error(w, "no data found", http.StatusNoContent)
		return
	}

	_, _ = fmt.Fprintf(w, string(data))
}

func (i IPXE) getCert(w http.ResponseWriter, _ *http.Request) {
	ns := i.Config.ConfigmapNS

	configMap, err := i.K8sClient.getConfigMag(ServiceServerCert, ns)
	if err != nil {
		log.Printf("Failed to get ConfigMap %s in Namespace %s, error: %s", ServiceServerCert, ns, err)
		http.Error(w, "no data found", http.StatusNoContent)
		return
	}

	//TODO check if ca.crt exists
	_, _ = fmt.Fprintf(w, configMap.Data["ca.crt"])
}

func (i IPXE) getChainByMac(w http.ResponseWriter, r *http.Request) {
	var mac string
	timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
		requestIPXEDuration.WithLabelValues(mac).Observe(v)
	}))
	defer func() {
		timer.ObserveDuration()
	}()

	params := mux.Vars(r)
	mac = params["mac"]
	part := params["part"]
	if mac != "" {
		ip, err := i.getIP(r)
		if err != nil {
			log.Printf("Error: %s\n", err)
			http.Error(w, "Internal Error", http.StatusNoContent)
			return
		}
		ipByMac, err := i.K8sClient.getIPByMac(mac, i.Config.IpamNS)
		if err != nil {
			log.Printf("Error: %s\n", err)
			http.Error(w, "Internal Error", http.StatusNoContent)
			return
		}

		if ip != ipByMac {
			log.Printf("SECURITY Error Alert! Request %#v", r)
			log.Printf("Request IP (%s) does not match with IP (%s) from IPAM", ip, ipByMac)
			http.Error(w, "Internal Error", http.StatusNoContent)
			return
		}

		uuid, err := i.K8sClient.getInventoryUUIDByMac(mac, i.Config.InventoryNS)
		if err != nil {
			log.Printf("Error: %s\n", err)
			http.Error(w, "Internal Error", http.StatusNoContent)
			return
		}

		// if inventory uuid is empty, assume it needs to be created
		if uuid == "" {
			log.Printf("Response the %s IPXE config file for %s (%s)", part, ip, mac)
			body, err := renderIpxeMacConfFile(mac, part)
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

			//TODO(flpeter) check with Andre
			//e := &event{
			//	UUID:    uuid,
			//	Reason:  "IPXE",
			//	Message: fmt.Sprintf("IPXE request for MAC %s", uuid),
			//}
			//h := newHttp()
			//requestBody, _ := json.Marshal(e)
			//resp, err := h.postRequest(requestBody)
			//if err != nil {
			//	h.log.Info("Can't send a request", err)
			//	log.Println(string(resp))
			//}
			log.Printf("Generate IPXE config for the client ...\n")

			configMapName := "ipxe-" + uuid
			configMap, err := i.K8sClient.getConfigMag(configMapName, i.Config.ConfigmapNS)
			if err != nil {
				http.Error(w, "UUID not found", http.StatusNoContent)
				return
			}

			if len(configMap.Data) == 0 {
				log.Printf("Not found configmap with UUID  %s", uuid)
				http.Error(w, "UUID not found", http.StatusNoContent)
				return
			}
			userData, ok := configMap.Data[part]
			if ok {
				_, err = w.Write([]byte(userData))
				if err != nil {
					http.Error(w, "failed to write iPXE config for mac", http.StatusNoContent)
					return
				}
			} else {
				log.Printf("key %s not found in ConfigMap for uuid  %s", part, uuid)
				http.Error(w, "Key not found", http.StatusNoContent)
				return
			}
		}
	}
}

func (i IPXE) getIgnitionByMacAndUUID(w http.ResponseWriter, r *http.Request) {
	var mac string
	var uuid string
	var part string
	timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
		requestIGNITIONDuration.WithLabelValues(mac).Observe(v)
	}))

	defer func() {
		timer.ObserveDuration()
	}()

	params := mux.Vars(r)
	mac = params["mac"]
	if mac == "" {
		http.Error(w, "no mac specified", http.StatusNoContent)
		return
	}

	uuid = params["uuid"]
	if uuid == "" {
		http.Error(w, "no uuid specified", http.StatusNoContent)
		return
	}
	part = params["part"]
	if part == "" {
		http.Error(w, "no ignition part specified", http.StatusNoContent)
		return
	}

	ip, err := i.getIP(r)
	if err != nil {
		log.Printf("Error: %s\n", err)
		http.Error(w, "Internal Error", http.StatusNoContent)
		return
	}

	ipByMac, err := i.K8sClient.getIPByMac(mac, i.Config.IpamNS)
	if err != nil {
		log.Printf("Error: %s\n", err)
		http.Error(w, "Internal Error", http.StatusNoContent)
		return
	}

	if ip != ipByMac {
		log.Printf("SECURITY Error Alert! Request %#v", r)
		log.Printf("Request IP (%s) does not match with IP (%s) from IPAM", ip, ipByMac)
		http.Error(w, "Internal Error", http.StatusNoContent)
		return
	}

	uuidByMac, err := i.K8sClient.getInventoryUUIDByMac(mac, i.Config.InventoryNS)
	if err != nil {
		log.Printf("Error: %s\n", err)
		http.Error(w, "Internal Error", http.StatusNoContent)
		return
	}

	partKey := fmt.Sprintf("ignition-%s", part)
	// if inventory uuid is empty, assume it needs to be created
	if uuidByMac == "" {
		var dataIn []byte
		log.Printf("Render default Ignition part %s from Secret, mac is %s and uuid is %s\n", partKey, mac, uuid)
		file := filepath.Join(DefaultSecretPath, partKey)
		if doesFileExist(file) {
			dataIn, err = os.ReadFile(file)
		}
		if len(dataIn) == 0 {
			log.Printf("Render default Ignition part %s from ConfigMap, mac is %s and uuid is %s\n", partKey, mac, uuid)
			file = filepath.Join(DefaultConfigMapPath, partKey)
			if doesFileExist(file) {
				dataIn, err = os.ReadFile(file)
			}
		}
		if err != nil {
			log.Printf("Error in ignition rendering before butane: %s", err)
			http.Error(w, "Error in ignition reading", http.StatusNoContent)
			return
		}

		kubeconfigSecretName := fmt.Sprintf("kubeconfig-inventory-%s", uuid)
		kubeconfigSecret, err := i.K8sClient.getSecret(kubeconfigSecretName, i.Config.InventoryNS)
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
			UUID       string
			Kubeconfig string
			Hostname   string
		}
		cfg := Config{Mac: mac, UUID: uuid, Kubeconfig: string(kubeconfig), Hostname: uuid}
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
		// uuid from mac and provided uuid are matching, so we need to deliver the ipxe-uuid config
		if uuidByMac == uuid {
			var userData string
			secretName := "ipxe-" + uuid
			secret, err := i.K8sClient.getSecret(secretName, i.Config.ConfigmapNS)
			if err != nil {
				http.Error(w, "no data found", http.StatusNoContent)
				return
			}

			if len(secret.Data) > 0 {
				if len(partKey) > 0 && len(secret.Data[partKey]) > 0 {
					userData = string(secret.Data[partKey])
				}
			}

			if len(userData) == 0 {
				log.Print("UserData is empty in specific secret")
				http.Error(w, "no data found", http.StatusNoContent)
				return
			} else {
				log.Printf("UserData: %+v", userData)
				userDataByte := []byte(userData)
				userDataJson := renderButane(userDataByte)
				log.Printf("UserDataJson: %s", userDataJson)

				_, err := w.Write([]byte(userDataJson))
				if err != nil {
					log.Printf("Failed to write ignition for uuid: %s err: %s", mac, err)
					http.Error(w, "Failed to write ignition for uuid", http.StatusNoContent)
					return
				}
				return
			}
		} else {
			log.Printf("SECURITY Error Alert! Request %#v", r)
			log.Printf("UUID (%s) by mac does not match with provided UUID (%s) from inventory", uuidByMac, uuid)
			http.Error(w, "Internal Error", http.StatusNoContent)
			return
		}
	}
}

func (i IPXE) getIP(r *http.Request) (string, error) {
	var clientIP string
	var err error
	if i.Config.DisableForwardHeader {
		clientIP, _, err = net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			return "", err
		}
	} else {
		clientIP = r.Header.Get("X-FORWARDED-FOR")
		if clientIP == "" {
			clientIP, _, err = net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				return "", err
			}
		}
	}

	if IpVersion(clientIP) == "ipv6" {
		ip, err := ipamv1alpha1.IPAddrFromString(clientIP)
		if err != nil {
			return "", err
		}
		return ip.String(), nil
	}

	return clientIP, nil
}

func (i IPXE) reloadApp(w http.ResponseWriter, r *http.Request) {
	ip, err := i.getIP(r)
	if err != nil {
		_, _ = w.Write([]byte(fmt.Sprintf("error, %s", err)))
		return
	}

	if ip == "127.0.0.1" {
		log.Print("Reload server because changed configmap")
		_, _ = w.Write([]byte("reloaded"))
		go os.Exit(0)
	} else {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("forbidden"))
	}
}

func ok200(w http.ResponseWriter, _ *http.Request) {
	_, _ = w.Write([]byte("ok\n"))
}
