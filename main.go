package main

import (
	"context"
	"fmt"
	inv "k8s-inventory/api/v1alpha1"
	"k8s.io/client-go/kubernetes/scheme"
	//	mreq1 "k8s-machine-requests/api/v1alpha1"
	"log"
	"net"
	"net/http"
	netdata "netdata/api/v1"
	"os"
	"regexp"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func main() {
	http.HandleFunc("/ipxe", getChain)
	if err := http.ListenAndServe(":8082", nil); err != nil {
		fmt.Println("Failed to start IPXE Server")
		os.Exit(1)
	}
}

func getChain(w http.ResponseWriter, r *http.Request) {

	ip := getIP(r)
	fmt.Println(ip)
	mac := getNetdata(ip)
	fmt.Println(mac)
	uuid := getInventory(mac)
	fmt.Println(uuid)
	//w.Write(uuid)

}

/*
func getMachineRequest(w http.ResponseWriter, r *http.Request) {
	fmt.Println("test1")

	if err := mreq1.AddToScheme(scheme.Scheme); err != nil {
		fmt.Println("unable to add registered types machine request to client scheme")
		os.Exit(1)
	}
	fmt.Println("test1")

	cl, err := client.New(config.GetConfigOrDie(), client.Options{})
	if err != nil {
		fmt.Println("Failed to create a client")
		os.Exit(1)
	}

	var mreqs mreq1.MachineRequestList
	err = cl.List(context.Background(), &mreqs, client.InNamespace("default"))
	if err != nil {
		fmt.Println("Failed to list machine requests in namespace default")
		os.Exit(1)
	}

	fmt.Printf("machine requests %+v", mreqs)
}
*/

func getInventory(mac string) string {
	if err := inv.AddToScheme(scheme.Scheme); err != nil {
		fmt.Println("unable to add registered types inventory to client scheme", err)
		os.Exit(1)
	}

	cl, err := client.New(config.GetConfigOrDie(), client.Options{})
	if err != nil {
		fmt.Println("Failed to create a client")
		os.Exit(1)
	}

	re, err := regexp.Compile(`[:]`)
	if err != nil {
		log.Fatal(err)
	}

	mac = re.ReplaceAllString(mac, "")

	var inventory inv.InventoryList
	err = cl.List(context.Background(), &inventory, client.InNamespace("default"), client.MatchingLabels{"macAddr": mac})
	if err != nil {
		fmt.Println("Failed to list crds netdata in namespace default", err)
		os.Exit(1)
	}

	clientUUID := inventory.Items[0].Spec.System.ID
	return clientUUID
}

func getNetdata(ip string) string {
	if err := netdata.AddToScheme(scheme.Scheme); err != nil {
		fmt.Println("Unable to add registered types netdata to client scheme", err)
		os.Exit(1)
	}

	cl, err := client.New(config.GetConfigOrDie(), client.Options{})
	if err != nil {
		fmt.Println("Failed to create a client", err)
		os.Exit(1)
	}

	var crds netdata.NetdataList
	err = cl.List(context.Background(), &crds, client.InNamespace("default"), client.MatchingLabels{"ipv4": ip})
	if err != nil {
		fmt.Println("Failed to list crds netdata in namespace default", err)
		os.Exit(1)
	}

	// TODO:
	// 1. check multi CRDs
	// 2. check does an element exists (CRD)

	clientMACAddr := crds.Items[0].Spec.MACAddress
	return clientMACAddr
}

func getIP(r *http.Request) string {
	forwarded := r.Header.Get("X-FORWARDED-FOR")
	if forwarded != "" {
		return forwarded
	}

	r.RemoteAddr, _, _ = net.SplitHostPort(r.RemoteAddr)

	return r.RemoteAddr
}
