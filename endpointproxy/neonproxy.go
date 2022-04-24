package endpointproxy

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/celer-network/goutils/log"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

const (
	neonHeaderRpcMethod = "header-rpc-method"
)

type NeonProxy struct {
	neonTargetUrl *url.URL
}

// NewProxy takes target host and creates a reverse proxy
func (c *NeonProxy) startNeonProxy(targetHost string, port int) error {
	var err error
	c.neonTargetUrl, err = url.Parse(targetHost)
	if err != nil {
		return err
	}
	p := httputil.NewSingleHostReverseProxy(c.neonTargetUrl)
	originalDirector := p.Director
	p.Director = func(req *http.Request) {
		originalDirector(req)
		c.modifyNeonRequest(req)
	}
	p.ModifyResponse = modifyNeonResponse()
	mux := http.NewServeMux()
	mux.HandleFunc("/", proxyRequestHandler(p))
	go startCustomProxyByPort(port, mux)
	return nil
}

func (c *NeonProxy) modifyNeonRequest(req *http.Request) {
	req.URL.Scheme = c.neonTargetUrl.Scheme
	req.URL.Host = c.neonTargetUrl.Host
	req.Host = c.neonTargetUrl.Host
	reqStr, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Warnf("invalid neon request err:%s", err.Error())
		return
	}
	var msg jsonrpcMessage
	if err = json.Unmarshal(reqStr, &msg); err != nil {
		log.Warnf("fail to unmarshal this neon req body err:%s", err.Error())
		return
	}
	req.Header.Set(neonHeaderRpcMethod, msg.Method)
	req.Body = ioutil.NopCloser(bytes.NewReader(reqStr))
}

func modifyNeonResponse() func(*http.Response) error {
	return func(resp *http.Response) error {
		if resp.Request != nil && resp.Request.Header.Get(neonHeaderRpcMethod) == MethodEthGetBlockByNumber {
			gzipReader, err := gzip.NewReader(resp.Body)
			if err != nil {
				return err
			}
			originData, err := ioutil.ReadAll(gzipReader)
			if err != nil {
				return err
			}
			var msg jsonrpcMessage
			if err = json.Unmarshal(originData, &msg); err != nil {
				return err
			}
			var result Header
			if err = json.Unmarshal(msg.Result, &result); err != nil {
				return err
			}
			if result.UncleHash == nil {
				result.UncleHash = &types.EmptyUncleHash
			}
			if result.Difficulty == nil {
				result.Difficulty = &hexutil.Big{}
			}
			if result.GasLimit == nil {
				result.GasLimit = new(hexutil.Uint64)
			}
			msg.Result, err = json.Marshal(result)
			if err != nil {
				return err
			}
			newData, err := json.Marshal(msg)
			if err != nil {
				return err
			}
			var b bytes.Buffer
			gz := gzip.NewWriter(&b)
			if _, err = gz.Write(newData); err != nil {
				return err
			}
			if err = gz.Close(); err != nil {
				return err
			}
			resp.Body = ioutil.NopCloser(bytes.NewReader(b.Bytes()))
			resp.ContentLength = int64(len(b.Bytes()))
		}
		return nil
	}
}
