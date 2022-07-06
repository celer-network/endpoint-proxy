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

type CloverProxy struct {
	cloverTargetUrl *url.URL
}

// NewProxy takes target host and creates a reverse proxy
func (h *CloverProxy) startCloverProxy(targetHost string, port int, chainId uint64) error {
	var err error
	h.cloverTargetUrl, err = url.Parse(targetHost)
	if err != nil {
		return err
	}
	p := httputil.NewSingleHostReverseProxy(h.cloverTargetUrl)
	originalDirector := p.Director
	p.Director = func(req *http.Request) {
		originalDirector(req)
		h.modifyCloverRequest(req)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", proxyRequestHandler(p))
	go startCustomProxyByPort(port, mux, chainId)
	return nil
}

func (h *CloverProxy) modifyCloverRequest(req *http.Request) {
	req.URL.Scheme = h.cloverTargetUrl.Scheme
	req.URL.Host = h.cloverTargetUrl.Host
	req.Host = h.cloverTargetUrl.Host
	reqStr, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Errorf("invalid clover request err:%s", err.Error())
		return
	}
	msg := &jsonrpcMessage{}
	if err = json.Unmarshal(reqStr, msg); err != nil {
		log.Errorf("fail to unmarshal this clover req body err:%s", err.Error())
		return
	}
	if msg.Method == MethodEthGetCode {
		newParams := strings.Replace(string(msg.Params), "\"pending\"", "\"latest\"", 1)
		msg.Params = []byte(newParams)
	}
	newMsg, marshalErr := json.Marshal(msg)
	if marshalErr != nil {
		log.Errorf("fail to marshal this new clover req, raw:%s, err:%s", string(newMsg), marshalErr.Error())
		return
	}
	req.Body = ioutil.NopCloser(bytes.NewReader(newMsg))
	req.ContentLength = int64(len(newMsg))
}
