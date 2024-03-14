// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package pkg

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v1"
)

type Config struct {
	ConfigmapNS          string `yaml:"configmap-namespace"`
	IpamNS               string `yaml:"ipam-namespace"`
	MachineRequestNS     string `yaml:"machine-request-namespace"`
	InventoryNS          string `yaml:"inventory-namespace"`
	ImageNS              string `yaml:"k8simage-namespace"`
	DisableForwardHeader bool   `yaml:"disable-forward-header,omitempty"`
}

func GetConf(configFile string) Config {
	var c Config
	yamlFile, err := os.ReadFile(configFile)
	if err != nil {
		log.Printf("Can not read config %s, err %s\n", ConfigFile, err)
		ns, _ := getInClusterNamespace()
		if len(ns) == 0 {
			ns = "default"
		}
		log.Printf("Application will use namespace '%s' for everything\n", ns)
		c = Config{
			ConfigmapNS:      ns,
			IpamNS:           ns,
			MachineRequestNS: ns,
			InventoryNS:      ns,
			ImageNS:          ns}
		log.Printf("Config is #%+v ", c)
		return c
	}
	err = yaml.Unmarshal(yamlFile, &c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}
	log.Printf("Config is #%+v ", c)
	return c
}

func getInClusterNamespace() (string, error) {
	ns, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("cannot determine in-cluster namespace: %w", err)
	}
	return string(ns), nil
}
