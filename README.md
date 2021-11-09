# iPXE Service

<img src="./docs/assets/logo.png" alt="Logo of the project" align="right">

## Status of the last deployments:
<img src="https://github.com/onmetal/ipxe-service/workflows/DockerImage2Harbor/badge.svg?branch-master">
<img src="https://github.com/onmetal/ipxe-service/workflows/ReleaseHelm/badge.svg?branch-master">

## Overview 

This project provides an HTTP server answering to requests according to matches, resources and mappings described as kubernetes resources.

It provides three different parts:

 - a library for an HTTP server serving requests according to configured query-matchers, mappings, resources and an optional Discovery API for metadata
 - a Kubernetes controller offering such a server by feeding it with configuration taken from Kubernetes resources.
 - a Kubernetes controller implementing the discovery API based on a machine Kubernetes resource

This ecosystem is intended to be used to serve iPXE requests when booting machines based on predefined rules. But it can also be used as a general matching engine to match requests to configurable resources.

## Installation, using and developing 

For more details please refer to documentation folder `/docs`

## Contributing 

We`d love to get a feedback from you. 
Please report bugs, suggestions or post question by opening a [Github issue]()

## License

[Apache License 2.0](https://github.com/helm/chart-testing/blob/main/LICENSE)
