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
