# ipxe-service

## Install iPXE into Kubernetes cluster

```bash
helm install ipxe-service ./chart
```
## Run iPXE locally

To build and run iPXE locally:
```bash
make all
make run
```

## HTTP Requests

For simple test use following command:
```bash
curl -k https://ipxe-service.local.ns1.fra3.infra.onmetal.de
ok
```
To get answer from iPXE :
```bash
curl -k https://ipxe-service.local.ns1.fra3.infra.onmetal.de/ipxe
#!ipxe

set base-url http://45.86.152.1/ipxe
kernel ${base-url}/rootfs.vmlinuz initrd=rootfs.initrd gl.ovl=/:tmpfs gl.url=${base-url}/root.squashfs gl.live=1 ip=dhcp console=ttyS1,115200n8 console=tty0 earlypri
ntk=ttyS1,115200n8 consoleblank=0 ignition.firstboot=1 ignition.config.url=${base-url}/ip${net0/ip}/ignition.json ignition.platform.id=metal
initrd ${base-url}/rootfs.initrd

boot
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
