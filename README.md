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
$ curl 127.0.0.1:8082/ipxe && printf "\n"
{"CRDName":"16bf7b2f8e9c","IPAddress":"127.0.0.1:43406"}
```
