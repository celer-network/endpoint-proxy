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
	"github.com/ethereum/go-ethereum/core/types"
)

const (
	zkSyncHeaderRpcMethod = "header-rpc-method"
)

type ZkSyncProxy struct {
	zkSyncTargetUrl *url.URL
}

// NewProxy takes target host and creates a reverse proxy
func (c *ZkSyncProxy) startZkSyncProxy(targetHost string, port int, chainId uint64) error {
	var err error
	c.zkSyncTargetUrl, err = url.Parse(targetHost)
	if err != nil {
		return err
	}
	p := httputil.NewSingleHostReverseProxy(c.zkSyncTargetUrl)
	originalDirector := p.Director
	p.Director = func(req *http.Request) {
		originalDirector(req)
		c.modifyZkSyncRequest(req)
	}
	p.ModifyResponse = modifyZkSyncResponse()
	mux := http.NewServeMux()
	mux.HandleFunc("/", proxyRequestHandler(p))
	go startCustomProxyByPort(port, mux, chainId, targetHost)
	return nil
}

func (c *ZkSyncProxy) modifyZkSyncRequest(req *http.Request) {
	req.URL.Scheme = c.zkSyncTargetUrl.Scheme
	req.URL.Host = c.zkSyncTargetUrl.Host
	req.Host = c.zkSyncTargetUrl.Host
	reqStr, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Warnf("invalid zkSync request err:%s", err.Error())
		return
	}
	var msg jsonrpcMessage
	if err = json.Unmarshal(reqStr, &msg); err != nil {
		log.Warnf("fail to unmarshal this zkSync req body err:%s", err.Error())
		return
	}
	req.Header.Set(zkSyncHeaderRpcMethod, msg.Method)
	req.Body = ioutil.NopCloser(bytes.NewReader(reqStr))
}

func modifyZkSyncResponse() func(*http.Response) error {
	return func(resp *http.Response) error {
		if resp.Request != nil && resp.Request.Header.Get(zkSyncHeaderRpcMethod) == MethodEthGetBlockByNumber {
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
			if result.Bloom == nil {
				result.Bloom = &types.Bloom{}
			}
			/*if result.Difficulty == nil {
				result.Difficulty = &hexutil.Big{}
			}*/
			result.BaseFee = nil

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
