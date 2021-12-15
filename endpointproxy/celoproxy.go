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
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

const (
	celoHeaderRpcMethod = "header-rpc-method"
)

var (
	celoTargetUrl *url.URL
)

// NewProxy takes target host and creates a reverse proxy
func startCeloProxy(targetHost string, port int) error {
	var err error
	celoTargetUrl, err = url.Parse(targetHost)
	if err != nil {
		return err
	}
	p := httputil.NewSingleHostReverseProxy(celoTargetUrl)
	originalDirector := p.Director
	p.Director = func(req *http.Request) {
		originalDirector(req)
		modifyCeloRequest(req)
	}
	p.ModifyResponse = modifyCeloResponse()
	mux := http.NewServeMux()
	mux.HandleFunc("/", proxyRequestHandler(p))
	go startCustomProxyByPort(port, mux)
	return nil
}

func modifyCeloRequest(req *http.Request) {
	req.URL.Scheme = celoTargetUrl.Scheme
	req.URL.Host = celoTargetUrl.Host
	req.Host = celoTargetUrl.Host
	reqStr, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Warnf("invalid celo request err:%s", err.Error())
		return
	}
	var msg jsonrpcMessage
	if err = json.Unmarshal(reqStr, &msg); err != nil {
		log.Warnf("fail to unmarshal this celo req body err:%s", err.Error())
		return
	}
	req.Header.Set(celoHeaderRpcMethod, msg.Method)
	req.Body = ioutil.NopCloser(bytes.NewReader(reqStr))
}

func modifyCeloResponse() func(*http.Response) error {
	return func(resp *http.Response) error {
		if resp.Request != nil && resp.Request.Header.Get(celoHeaderRpcMethod) == MethodEthGetBlockByNumber {
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
				emptyHash := types.EmptyUncleHash
				result.UncleHash = &emptyHash
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

// from eth client
type Header struct {
	ParentHash  *common.Hash      `json:"parentHash"       gencodec:"required"`
	UncleHash   *common.Hash      `json:"sha3Uncles"       gencodec:"required"`
	Coinbase    *common.Address   `json:"miner"            gencodec:"required"`
	Root        *common.Hash      `json:"stateRoot"        gencodec:"required"`
	TxHash      *common.Hash      `json:"transactionsRoot" gencodec:"required"`
	ReceiptHash *common.Hash      `json:"receiptsRoot"     gencodec:"required"`
	Bloom       *types.Bloom      `json:"logsBloom"        gencodec:"required"`
	Difficulty  *hexutil.Big      `json:"difficulty"       gencodec:"required"`
	Number      *hexutil.Big      `json:"number"           gencodec:"required"`
	GasLimit    *hexutil.Uint64   `json:"gasLimit"         gencodec:"required"`
	GasUsed     *hexutil.Uint64   `json:"gasUsed"          gencodec:"required"`
	Time        *hexutil.Uint64   `json:"timestamp"        gencodec:"required"`
	Extra       *hexutil.Bytes    `json:"extraData"        gencodec:"required"`
	MixDigest   *common.Hash      `json:"mixHash"`
	Nonce       *types.BlockNonce `json:"nonce"`
	BaseFee     *hexutil.Big      `json:"baseFeePerGas" rlp:"optional"`
}
