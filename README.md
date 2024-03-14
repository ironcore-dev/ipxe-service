# iPXE Service

<img src="./docs/assets/logo.png" alt="Logo of the project" align="right">

[![REUSE status](https://api.reuse.software/badge/github.com/ironcore-dev/ipxe)](https://api.reuse.software/info/github.com/ironcore-dev/ipxe)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat-square)](http://makeapullrequest.com)
[![Build](https://github.com/ironcore-dev/ipxe/actions/workflows/publish-docker.yml/badge.svg)](https://github.com/ironcore-dev/ipxe/actions/workflows/publish-docker.yml)
[![GitHub License](https://img.shields.io/static/v1?label=License&message=Apache-2.0&color=blue&style=flat-square)](LICENSE)

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

[Apache License 2.0](LICENSE)
