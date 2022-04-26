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

type NervosProxy struct {
	nervosTargetUrl *url.URL
}

// NewProxy takes target host and creates a reverse proxy
func (h *NervosProxy) startNervosProxy(targetHost string, port int) error {
	var err error
	h.nervosTargetUrl, err = url.Parse(targetHost)
	if err != nil {
		return err
	}
	p := httputil.NewSingleHostReverseProxy(h.nervosTargetUrl)
	originalDirector := p.Director
	p.Director = func(req *http.Request) {
		originalDirector(req)
		h.modifyNervosRequest(req)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", proxyRequestHandler(p))
	go startCustomProxyByPort(port, mux)
	return nil
}

func (h *NervosProxy) modifyNervosRequest(req *http.Request) {
	req.URL.Scheme = h.nervosTargetUrl.Scheme
	req.URL.Host = h.nervosTargetUrl.Host
	req.Host = h.nervosTargetUrl.Host
	reqStr, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Errorf("invalid nervos request err:%s", err.Error())
		return
	}
	msg := &jsonrpcMessage{}
	if err = json.Unmarshal(reqStr, msg); err != nil {
		log.Errorf("fail to unmarshal this nervos req body err:%s", err.Error())
		return
	}
	if msg.Method == MethodEthCall {
		newParams := strings.Replace(string(msg.Params), ",\"from\":\"0x0000000000000000000000000000000000000000\"", "", 1)
		msg.Params = []byte(newParams)
	}
	newMsg, marshalErr := json.Marshal(msg)
	if marshalErr != nil {
		log.Errorf("fail to marshal this new nervos req, raw:%s, err:%s", string(newMsg), marshalErr.Error())
		return
	}
	req.Body = ioutil.NopCloser(bytes.NewReader(newMsg))
	req.ContentLength = int64(len(newMsg))
}
