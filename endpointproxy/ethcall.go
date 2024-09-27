package endpointproxy

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/celer-network/goutils/log"
)

type EthCallProxy struct {
	ethCallTargetUrl *url.URL
}

// NewProxy takes target host and creates a reverse proxy
func (h *EthCallProxy) startEthCallProxy(targetHost string, port int, chainId uint64) error {
	var err error
	h.ethCallTargetUrl, err = url.Parse(targetHost)
	if err != nil {
		return err
	}
	p := httputil.NewSingleHostReverseProxy(h.ethCallTargetUrl)
	originalDirector := p.Director
	p.Director = func(req *http.Request) {
		originalDirector(req)
		h.modifyEthCallRequest(req)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", proxyRequestHandler(p))
	go startCustomProxyByPort(port, mux, chainId, targetHost)
	return nil
}

func (h *EthCallProxy) modifyEthCallRequest(req *http.Request) {
	req.URL.Scheme = h.ethCallTargetUrl.Scheme
	req.URL.Host = h.ethCallTargetUrl.Host
	req.Host = h.ethCallTargetUrl.Host
	req.URL.Path = strings.TrimRight(req.URL.Path, "/")
	reqStr, err := io.ReadAll(req.Body)
	if err != nil {
		log.Errorf("invalid eth_call request err:%s", err.Error())
		return
	}
	msg := &jsonrpcMessage{}
	if err = json.Unmarshal(reqStr, msg); err != nil {
		log.Errorf("fail to unmarshal this eth_call req body err:%s", err.Error())
		return
	}

	switch msg.Method {
	case MethodEthCall, MethodEthEstimateGas:
		newParams := strings.Replace(string(msg.Params), "\"input\":", "\"data\":", 1)
		msg.Params = []byte(newParams)
	}
	newMsg, marshalErr := json.Marshal(msg)
	if marshalErr != nil {
		log.Errorf("fail to marshal this new eth_call req, raw:%s, err:%s", string(newMsg), marshalErr.Error())
		return
	}

	req.Header.Set(HeaderRpcMethod, msg.Method)
	req.Body = io.NopCloser(bytes.NewReader(newMsg))
	req.ContentLength = int64(len(newMsg))
}
