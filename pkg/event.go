package pkg

import (
	"bytes"
	"github.com/onmetal/metal-api-gateway/app/logger"
	"io/ioutil"
	"net/http"
	"os"
)

type httpClient struct {
	*http.Client

	log logger.Logger
}

// TODO(flpeter) check with Andre
type event struct {
	UUID    string `json:"uuid"`
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`
}

func newHttp() *httpClient {
	c := &http.Client{Timeout: TimeoutSecond}
	l := logger.New()
	return &httpClient{
		Client: c,
		log:    l,
	}
}

func (h *httpClient) postRequest(requestBody []byte) ([]byte, error) {
	var url string
	if os.Getenv("HANDLER_URL") == "" {
		url = "http://localhost:8088/api/v1/event"
	} else {
		url = os.Getenv("HANDLER_URL")
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}
	token, err := getToken()
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", token)
	resp, err := h.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func getToken() (string, error) {
	if _, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount/token"); os.IsNotExist(err) {
		return "", err
	}
	data, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

//TODO(flpeter) check with Andre
//func postEvent(ip string, uuid string) {
//	e := &event{
//		UUID:    uuid,
//		Reason:  "Ignition",
//		Message: fmt.Sprintf("Ignition request for ip %s", ip),
//	}
//	h := newHttp()
//	requestBody, _ := json.Marshal(e)
//	resp, err := h.postRequest(requestBody)
//	if err != nil {
//		h.log.Error("can't send a request", err)
//		fmt.Println(string(resp))
//	}
//}
