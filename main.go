// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"

	"github.com/ironcore-dev/ipxe-service/pkg"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func main() {
	fmt.Println("iPXE is stating ...")

	conf := pkg.GetConf(pkg.ConfigFile)
	k8sClient := pkg.NewK8sClient(nil, client.Options{})
	ipxe := pkg.IPXE{
		Config:    conf,
		K8sClient: k8sClient,
	}

	ipxe.Start()
}
