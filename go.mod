module ipxe-service

go 1.16

require (
	github.com/coreos/butane v0.12.1
	github.com/google/go-cmp v0.5.6 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/onmetal/k8s-inventory v0.0.0-20210608091530-5af0dfa20b72
	github.com/onmetal/k8s-machine-requests v0.0.0-20210505193151-fbf6c179a00d
	github.com/onmetal/netdata v0.0.0-20210628111550-04c33fc83084
	gopkg.in/yaml.v1 v1.0.0-20140924161607-9f9df34309c0
	k8s.io/client-go v0.21.2
	k8s.io/klog/v2 v2.9.0 // indirect
	sigs.k8s.io/controller-runtime v0.9.2
	sigs.k8s.io/structured-merge-diff/v4 v4.1.1 // indirect
)
