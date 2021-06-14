package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	buconfig "github.com/coreos/butane/config"
	"github.com/coreos/butane/config/common"
	inv "github.com/onmetal/k8s-inventory/api/v1alpha1"
	mreq1 "github.com/onmetal/k8s-machine-requests/api/v1alpha1"
	netdata "github.com/onmetal/netdata/api/v1"
	"gopkg.in/yaml.v1"

	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func main() {
	http.HandleFunc("/", ok200)
	http.HandleFunc("/ipxe", getChain)
	http.HandleFunc("/ignition", getIgnition)
	if err := http.ListenAndServe(":8082", nil); err != nil {
		log.Fatal("Failed to start IPXE Server", err)
		os.Exit(11)
	}
}

func ok200(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "ok")
}

type dataconf struct {
	NetdataNS        string `yaml:"netdata-namespace"`
	MachineRequestNS string `yaml:"machine-request-namespace"`
	InventoryNS      string `yaml:"inventory-namespace"`
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

func getIgnition(w http.ResponseWriter, r *http.Request) {
	mac := getMac(r)
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

func getChain(w http.ResponseWriter, r *http.Request) {
	mac := getMac(r)
	if mac != "" {
		uuid := getUUIDbyInventory(mac)
		if uuid == "" {
			log.Printf("Not found client's MAC Address (%s) in Inventory: ", mac)
			log.Println("Response the default IPXE ConfigMap ...")
			getDefaultIPXE, err := ioutil.ReadFile("/etc/ipxe-service/ipxe-default")
			if err != nil {
				log.Fatal("Unable to read the default ipxe config file", err)
				os.Exit(23)
			}
			fmt.Fprintf(w, string(getDefaultIPXE))
		} else {
			fmt.Fprintf(w, "Generate IPXE config for the client ...\n")
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
	err := cl.List(context.Background(), &crds, client.InNamespace(conf.NetdataNS), client.MatchingLabels{"ipv4": ip})
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
	} else {
		return clientMACAddr
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
