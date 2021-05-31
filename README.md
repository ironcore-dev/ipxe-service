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
