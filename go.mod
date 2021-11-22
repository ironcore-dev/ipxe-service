module ipxe-service

go 1.17

require (
	github.com/coreos/butane v0.12.1
	github.com/google/go-cmp v0.5.6 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/onmetal/ipam v0.0.0-20211029144623-1398cd13a1ae // indirect
	github.com/onmetal/k8s-image v0.0.0-20210829194124-989dac3e4e7f
	github.com/onmetal/k8s-inventory v0.0.0-20210608091530-5af0dfa20b72
	github.com/onmetal/k8s-machine-instance v0.0.0-20210812162659-331455e053d8
	github.com/onmetal/k8s-machine-requests v0.0.0-20210901134901-3a2a3b92842c
	github.com/onmetal/machine-operator v0.9.0
	github.com/onmetal/netdata v0.0.0-20210906100955-18e190d4cdee
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/common v0.30.0 // indirect
	gopkg.in/yaml.v1 v1.0.0-20140924161607-9f9df34309c0
	k8s.io/apimachinery v0.21.2
	k8s.io/client-go v0.21.2
	k8s.io/klog/v2 v2.9.0 // indirect
	sigs.k8s.io/controller-runtime v0.9.2
	sigs.k8s.io/structured-merge-diff/v4 v4.1.1 // indirect
)
