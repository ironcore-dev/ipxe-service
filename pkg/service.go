// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package pkg

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	corev1 "k8s.io/api/core/v1"
)

type IPXE struct {
	Config    Config
	K8sClient K8sClient
}

func (i IPXE) Start() {
	prometheus.MustRegister(requestIPXEDuration)
	prometheus.MustRegister(requestIGNITIONDuration)

	rtr := i.getRouter()
	http.Handle("/", rtr)
	http.HandleFunc("/-/reload", i.reloadApp)
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/cert", i.getCert)
	if err := http.ListenAndServe(":8082", nil); err != nil {
		log.Fatal("Failed to start IPXE Server", err)
	}
}
func (i IPXE) getRouter() *mux.Router {
	rtr := mux.NewRouter()
	rtr.HandleFunc("/ipxe", i.getChainDefault).Methods("GET")
	rtr.HandleFunc("/ipxe/{uuid:[a-f0-9-]+}/{part:[a-z0-9-]+}", i.getChainByUUID).Methods("GET")
	rtr.HandleFunc("/ignition/{uuid:[a-z0-9-]+}/{part:[a-z0-9-]+}", i.getIgnitionByUUID).Methods("GET")
	rtr.HandleFunc("/", ok200).Methods("GET")

	return rtr
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
		http.Error(w, "no data found", http.StatusInternalServerError)
		return
	}

	_, _ = fmt.Fprint(w, string(data))
}

func (i IPXE) getCert(w http.ResponseWriter, _ *http.Request) {
	ns := i.Config.ConfigmapNS

	configMap, err := i.K8sClient.getConfigMag(ServiceServerCert, ns)
	if err != nil {
		log.Printf("Failed to get ConfigMap %s in Namespace %s, error: %s", ServiceServerCert, ns, err)
		http.Error(w, "no data found", http.StatusInternalServerError)
		return
	}

	//TODO check if ca.crt exists
	_, _ = fmt.Fprintf(w, configMap.Data["ca.crt"])
}

func (i IPXE) getChainByUUID(w http.ResponseWriter, r *http.Request) {
	var uuid string
	timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
		requestIPXEDuration.WithLabelValues(uuid).Observe(v)
	}))
	defer func() {
		timer.ObserveDuration()
	}()

	params := mux.Vars(r)
	uuid = params["uuid"]
	part := params["part"]
	if uuid != "" {
		clientIP, err := i.getIP(r)
		if err != nil {
			log.Printf("Error: %s\n", err)
			http.Error(w, "Internal Error", http.StatusInternalServerError)
			return
		}
		mac, err := i.K8sClient.getMacFromIP(clientIP, i.Config.IpamNS)
		if err != nil {
			log.Printf("Error: %s\n", err)
			http.Error(w, "Internal Error", http.StatusInternalServerError)
			return
		}

		inventory, err := i.K8sClient.getInventory(uuid, i.Config.InventoryNS)
		if err != nil {
			log.Printf("Error: %s\n", err)
			http.Error(w, "Internal Error", http.StatusInternalServerError)
			return
		}

		// if inventory uuid is empty, assume it needs to be created
		if inventory.Spec.System == nil || inventory.Spec.System.ID == "" {
			log.Printf("Response the %s IPXE config file for %s (%s)", part, clientIP, uuid)
			body, err := readIpxeConfFile(part)
			if err != nil {
				http.Error(w, "failed to render iPXE config for mac", http.StatusInternalServerError)
				return
			}
			_, err = w.Write(body)
			if err != nil {
				http.Error(w, "failed to write iPXE config for mac", http.StatusInternalServerError)
				return
			}
		} else {
			err := checkInventoryMac(inventory, mac)
			if err != nil {
				i.K8sClient.EventRecorder.Eventf(inventory, corev1.EventTypeWarning,
					"Denied", "Denied client %s because mac '%s' does not match for inventory", clientIP, mac)
				log.Printf("SECURITY Error Alert! Request %#v", r)
				log.Printf("MAC (%s) does not match with provided UUID (%s) from inventory", mac, uuid)
				http.Error(w, "Internal Error", http.StatusInternalServerError)
				return
			}

			log.Printf("Generate iPXE config for the client %s\n", clientIP)
			i.K8sClient.EventRecorder.Eventf(inventory, corev1.EventTypeNormal, "Generate",
				"Generate iPXE config for client %s", clientIP)

			configMapName := "ipxe-" + uuid
			configMap, err := i.K8sClient.getConfigMag(configMapName, i.Config.ConfigmapNS)
			if err != nil {
				http.Error(w, "UUID not found", http.StatusInternalServerError)
				return
			}

			if len(configMap.Data) == 0 {
				log.Printf("Not found configmap with UUID  %s", uuid)
				http.Error(w, "UUID not found", http.StatusInternalServerError)
				return
			}
			userData, ok := configMap.Data[part]
			if ok {
				_, err = w.Write([]byte(userData))
				if err != nil {
					http.Error(w, "failed to write iPXE config for mac", http.StatusInternalServerError)
					return
				}
			} else {
				log.Printf("key %s not found in ConfigMap for uuid  %s", part, uuid)
				http.Error(w, "Key not found", http.StatusInternalServerError)
				return
			}
		}
	}
}

func (i IPXE) getIgnitionByUUID(w http.ResponseWriter, r *http.Request) {
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
	uuid = params["uuid"]
	if uuid == "" {
		http.Error(w, "no uuid specified", http.StatusInternalServerError)
		return
	}
	part = params["part"]
	if part == "" {
		http.Error(w, "no ignition part specified", http.StatusInternalServerError)
		return
	}

	clientIP, err := i.getIP(r)
	if err != nil {
		log.Printf("Error: %s\n", err)
		http.Error(w, "Internal Error", http.StatusInternalServerError)
		return
	}

	mac, err = i.K8sClient.getMacFromIP(clientIP, i.Config.IpamNS)
	if err != nil {
		log.Printf("Error: %s\n", err)
		http.Error(w, "Internal Error", http.StatusInternalServerError)
		return
	}

	inventory, err := i.K8sClient.getInventory(uuid, i.Config.InventoryNS)
	if err != nil {
		log.Printf("Error: %s\n", err)
		http.Error(w, "Internal Error", http.StatusInternalServerError)
		return
	}

	partKey := fmt.Sprintf("ignition-%s", part)
	// if inventory uuid is empty, assume it needs to be created
	if inventory.Spec.System == nil || inventory.Spec.System.ID == "" {
		var dataIn []byte
		log.Printf("Render default Ignition part %s from Secret, mac is %s and uuid is %s\n", partKey, mac, uuid)
		defaultSecretPath := os.Getenv("IPXE_DEFAULT_SECRET_PATH")
		if defaultSecretPath == "" {
			defaultSecretPath = DefaultSecretPath
		}
		file := filepath.Join(defaultSecretPath, partKey)
		if doesFileExist(file) {
			dataIn, err = os.ReadFile(file)
		}
		if len(dataIn) == 0 {
			log.Printf("Render default Ignition part %s from ConfigMap, mac is %s and uuid is %s\n", partKey, mac, uuid)
			defaultConfigMapPath := os.Getenv("IPXE_DEFAULT_CONFIGMAP_PATH")
			if defaultConfigMapPath == "" {
				defaultConfigMapPath = DefaultConfigMapPath
			}
			file = filepath.Join(defaultConfigMapPath, partKey)
			if doesFileExist(file) {
				dataIn, err = os.ReadFile(file)
			}
		}
		if err != nil {
			log.Printf("Error in ignition rendering before butane: %s", err)
			http.Error(w, "Error in ignition reading", http.StatusInternalServerError)
			return
		}

		kubeconfigSecretName := fmt.Sprintf("kubeconfig-inventory-%s", uuid)
		kubeconfigSecret, err := i.K8sClient.getSecret(kubeconfigSecretName, i.Config.InventoryNS)
		if err != nil {
			log.Printf("Error getting kubeconfig for inventory: %s", err)
			http.Error(w, "Error in ignition reading", http.StatusInternalServerError)
			return
		}

		kubeconfig, exists := kubeconfigSecret.Data["kubeconfig"]
		if !exists {
			log.Printf("Error getting kubeconfig data for inventory: %s", err)
			http.Error(w, "Error in ignition reading", http.StatusInternalServerError)
			return
		}

		type Config struct {
			UUID       string
			Kubeconfig string
			Hostname   string
		}
		cfg := Config{UUID: uuid, Kubeconfig: string(kubeconfig), Hostname: uuid}
		tmpl, err := template.New("ignition").Funcs(sprig.HermeticTxtFuncMap()).Parse(string(dataIn))
		if err != nil {
			http.Error(w, "Error in ignition template creation", http.StatusInternalServerError)
			return
		}
		var ignition bytes.Buffer
		err = tmpl.Execute(&ignition, cfg)
		if err != nil {
			http.Error(w, "Error in ignition template rendering", http.StatusInternalServerError)
			return
		}
		resData, err := renderButane(ignition.Bytes())
		if err != nil {
			http.Error(w, "Error in render butane", http.StatusInternalServerError)
			return
		}

		_, err = w.Write([]byte(resData))
		if err != nil {
			log.Printf("Failed to write ignition for mac: %s err: %s", mac, err)
			http.Error(w, "Failed to write ignition for mac", http.StatusInternalServerError)
			return
		}
	} else {
		err = checkInventoryMac(inventory, mac)
		if err != nil {
			i.K8sClient.EventRecorder.Eventf(inventory, corev1.EventTypeWarning,
				"Denied", "Denied client %s because mac '%s' does not match for inventory", clientIP, mac)
			log.Printf("SECURITY Error Alert! Request %#v", r)
			log.Printf("MAC (%s) does not match with provided UUID (%s) from inventory", mac, uuid)
			http.Error(w, "Internal Error", http.StatusInternalServerError)
			return
		}

		var userData string
		secretName := "ipxe-" + uuid
		secret, err := i.K8sClient.getSecret(secretName, i.Config.ConfigmapNS)
		if err != nil {
			http.Error(w, "no data found", http.StatusInternalServerError)
			return
		}

		if len(secret.Data) > 0 {
			if len(partKey) > 0 && len(secret.Data[partKey]) > 0 {
				userData = string(secret.Data[partKey])
			}
		}

		if len(userData) == 0 {
			log.Print("UserData is empty in specific secret")
			http.Error(w, "no data found", http.StatusInternalServerError)
			return
		} else {
			log.Printf("Render ignition %s for client %s", secretName, clientIP)
			i.K8sClient.EventRecorder.Eventf(inventory, corev1.EventTypeNormal, "Ignition",
				"Render ignition %s for client %s", secretName, clientIP)

			//TODO add as debug log
			//log.Printf("UserData: %+v", userData)
			userDataByte := []byte(userData)
			userDataJson, err := renderButane(userDataByte)
			if err != nil {
				http.Error(w, "Error in render butane", http.StatusInternalServerError)
				return
			}
			//TODO add as debug log
			//log.Printf("UserDataJson: %s", userDataJson)

			_, err = w.Write([]byte(userDataJson))
			if err != nil {
				log.Printf("Failed to write ignition for uuid: %s err: %s", mac, err)
				http.Error(w, "Failed to write ignition for uuid", http.StatusInternalServerError)
				return
			}
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
	_, err := w.Write([]byte("ok\n"))
	if err != nil {
		http.Error(w, "Failed to return 200 (OK)", http.StatusInternalServerError)
	}
}
