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
	handler := http.HandlerFunc(getChainDefault)
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

	expected := `{"ignition":{"version":"3.2.0"},"passwd":{"users":[{"name":"core","sshAuthorizedKeys":["ssh-rsa AAAA"]}]}}`
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

	expected := `Not found ipam ip obj
`
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got '%v' want '%v'",
			rr.Body.String(), expected)
	}
}

func TestIgnPartHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/ignition/testpart", nil)
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

	expected := `{"ignition":{"version":"3.2.0"},"passwd":{"users":[{"name":"testuser","sshAuthorizedKeys":["ssh-rsa TTTT"]}]}}`
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}
func TestIgnSecretPartHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/ignition/fromsecret", nil)
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

	expected := `{"ignition":{"version":"3.2.0"},"storage":{"files":[{"overwrite":true,"path":"/root/install.sh","contents":{"source":"http://45.86.152.1/install-kubernetes-from-scratch.sh"},"mode":493},{"overwrite":true,"path":"/etc/init.d/helper.sh","contents":{"compression":"gzip","source":"data:;base64,H4sIAAAAAAAC/6SOP0/DMBDFd3+Kh/GQVHKOdGFAqcSAxAIMsFGQUvtCLLlOiB0qpHx41JZUFSvb6f2537u8oDEOtHGBOHxhU8dWCM915FhJlX0M3EN7yPun55fH24e7SoKGMVD8jom3lgIn19CxQItcil3rPOMVKmPTdpDq6ElM2Blon0P7hCXebmA7AQD/oe370TP3KIXtAou2iynUW65UZuqEXzomrA/ZA+H8/WzEbkiYMAb3CW3+yql2HjqUJ92MCdpW0M0yFyKyhXaQ9F4ur4uroizKYkFWgjgZ2k+Koh9cSA3kKQE1b4XK5jNfB4nV6rz4EwAA//8Mpfu4ogEAAA=="},"mode":493}]},"systemd":{"units":[{"dropins":[{"contents":"[Service]\nExecStartPost=/etc/init.d/helper.sh\n","name":"updatehosts.conf"}],"enabled":true,"name":"systemd-hostnamed.service"}]}}`
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
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
