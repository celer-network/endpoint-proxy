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

type CrabProxy struct {
	crabTargetUrl *url.URL
}

// NewProxy takes target host and creates a reverse proxy
func (h *CrabProxy) startCrabProxy(targetHost string, port int, chainId uint64) error {
	var err error
	h.crabTargetUrl, err = url.Parse(targetHost)
	if err != nil {
		return err
	}
	p := httputil.NewSingleHostReverseProxy(h.crabTargetUrl)
	originalDirector := p.Director
	p.Director = func(req *http.Request) {
		originalDirector(req)
		h.modifyCrabRequest(req)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", proxyRequestHandler(p))
	go startCustomProxyByPort(port, mux, chainId)
	return nil
}

func (h *CrabProxy) modifyCrabRequest(req *http.Request) {
	req.URL.Scheme = h.crabTargetUrl.Scheme
	req.URL.Host = h.crabTargetUrl.Host
	req.Host = h.crabTargetUrl.Host
	reqStr, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Errorf("invalid crab request err:%s", err.Error())
		return
	}
	msg := &jsonrpcMessage{}
	if err = json.Unmarshal(reqStr, msg); err != nil {
		log.Errorf("fail to unmarshal this crab req body err:%s", err.Error())
		return
	}
	if msg.Method == MethodEthGetCode {
		newParams := strings.Replace(string(msg.Params), "\"pending\"", "\"latest\"", 1)
		msg.Params = []byte(newParams)
	}
	newMsg, marshalErr := json.Marshal(msg)
	if marshalErr != nil {
		log.Errorf("fail to marshal this new crab req, raw:%s, err:%s", string(newMsg), marshalErr.Error())
		return
	}
	req.Body = ioutil.NopCloser(bytes.NewReader(newMsg))
	req.ContentLength = int64(len(newMsg))
}
