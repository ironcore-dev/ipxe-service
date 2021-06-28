# ipxe-service

---

## Go run

```bash
go run main.go
```

or

```bash
make run
```

## HTTP Request

```bash
curl 127.0.0.1:8082
ok
```


```bash
curl 127.0.0.1:8082/ipxe
#!ipxe

set base-url http://45.86.152.1/ipxe
kernel ${base-url}/rootfs.vmlinuz initrd=rootfs.initrd gl.ovl=/:tmpfs gl.url=${base-url}/root.squashfs gl.live=1 ip=dhcp console=ttyS1,115200n8 console=tty0 earlyprintk=ttyS1,115200n8 consoleblank=0 ignition.firstboot=1 ignition.config.url=${base-url}/ip${net0/ip}/ignition.json ignition.platform.id=metal
initrd ${base-url}/rootfs.initrd

boot
```

```
curl 127.0.0.1:8082/ignition

{"ignition":{"version":"3.1.0"},"passwd":{"users":[{"name":"core","sshAuthorizedKeys":["ssh-rsa AAAAB3NzaC1yc2EAAAADAV948oWe/YQPC4D key@key"]}]}}
```


### Test with minikube

```
kubectl create -f ./config/crd/bases/machine.onmetal.de_inventories.yaml -f ./config/crd/bases/machine.onmetal.de_netdata.yaml
kubectl create -f ./config/samples/machine.onmetal.de_v1_netdata.yaml
make test
```
## Exit codes

- **11** - Failed to start IPXE Server
- **12** - Unable to add registered types machine request to client scheme
- **14** - Failed to list machine requests in namespace default
- **15** - Unable to add registered types inventory to client scheme
- **17** - Failed to list crds netdata in namespace default
- **18** - Unable to add registered types netdata to client scheme
- **19** - Failed to create an  client
- **20** - Failed to list crds netdata in namespace default
- **21** - Unable to create a client's ipxe directory
- **22** - Unable to create a client's ipxe file
- **23** - Unable to read the default ipxe config file
- **33** - Not found netdata for ipv4

