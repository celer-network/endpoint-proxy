package endpointproxy

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	"github.com/celer-network/goutils/log"
)

const (
	ontologyHeaderRpcMethod = "header-rpc-method"
)

type OntologyProxy struct {
	ontologyTargetUrl *url.URL
}

// NewProxy takes target host and creates a reverse proxy
func (c *OntologyProxy) startOntologyProxy(targetHost string, port int) error {
	var err error
	c.ontologyTargetUrl, err = url.Parse(targetHost)
	if err != nil {
		return err
	}
	p := httputil.NewSingleHostReverseProxy(c.ontologyTargetUrl)
	originalDirector := p.Director
	p.Director = func(req *http.Request) {
		originalDirector(req)
		c.modifyOntologyRequest(req)
	}
	p.ModifyResponse = modifyOntologyResponse()
	mux := http.NewServeMux()
	mux.HandleFunc("/", proxyRequestHandler(p))
	go startCustomProxyByPort(port, mux)
	return nil
}

func (c *OntologyProxy) modifyOntologyRequest(req *http.Request) {
	req.URL.Scheme = c.ontologyTargetUrl.Scheme
	req.URL.Host = c.ontologyTargetUrl.Host
	req.Host = c.ontologyTargetUrl.Host
	reqStr, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Warnf("invalid ontology request err:%s", err.Error())
		return
	}
	var msg jsonrpcMessage
	if err = json.Unmarshal(reqStr, &msg); err != nil {
		log.Warnf("fail to unmarshal this ontology req body err:%s", err.Error())
		return
	}
	req.Header.Set(ontologyHeaderRpcMethod, msg.Method)
	req.Body = ioutil.NopCloser(bytes.NewReader(reqStr))
}

func modifyOntologyResponse() func(*http.Response) error {
	return func(resp *http.Response) error {
		if resp.Request != nil && resp.Request.Header.Get(ontologyHeaderRpcMethod) == MethodEthGetBlockByNumber {
			originData, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			newData := strings.Replace(string(originData), "\"stateRoot\":\"0x\"", "\"stateRoot\":\"0x0000000000000000000000000000000000000000000000000000000000000000\"", 1)
			resp.Body = ioutil.NopCloser(bytes.NewReader([]byte(newData)))
			resp.ContentLength = int64(len([]byte(newData)))
			resp.Header.Set("Content-Length", strconv.Itoa(len(newData)))
		}
		return nil
	}
}
