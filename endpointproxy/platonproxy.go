package endpointproxy

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	"github.com/celer-network/goutils/log"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

const (
	platonHeaderRpcMethod = "header-rpc-method"
)

type PlatonProxy struct {
	platonTargetUrl *url.URL
}

// NewProxy takes target host and creates a reverse proxy
func (c *PlatonProxy) startPlatonProxy(targetHost string, port int) error {
	var err error
	c.platonTargetUrl, err = url.Parse(targetHost)
	if err != nil {
		return err
	}
	p := httputil.NewSingleHostReverseProxy(c.platonTargetUrl)
	originalDirector := p.Director
	p.Director = func(req *http.Request) {
		originalDirector(req)
		c.modifyPlatonRequest(req)
	}
	p.ModifyResponse = modifyPlatonResponse()
	mux := http.NewServeMux()
	mux.HandleFunc("/", proxyRequestHandler(p))
	go startCustomProxyByPort(port, mux)
	return nil
}

func (c *PlatonProxy) modifyPlatonRequest(req *http.Request) {
	req.URL.Scheme = c.platonTargetUrl.Scheme
	req.URL.Host = c.platonTargetUrl.Host
	req.Host = c.platonTargetUrl.Host
	req.URL.Path = strings.TrimRight(req.URL.Path, "/")
	reqStr, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Warnf("invalid platon request err:%s", err.Error())
		return
	}
	var msg jsonrpcMessage
	if err = json.Unmarshal(reqStr, &msg); err != nil {
		log.Warnf("fail to unmarshal this platon req body err:%s", err.Error())
		return
	}
	req.Header.Set(platonHeaderRpcMethod, msg.Method)
	req.Body = ioutil.NopCloser(bytes.NewReader(reqStr))
}

func modifyPlatonResponse() func(*http.Response) error {
	return func(resp *http.Response) error {
		if resp.Request != nil && resp.Request.Header.Get(platonHeaderRpcMethod) == MethodEthGetBlockByNumber {
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
			resp.Header.Set("Content-Length", strconv.Itoa(len(b.Bytes())))
		}
		return nil
	}
}
