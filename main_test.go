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

	//expected := `{"IP":"127.0.0.1","MAC":"16:bf:7b:2f:8e:9c","UUID":"a967954c-3475-11b2-a85c-84d8b4f8cd2d"}`
	expected := "#!ipxe\nset base-url http://45.86.152.1/ipxe\nkernel ${base-url}/rootfs.vmlinuz initrd=rootfs.initrd gl.ovl=/:tmpfs gl.url=${base-url}/root.squashfs gl.live=1 ip=dhcp console=ttyS1,115200n8 console=tty0 earlyprintk=ttyS1,115200n8 consoleblank=0 ignition.firstboot=1 ignition.config.url=${base-url}/tmp.ign ignition.platform.id=metal\ninitrd ${base-url}/rootfs.initrd\nboot\n"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}
