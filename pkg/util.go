package pkg

import (
	"bytes"
	"encoding/hex"
	"github.com/Masterminds/sprig"
	buconfig "github.com/coreos/butane/config"
	"github.com/coreos/butane/config/common"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path"
	"text/template"
)

func IpVersion(s string) string {
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '.':
			return "ipv4"
		case ':':
			return "ipv6"
		}
	}
	return ""
}

func FullIPv6(ip net.IP) string {
	dst := make([]byte, hex.EncodedLen(len(ip)))
	_ = hex.Encode(dst, ip)
	return string(dst[0:4]) + ":" +
		string(dst[4:8]) + ":" +
		string(dst[8:12]) + ":" +
		string(dst[12:16]) + ":" +
		string(dst[16:20]) + ":" +
		string(dst[20:24]) + ":" +
		string(dst[24:28]) + ":" +
		string(dst[28:])
}

func doesFileExist(fileName string) bool {
	_, err := os.Stat(fileName)
	// check if error is "file not exists"
	if os.IsNotExist(err) {
		return false
	} else {
		return true
	}
}

func renderButane(dataIn []byte) string {
	// render by butane to json
	options := common.TranslateBytesOptions{
		Raw:    true,
		Strict: false,
		Pretty: false,
	}
	options.NoResourceAutoCompression = true
	dataOut, _, err := buconfig.TranslateBytes(dataIn, options)
	if err != nil {
		log.Printf("\nError in ignition rendering.dataIn is : %+v\n", dataIn)
		log.Printf("Error in ignition rendering: %+v", err)
	}
	return string(dataOut)
}

func readIpxeConfFile(part string) ([]byte, error) {
	var ipxeData []byte
	var err error
	ipxeData, err = ioutil.ReadFile(path.Join(DefaultSecretPath, part))
	if err != nil {
		ipxeData, err = ioutil.ReadFile(path.Join(DefaultConfigMapPath, part))
		if err != nil {
			log.Printf("Problem with default secret and configmap #%v ", err)
			return nil, err
		}
	}

	return ipxeData, nil
}

func renderIpxeMacConfFile(mac, part string) ([]byte, error) {
	ipxeData, err := readIpxeConfFile(part)
	if err != nil {
		return nil, err
	}

	type Config struct {
		Mac string
	}
	cfg := Config{Mac: mac}
	tmpl, err := template.New(part).Funcs(sprig.HermeticTxtFuncMap()).Parse(string(ipxeData))
	if err != nil {
		return nil, err
	}
	var ipxe bytes.Buffer
	err = tmpl.Execute(&ipxe, cfg)
	if err != nil {
		return nil, err
	}

	return ipxe.Bytes(), nil
}
