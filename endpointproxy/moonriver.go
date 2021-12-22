package endpointproxy

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/celer-network/goutils/log"
)

type MoonRiverProxy struct {
	moonRiverTargetUrl *url.URL
}

// NewProxy takes target host and creates a reverse proxy
func (m *MoonRiverProxy) startMoonRiverProxy(targetHost string, port int) error {
	var err error
	m.moonRiverTargetUrl, err = url.Parse(targetHost)
	if err != nil {
		return err
	}
	p := httputil.NewSingleHostReverseProxy(m.moonRiverTargetUrl)
	originalDirector := p.Director
	p.Director = func(req *http.Request) {
		originalDirector(req)
		m.modifyMoonRiverRequest(req)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", proxyRequestHandler(p))
	go startCustomProxyByPort(port, mux)
	return nil
}

func (m *MoonRiverProxy) modifyMoonRiverRequest(req *http.Request) {
	req.URL.Scheme = m.moonRiverTargetUrl.Scheme
	req.URL.Host = m.moonRiverTargetUrl.Host
	req.Host = m.moonRiverTargetUrl.Host
	reqStr, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Errorf("invalid moonriver request err:%s", err.Error())
		return
	}
	msg := &jsonrpcMessage{}
	if err = json.Unmarshal(reqStr, msg); err != nil {
		log.Errorf("fail to unmarshal this moonriver req body err:%s", err.Error())
		return
	}
	if msg.Method == MethodEthGetCode {
		newParams := strings.Replace(string(msg.Params), "\"pending\"", "\"latest\"", 1)
		msg.Params = []byte(newParams)
	}
	newMsg, marshalErr := json.Marshal(msg)
	if marshalErr != nil {
		log.Errorf("fail to marshal this new moonriver req, raw:%s, err:%s", string(newMsg), marshalErr.Error())
		return
	}
	req.Body = ioutil.NopCloser(bytes.NewReader(newMsg))
	req.ContentLength = int64(len(newMsg))
}
