# iPXE Service

<img src="./docs/assets/logo.png" alt="Logo of the project" align="right">

## Status of the last deployments:
[![Build and Publish Docker Image](https://github.com/ironcore-dev/ipxe-service/actions/workflows/publish-docker.yml/badge.svg)](https://github.com/ironcore-dev/ipxe-service/actions/workflows/publish-docker.yml)

## Overview 

The project provides an HTTP server which is answering to requests according to matches, resources and mappings described as kubernetes resources.

It consists of three different parts:

 - a library for an HTTP server serving requests according to configured query-matchers, mappings, resources and an optional Discovery API for metadata
 - a Kubernetes controller offering such a server by feeding it with configuration taken from Kubernetes resources.
 - a Kubernetes controller implementing the discovery API based on a machine Kubernetes resource

## Installation, using and developing 

For more details please refer to documentation folder `/docs`

## Contributing 

We`d love to get a feedback from you. 
Please report bugs, suggestions or post question by opening a [Github issue]()

## License

[Apache License 2.0](LICENCE)
