package endpointproxy

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/celer-network/goutils/log"
)

const (
	MethodEthGetCode          = "eth_getCode"
	MethodEthGetBlockByNumber = "eth_getBlockByNumber"

	astarChainId  = 592
	shidenChainId = 336

	harmonyChainId        = 1666600000
	harmonyTestnetChainId = 1666700000

	celoChainId        = 42220
	celoTestnetChainId = 44787

	acalaChainId        = 787
	acalaTestnetChainId = 595

	cloverChainId        = 1024
	cloverTestnetChainId = 1023
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
	case astarChainId, shidenChainId:
		h := new(AstarProxy)
		err = h.startAstarProxy(originEndpoint, port)
	case acalaTestnetChainId, acalaChainId:
		h := new(AcalaProxy)
		err = h.startAcalaProxy(originEndpoint, port)
	case cloverChainId, cloverTestnetChainId:
		h := new(CloverProxy)
		err = h.startCloverProxy(originEndpoint, port)
	case harmonyChainId, harmonyTestnetChainId:
		h := new(HarmonyProxy)
		err = h.startHarmonyProxy(originEndpoint, port)
	case celoChainId, celoTestnetChainId:
		c := new(CeloProxy)
		err = c.startCeloProxy(originEndpoint, port)
	default:
		return fmt.Errorf("do not support proxy for this chain, origin endpoint:%s, chainId:%d", originEndpoint, chainId)
	}
	if err != nil {
		log.Errorf("fail to start this proxy, err:%s", err.Error())
		return err
	}
	smallDelay()
	log.Infof("start proxy for chain:%d, endpoint:%s, port:%d", chainId, originEndpoint, port)
	return nil
}

func startCustomProxyByPort(port int, handler http.Handler) {
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), handler)
	if err != nil {
		log.Fatal(err)
	}
	return
}

// ProxyRequestHandler handles the http request using proxy
func proxyRequestHandler(proxy *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		proxy.ServeHTTP(w, r)
	}
}

func smallDelay() {
	time.Sleep(100 * time.Millisecond)
}
