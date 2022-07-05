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

type ConfluxProxy struct {
	confluxTargetUrl *url.URL
}

// NewProxy takes target host and creates a reverse proxy
func (h *ConfluxProxy) startConfluxProxy(targetHost string, port int, chainId uint64) error {
	var err error
	h.confluxTargetUrl, err = url.Parse(targetHost)
	if err != nil {
		return err
	}
	p := httputil.NewSingleHostReverseProxy(h.confluxTargetUrl)
	originalDirector := p.Director
	p.Director = func(req *http.Request) {
		originalDirector(req)
		h.modifyConfluxRequest(req)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", proxyRequestHandler(p))
	go startCustomProxyByPort(port, mux, chainId)
	return nil
}

func (h *ConfluxProxy) modifyConfluxRequest(req *http.Request) {
	req.URL.Scheme = h.confluxTargetUrl.Scheme
	req.URL.Host = h.confluxTargetUrl.Host
	req.Host = h.confluxTargetUrl.Host
	reqStr, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Errorf("invalid conflux request err:%s", err.Error())
		return
	}
	msg := &jsonrpcMessage{}
	if err = json.Unmarshal(reqStr, msg); err != nil {
		log.Errorf("fail to unmarshal this conflux req body err:%s", err.Error())
		return
	}
	if msg.Method == MethodEthGetCode {
		newParams := strings.Replace(string(msg.Params), "\"pending\"", "\"latest\"", 1)
		msg.Params = []byte(newParams)
	}
	if msg.Method == MethodEthCall {
		newParams := strings.Replace(string(msg.Params), ",\"from\":\"0x0000000000000000000000000000000000000000\"", "", 1)
		msg.Params = []byte(newParams)
	}
	newMsg, marshalErr := json.Marshal(msg)
	if marshalErr != nil {
		log.Errorf("fail to marshal this new conflux req, raw:%s, err:%s", string(newMsg), marshalErr.Error())
		return
	}
	req.Body = ioutil.NopCloser(bytes.NewReader(newMsg))
	req.ContentLength = int64(len(newMsg))
}
