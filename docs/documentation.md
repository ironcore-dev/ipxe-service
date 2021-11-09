# iPXE Service

<img src="./images/logo.sample.png" alt="Logo of the project" align="right">

[![Build Status](https://img.shields.io/travis/npm/npm/latest.svg?style=flat-square)](https://travis-ci.org/npm/npm) [![npm](https://img.shields.io/npm/v/npm.svg?style=flat-square)](https://www.npmjs.com/package/npm) [![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat-square)](http://makeapullrequest.com) [![GitHub license](https://img.shields.io/badge/license-MIT-blue.svg?style=flat-square)](https://github.com/your/your-project/blob/master/LICENSE)

## Background/Overview 

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

### Built With 
List main libraries, frameworks used including versions 

### Prerequisites
What is needed to set up the dev environment. For instance, global dependencies or any other tools. include download links

A quick introduction of the minimal setup/ required tools  you need to get a hello world up & running.

```shell
commands here
```
Here you should say what actually happens when you execute the code above.

### Setting up Dev

Here's a brief intro about what a developer must do in order to start developing
the project further:

```shell
git clone https://github.com/your/your-project.git
cd your-project/
packagemanager install
```

And state what happens step-by-step. If there is any virtual environment, local server or database feeder needed, explain here.

### Building (if needed)

If your project needs some additional steps for the developer to build the
project after some code changes, state them here. for example:

```shell
./configure
make
make install
```

Here again you should state what actually happens when the code above gets executed.

### Deploying / Publishing (if needed)
give instructions on how to build and release a new version
In case there's some step you have to take that publishes this project to a
server, this is the right time to state it.

```shell
packagemanager deploy your-project -s server.com -u username -p password
```

And again you'd need to tell what the previous code actually does.

## Versioning

We can maybe use [SemVer](http://semver.org/) for versioning. For the versions available, see the [link to tags on this repository](/tags).


## Configuration

Here you should write what are all of the configurations a user can enter when using the project.

## Tests

For simple test use following command:

```shell
curl -k https://ipxe-service.local.ns1.fra3.infra.onmetal.de
ok
```

To get answer from iPXE :

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
