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

type AstarProxy struct {
	astarTargetUrl *url.URL
}

// NewProxy takes target host and creates a reverse proxy
func (h *AstarProxy) startAstarProxy(targetHost string, port int) error {
	var err error
	h.astarTargetUrl, err = url.Parse(targetHost)
	if err != nil {
		return err
	}
	p := httputil.NewSingleHostReverseProxy(h.astarTargetUrl)
	originalDirector := p.Director
	p.Director = func(req *http.Request) {
		originalDirector(req)
		h.modifyAstarRequest(req)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", proxyRequestHandler(p))
	go startCustomProxyByPort(port, mux)
	return nil
}

func (h *AstarProxy) modifyAstarRequest(req *http.Request) {
	req.URL.Scheme = h.astarTargetUrl.Scheme
	req.URL.Host = h.astarTargetUrl.Host
	req.Host = h.astarTargetUrl.Host
	reqStr, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Errorf("invalid astar request err:%s", err.Error())
		return
	}
	msg := &jsonrpcMessage{}
	if err = json.Unmarshal(reqStr, msg); err != nil {
		log.Errorf("fail to unmarshal this astar req body err:%s", err.Error())
		return
	}
	if msg.Method == MethodEthGetCode {
		newParams := strings.Replace(string(msg.Params), "\"pending\"", "\"latest\"", 1)
		msg.Params = []byte(newParams)
	}
	newMsg, marshalErr := json.Marshal(msg)
	if marshalErr != nil {
		log.Errorf("fail to marshal this new astar req, raw:%s, err:%s", string(newMsg), marshalErr.Error())
		return
	}
	req.Body = ioutil.NopCloser(bytes.NewReader(newMsg))
	req.ContentLength = int64(len(newMsg))
}
