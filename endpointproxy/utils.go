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
	MethodEthCall             = "eth_call"

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

	confluxChainId = 1030

	ontologyChainId = 58

	crabChainId = 44

	platonChainId = 210425

	sxChainId        = 416
	sxTestnetChainId = 647

	godwokenTestnetChainId = 71401
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

var chainIdPortMap = make(map[uint64]int)

func alreadyStarted(chainId uint64, port int) bool {
	curPort, ok := chainIdPortMap[chainId]
	if ok && curPort == port {
		return true
	} else {
		return false
	}
}

// it will use chainId to determined which proxy to launch
func StartProxy(originEndpoint string, chainId uint64, port int) error {
	if alreadyStarted(chainId, port) {
		smallDelay()
		log.Infof("proxy for chain:%d, endpoint:%s, port:%d already started", chainId, originEndpoint, port)
		return nil
	}
	var err error
	switch chainId {
	case godwokenTestnetChainId:
		h := new(GodwokenProxy)
		err = h.startGodwokenProxy(originEndpoint, port)
	case sxChainId, sxTestnetChainId:
		h := new(SxProxy)
		err = h.startSxProxy(originEndpoint, port)
	case platonChainId:
		h := new(PlatonProxy)
		err = h.startPlatonProxy(originEndpoint, port)
	case crabChainId:
		h := new(CrabProxy)
		err = h.startCrabProxy(originEndpoint, port)
	case ontologyChainId:
		h := new(OntologyProxy)
		err = h.startOntologyProxy(originEndpoint, port)
	case confluxChainId:
		h := new(ConfluxProxy)
		err = h.startConfluxProxy(originEndpoint, port)
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
	chainIdPortMap[chainId] = port
	return nil
}

func startCustomProxyByPort(port int, handler http.Handler) {
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), handler)
	if err != nil {
		log.Fatal(err)
	}
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
