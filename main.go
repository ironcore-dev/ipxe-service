package main

import (
  "context"
  "encoding/json"
  "fmt"
  "k8s.io/apimachinery/pkg/runtime"
  "k8s.io/client-go/kubernetes/scheme"

  metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
  "net/http"
  dev1 "netdata/api/v1"
  "os"
  "sigs.k8s.io/controller-runtime/pkg/client"
  "sigs.k8s.io/controller-runtime/pkg/client/config"
)

func main() {
  http.HandleFunc("/ipxe", ipxeServer)
  if err := http.ListenAndServe(":8082", nil); err != nil {
    fmt.Printf("Failed to start IPXE Server\n")
    os.Exit(1)
  }

}

var (
  SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
)

func addKnownTypes(scheme *runtime.Scheme) error {
  scheme.AddKnownTypes(dev1.GroupVersion,
    &dev1.Netdata{},
    &dev1.NetdataList{},
)

  metav1.AddToGroupVersion(scheme, dev1.GroupVersion)
  return nil
}

func ipxeServer(w http.ResponseWriter, r *http.Request) {

  dev1.AddToScheme(scheme.Scheme)

  fmt.Printf("URL: %s\n", r.URL.Path[1:])
  fmt.Printf("GroupVersion: %s\n", dev1.GroupVersion)

  cl, err := client.New(config.GetConfigOrDie(), client.Options{})
  if err != nil {
    fmt.Printf("Failed to create a client\n")
    os.Exit(1)
  }
	
  var crds dev1.NetdataList
  err = cl.List(context.Background(), &crds, client.InNamespace("default"))
  if err != nil {
    fmt.Printf("Failed to list crds netdata in namespace default: %v\n", err)
    os.Exit(1)
  }

  w.Header().Add("Content-Type", "application/json")
  resp, _ := json.Marshal(map[string]string{
    "IPAddress": GetIP(r),
    "CRDName": crds.Items[0].ObjectMeta.Name,
  })

  w.Write(resp)
}

func GetIP(r *http.Request) string {
  forwarded := r.Header.Get("X-FORWARDED-FOR")
  if forwarded != "" {
    return forwarded
  }
  
  return r.RemoteAddr
}

