// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package pkg

import "time"

const (
	TimeoutSecond           = 5 * time.Second
	ConfigFile              = "/etc/ipxe-service/config.yaml"
	ServiceServerCert       = "ipxe-service-server-cert"
	DefaultSecretPath       = "/etc/ipxe-default-secret"
	DefaultConfigMapPath    = "/etc/ipxe-default-cm"
	InventoryMacLabelPrefix = "metal.ironcore.dev/mac-address-"
)
