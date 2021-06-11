# ipxe-service

---

## Go run

```bash
$ go run main.go
```

or

```bash
$ make run
```

## HTTP Request

```bash
$ curl 127.0.0.1:8082
404 page not found
```

```bash
$ curl -s 127.0.0.1:8082/ipxe | jq .
{
  "IP": "127.0.0.1",
  "MAC": "16:bf:7b:2f:8e:9c",
  "UUID": "a967954c-3475-11b2-a85c-84d8b4f8cd2d"
}
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

