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
kubectl create -f ./config/samples/machine.onmetal.de_v1_netdata.yaml -f ./config/samples/machine_v1alpha1_inventory.yaml
make test

```
