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

type OntologyProxy struct {
	ontologyTargetUrl *url.URL
}

// NewProxy takes target host and creates a reverse proxy
func (h *OntologyProxy) startOntologyProxy(targetHost string, port int) error {
	var err error
	h.ontologyTargetUrl, err = url.Parse(targetHost)
	if err != nil {
		return err
	}
	p := httputil.NewSingleHostReverseProxy(h.ontologyTargetUrl)
	originalDirector := p.Director
	p.Director = func(req *http.Request) {
		originalDirector(req)
		h.modifyOntologyRequest(req)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", proxyRequestHandler(p))
	go startCustomProxyByPort(port, mux)
	return nil
}

func (h *OntologyProxy) modifyOntologyRequest(req *http.Request) {
	req.URL.Scheme = h.ontologyTargetUrl.Scheme
	req.URL.Host = h.ontologyTargetUrl.Host
	req.Host = h.ontologyTargetUrl.Host
	reqStr, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Errorf("invalid ontology request err:%s", err.Error())
		return
	}
	msg := &jsonrpcMessage{}
	if err = json.Unmarshal(reqStr, msg); err != nil {
		log.Errorf("fail to unmarshal this ontology req body err:%s", err.Error())
		return
	}
	if msg.Method == MethodEthGetBlockByNumber {
		newParams := strings.Replace(string(msg.Params), "\"stateRoot\":\"0x\"", "\"stateRoot\":\"0x0000000000000000000000000000000000000000000000000000000000000000\"", 1)
		msg.Params = []byte(newParams)
	}
	newMsg, marshalErr := json.Marshal(msg)
	if marshalErr != nil {
		log.Errorf("fail to marshal this new ontology req, raw:%s, err:%s", string(newMsg), marshalErr.Error())
		return
	}
	req.Body = ioutil.NopCloser(bytes.NewReader(newMsg))
	req.ContentLength = int64(len(newMsg))
}
