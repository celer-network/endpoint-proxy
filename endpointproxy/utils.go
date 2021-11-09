package endpointproxy

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"

	"github.com/celer-network/goutils/log"
)

const (
	MethodEthGetCode          = "eth_getCode"
	MethodEthGetBlockByNumber = "eth_getBlockByNumber"

	harmonyChainId        = 1666600000
	harmonyTestnetChainId = 1666700000
	celoChainId           = 42220
	celoTestnetChainId    = 44787
)

// this struct is copied from eth client, so we need to pay attention to the update of eth client
type jsonrpcMessage struct {
	Version string          `json:"jsonrpc,omitempty"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Error   *jsonError      `json:"error,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
}

type jsonError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// it will use chainId to determined which proxy to launch
func StartProxy(originEndpoint string, chainId uint64, port int) error {
	var err error
	switch chainId {
	case harmonyChainId, harmonyTestnetChainId:
		err = startHarmonyProxy(originEndpoint, port)
	case celoChainId, celoTestnetChainId:
		err = startCeloProxy(originEndpoint, port)
	default:
		return fmt.Errorf("do not support proxy for this chain, origin endpoint:%s, chainId:%d", originEndpoint, chainId)
	}
	if err != nil {
		log.Errorf("fail to start this proxy, err:%s", err.Error())
	}
	return err
}

func startCustomProxyByPort(port int, handler http.Handler) error {
	return http.ListenAndServe(fmt.Sprintf(":%d", port), handler)
}

// ProxyRequestHandler handles the http request using proxy
func proxyRequestHandler(proxy *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		proxy.ServeHTTP(w, r)
	}
}
