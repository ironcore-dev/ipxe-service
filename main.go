package main

import (
	"fmt"
	"github.com/onmetal/ipxe-service/pkg"
)

func main() {
	fmt.Println("iPXE is stating ...")

	conf := pkg.GetConf()
	k8sClient := pkg.NewK8sClient()
	ipxe := pkg.IPXE{
		Config:    conf,
		K8sClient: k8sClient,
	}

	ipxe.Start()
}
