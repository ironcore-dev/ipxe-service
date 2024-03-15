# iPXE Service

<img src="./assets/logo.png" alt="Logo of the project" align="right">

## Overview

This project provides an HTTP server answering to requests according to matches, resources and mappings described as kubernetes resources.

It provides three different parts:

 - a library for an HTTP server serving requests according to configured query-matchers, mappings, resources and an optional Discovery API for metadata
 - a Kubernetes controller offering such a server by feeding it with configuration taken from Kubernetes resources.
 - a Kubernetes controller implementing the discovery API based on a machine Kubernetes resource

This ecosystem is intended to be used to serve iPXE requests when booting machines based on predefined rules. But it can also be used as a general matching engine to match requests to configurable resources.

## Licensing

[Apache License 2.0](https://github.com/helm/chart-testing/blob/main/LICENSE)
