# iPXE Service

<img src="./assets/logo.png" alt="Logo of the project" align="right">

## Overview

This project provides an HTTP server answering to requests according to matches, resources and mappings described as kubernetes resources.

It provides three different parts:

 - a library for an HTTP server serving requests according to configured query-matchers, mappings, resources and an optional Discovery API for metadata
 - a Kubernetes controller offering such a server by feeding it with configuration taken from Kubernetes resources.
 - a Kubernetes controller implementing the discovery API based on a machine Kubernetes resource

This ecosystem is intended to be used to serve iPXE requests when booting machines based on predefined rules. But it can also be used as a general matching engine to match requests to configurable resources.

## Instalation

### Install iPXE into Kubernetes cluster
If you want to specify a location different that fra3, please provide it as a value. Currently only fra3 and fra4 are supported

```shell
helm install ipxe-service ./chart [--set location=fra4]
```

### Run iPXE locally
To build and run iPXE locally:

```shell
make all
make run
curl http://127.0.0.1:8082
```

### Setting up Dev

Here's a brief intro about what a developer must do in order to start developing the project further:

```shell
git clone https://github.com/ironcore-dev/ipxe.git
cd ipxe/
helm install ipxe ./chart
```

## Tests

For simple test use following command:

```shell
curl -k https://ipxe-service.local.ns1.fra3.infra.onmetal.de
ok
```

To get ca certificate for validate https
```shell
curl -k https://ipxe-service.local.ns1.fra3.infra.onmetal.de/cert
```

To get answer from iPXE:

```shell
curl -k https://ipxe-service.local.ns1.fra3.infra.onmetal.de/ipxe
#!ipxe

set base-url http://45.86.152.1/ipxe
kernel ${base-url}/rootfs.vmlinuz initrd=rootfs.initrd gl.ovl=/:tmpfs gl.url=${base-url}/root.squashfs gl.live=1 ip=dhcp console=ttyS1,115200n8 console=tty0 earlypri
ntk=ttyS1,115200n8 consoleblank=0 ignition.firstboot=1 ignition.config.url=${base-url}/ip${net0/ip}/ignition.json ignition.platform.id=metal
initrd ${base-url}/rootfs.initrd

boot
```

## Licensing

[Apache License 2.0](https://github.com/helm/chart-testing/blob/main/LICENSE)
