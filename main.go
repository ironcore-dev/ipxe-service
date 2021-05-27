package main

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	netdata "netdata/api/v1"
        inv "k8s-inventory/api/v1alpha1"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func main() {
	http.HandleFunc("/ipxe", getNetdata)
	if err := http.ListenAndServe(":8082", nil); err != nil {
		fmt.Println("Failed to start IPXE Server\n")
		os.Exit(1)
	}
}

var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypesInv)
)

func addKnownTypesInv(scheme *runtime.Scheme) error {
        scheme.AddKnownTypes(inv.GroupVersion,
                &inv.Inventory{},
                &inv.InventoryList{},
        )

        metav1.AddToGroupVersion(scheme, inv.GroupVersion)
        return nil
}

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(netdata.GroupVersion,
		&netdata.Netdata{},
		&netdata.NetdataList{},
	)

	metav1.AddToGroupVersion(scheme, netdata.GroupVersion)
	return nil
}

func getInventory(w http.ResponseWriter, r *http.Request) {
        inv.AddToScheme(scheme.Scheme)

        cl, err := client.New(config.GetConfigOrDie(), client.Options{})
        if err != nil {
                fmt.Println("Failed to create a client\n")
                os.Exit(1)
        }

        var inventory inv.InventoryList
        err = cl.List(context.Background(), &inventory, client.InNamespace("default"), client.MatchingLabels{"macAddr": "3868dd268df5"})
        if err != nil {
                fmt.Println("Failed to list crds netdata in namespace default: %v\n", err)
                os.Exit(1)
        }

        clientUUID := inventory.Items[0].Spec.System.ID
        fmt.Println(clientUUID)
}

func getNetdata(w http.ResponseWriter, r *http.Request) {
        netdata.AddToScheme(scheme.Scheme)

        cl, err := client.New(config.GetConfigOrDie(), client.Options{})
        if err != nil {
                fmt.Println("Failed to create a client\n")
                os.Exit(1)
        }

        var crds netdata.NetdataList
        err = cl.List(context.Background(), &crds, client.InNamespace("default"), client.MatchingLabels{"ipv4": "10.20.30.40"})
        if err != nil {
                fmt.Println("Failed to list crds netdata in namespace default: %v\n", err)
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
