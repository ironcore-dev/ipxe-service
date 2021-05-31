package main

import (
	"context"
	"fmt"
	"k8s.io/client-go/kubernetes/scheme"

	inv "k8s-inventory/api/v1alpha1"
	mreq1 "k8s-machine-requests/api/v1alpha1"
	"net/http"
	netdata "netdata/api/v1"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func main() {
	http.HandleFunc("/ipxe", getNetdata)
	http.HandleFunc("/inv", getInventory)
	http.HandleFunc("/reqs", getMachineRequest)
	if err := http.ListenAndServe(":8082", nil); err != nil {
		fmt.Println("Failed to start IPXE Server")
		os.Exit(1)
	}
}

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

func getInventory(w http.ResponseWriter, r *http.Request) {
	if err := inv.AddToScheme(scheme.Scheme); err != nil {
		fmt.Println("unable to add registered types inventory to client scheme")
		os.Exit(1)
	}

	cl, err := client.New(config.GetConfigOrDie(), client.Options{})
	if err != nil {
		fmt.Println("Failed to create a client")
		os.Exit(1)
	}

	var inventory inv.InventoryList
	err = cl.List(context.Background(), &inventory, client.InNamespace("default"), client.MatchingLabels{"macAddr": "3868dd268df5"})
	if err != nil {
		fmt.Println("Failed to list crds netdata in namespace default")
		os.Exit(1)
	}

	clientUUID := inventory.Items[0].Spec.System.ID
	fmt.Println(clientUUID)
}

func getNetdata(w http.ResponseWriter, r *http.Request) {
	if err := netdata.AddToScheme(scheme.Scheme); err != nil {
		fmt.Println("unable to add registered types netdata to client scheme")
		os.Exit(1)
	}

	cl, err := client.New(config.GetConfigOrDie(), client.Options{})
	if err != nil {
		fmt.Println("Failed to create a client")
		os.Exit(1)
	}

	var crds netdata.NetdataList
	err = cl.List(context.Background(), &crds, client.InNamespace("default"), client.MatchingLabels{"ipv4": "10.20.30.40"})
	if err != nil {
		fmt.Println("Failed to list crds netdata in namespace default")
		os.Exit(1)
	}

	// TODO:
	// 1. check multi CRDs
	// 2. check does an element exists (CRD)

	clientMACAddr := crds.Items[0].Spec.MACAddress
	fmt.Println(clientMACAddr)
	//return clientMACAddr
}

func getIP(r *http.Request) string {
	forwarded := r.Header.Get("X-FORWARDED-FOR")
	if forwarded != "" {
		return forwarded
	}

	return r.RemoteAddr
}
