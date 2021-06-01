package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	inv "k8s-inventory/api/v1alpha1"
	mreq1 "k8s-machine-requests/api/v1alpha1"
	"k8s.io/client-go/kubernetes/scheme"
	"log"
	"net"
	"net/http"
	netdata "netdata/api/v1"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"strings"
)

func main() {
	http.HandleFunc("/ipxe", getChain)
	if err := http.ListenAndServe(":8082", nil); err != nil {
		log.Fatal("Failed to start IPXE Server")
		os.Exit(11)
	}
}

func getChain(w http.ResponseWriter, r *http.Request) {

	ip := getIP(r)
	fmt.Println(ip)
	mac := getNetdata(ip)
	fmt.Println(mac)
	uuid := getInventory(mac)
	fmt.Println(uuid)

	getConf(ip)

	w.Header().Add("Content-Type", "application/json")
	resp, _ := json.Marshal(map[string]string{
		"IP":   ip,
		"MAC":  mac,
		"UUID": uuid,
	})
	w.Write(resp)
}

func createClient() client.Client {
	cl, err := client.New(config.GetConfigOrDie(), client.Options{})
	if err != nil {
		log.Fatal("Failed to create a client", err)
		os.Exit(19)
	}
	return cl
}

func getMachineRequest(w http.ResponseWriter, r *http.Request) {
	fmt.Println("test1")

	if err := mreq1.AddToScheme(scheme.Scheme); err != nil {
		log.Fatal("unable to add registered types machine request to client scheme")
		os.Exit(12)
	}
	fmt.Println("test1")

	cl := createClient()

	var mreqs mreq1.MachineRequestList
	err := cl.List(context.Background(), &mreqs, client.InNamespace("default"))
	if err != nil {
		log.Fatal("Failed to list machine requests in namespace default")
		os.Exit(14)
	}

	fmt.Printf("machine requests %+v", mreqs)
}

func getInventory(mac string) string {
	if err := inv.AddToScheme(scheme.Scheme); err != nil {
		log.Fatal("unable to add registered types inventory to client scheme", err)
		os.Exit(15)
	}

	cl := createClient()

	mac = strings.ReplaceAll(mac, ":", "")

	var inventory inv.InventoryList
	err := cl.List(context.Background(), &inventory, client.InNamespace("default"), client.MatchingLabels{"macAddr": mac})
	if err != nil {
		log.Fatal("Failed to list crds netdata in namespace default", err)
		os.Exit(17)
	}

	clientUUID := inventory.Items[0].Spec.System.ID
	return clientUUID
}

func getNetdata(ip string) string {
	if err := netdata.AddToScheme(scheme.Scheme); err != nil {
		log.Fatal("Unable to add registered types netdata to client scheme", err)
		os.Exit(18)
	}

	cl := createClient()

	var crds netdata.NetdataList
	err := cl.List(context.Background(), &crds, client.InNamespace("default"), client.MatchingLabels{"ipv4": ip})
	if err != nil {
		log.Fatal("Failed to list crds netdata in namespace default", err)
		os.Exit(20)
	}

	// TODO:
	// 1. check multi CRDs
	// 2. check does an element exists (CRD)

	var clientMACAddr string
	if len(crds.Items) > 0 {
		clientMACAddr = crds.Items[0].Spec.MACAddress
	} else {
		log.Fatalf("not found netdata for ipv4 %s", ip)
		os.Exit(33)
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

func getConf(ip string) {

	err := os.Mkdir("ip"+ip, 0755)
	if err != nil {
		log.Fatalf("Unable to create a client's ipxe directory:", err)
		os.Exit(21)
	}

	file, err := os.Create("./ip" + ip + "/boot.ipxe")
	if err != nil {
		log.Fatalf("Unable to create a client's ipxe file:", err)
		os.Exit(22)
	}

	defer file.Close()

	defaultIPXEContent, err := ioutil.ReadFile("etc/ipxe/ipxe-default-config.yaml")
	if err != nil {
		log.Fatalf("Unable to read the default ipxe config file:", err)
		os.Exit(23)
	}

	file.Write(defaultIPXEContent)
}
