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
	nervosHeaderRpcMethod = "header-rpc-method"
)

type NervosProxy struct {
	nervosTargetUrl *url.URL
}

// NewProxy takes target host and creates a reverse proxy
func (c *NervosProxy) startNervosProxy(targetHost string, port int) error {
	var err error
	c.nervosTargetUrl, err = url.Parse(targetHost)
	if err != nil {
		return err
	}
	p := httputil.NewSingleHostReverseProxy(c.nervosTargetUrl)
	originalDirector := p.Director
	p.Director = func(req *http.Request) {
		originalDirector(req)
		c.modifyNervosRequest(req)
	}
	p.ModifyResponse = modifyNervosResponse()
	mux := http.NewServeMux()
	mux.HandleFunc("/", proxyRequestHandler(p))
	go startCustomProxyByPort(port, mux)
	return nil
}

func (c *NervosProxy) modifyNervosRequest(req *http.Request) {
	req.URL.Scheme = c.nervosTargetUrl.Scheme
	req.URL.Host = c.nervosTargetUrl.Host
	req.Host = c.nervosTargetUrl.Host
	reqStr, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Warnf("invalid nervos request err:%s", err.Error())
		return
	}
	var msg jsonrpcMessage
	if err = json.Unmarshal(reqStr, &msg); err != nil {
		log.Warnf("fail to unmarshal this nervos req body err:%s", err.Error())
		return
	}
	req.Header.Set(nervosHeaderRpcMethod, msg.Method)
	req.Body = ioutil.NopCloser(bytes.NewReader(reqStr))
}

func modifyNervosResponse() func(*http.Response) error {
	return func(resp *http.Response) error {
		if resp.Request != nil && resp.Request.Header.Get(nervosHeaderRpcMethod) == MethodEthGetBlockByNumber {
			originData, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			newData := strings.Replace(string(originData), ",\"from\":\"0x0000000000000000000000000000000000000000\"", "", 1)
			resp.Body = ioutil.NopCloser(bytes.NewReader([]byte(newData)))
			resp.ContentLength = int64(len([]byte(newData)))
			resp.Header.Set("Content-Length", strconv.Itoa(len(newData)))
		}
		return nil
	}
}
