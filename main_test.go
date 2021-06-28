package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetChainHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/ipxe", nil)
	req.Header.Set("X-FORWARDED-FOR", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(getChain)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	expected := `#!ipxe

set base-url http://45.86.152.1/ipxe
kernel ${base-url}/rootfs.vmlinuz initrd=rootfs.initrd gl.ovl=/:tmpfs gl.url=${base-url}/root.squashfs gl.live=1 ip=dhcp console=ttyS1,115200n8 console=tty0 earlyprintk=ttyS1,115200n8 consoleblank=0 ignition.firstboot=1 ignition.config.url=${base-url}/ip${net0/ip}/ignition.json ignition.platform.id=metal
initrd ${base-url}/rootfs.initrd
boot
`
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}

func TestIgnitionHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/ignition", nil)
	req.Header.Set("X-FORWARDED-FOR", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(getIgnition)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	expected := `{"ignition":{"version":"3.1.0"},"passwd":{"users":[{"name":"core","sshAuthorizedKeys":["ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDmLUNS/nKfTxX95sOJB57qrKOqNggYZR/PUzeKgXVmpqWPfL33jh1c02RdJm028TcRLKRpu+HHOf4CeXZf52qOgqETVNwPa12LGR0u2ucSrAIxWqhuOr/P2A35rp7BAmpNFWS0PIUr6IIPapbe8tVvuVgrlJga03LuTSH8XuHutN0WWUi2l0qFze+3+RqmhGTrCGIAM2XBC1LgnOobOMYDNxc5HD7Hai8frxoGuXVBA2yOIgUin4DYNV/8Fo4vBhAPjqzMNoWKHY01cySXYbvuTZP0jccoMHwECVIwOCijOettHRN32wmbpuBtGdh6DUwLo8iIGOV948oWe/YQPC4D dima@fobos2"]}]}}`
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}
func TestIgnition204Handler(t *testing.T) {
	req, err := http.NewRequest("GET", "/ignition", nil)
	req.Header.Set("X-FORWARDED-FOR", "127.0.0.100")
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(getIgnition)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNoContent {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusNoContent)
	}

	expected := `not found netdata
`
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got '%v' want '%v'",
			rr.Body.String(), expected)
	}
}

func TestRoot200Handler(t *testing.T) {
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(ok200)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusNoContent)
	}

	expected := "ok\n"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got '%v' want '%v'",
			rr.Body.String(), expected)
	}
}
