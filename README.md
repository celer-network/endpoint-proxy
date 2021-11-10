# endpoint-proxy
Used for some chain which can not support eth-client

As some block-chain miss required field or header which cause error in using eth-client.
This proxy can be used for fill this missed data for such chains.
Now, only specified chain is involved.
You can start a single proxy server or just start a proxy proess in your code which means there is no need to deploy another single proxy server.

##1. start a  proxy server.
-p {port}
-endpoint {remote endpoint}
-cid {chain id}
All the three param is required, and we will use chain id to check which proxy to launch.
```
harmonyChainId            = 1666600000
harmonyTestnetChainId     = 1666700000
celoChainId               = 42220
celoTestnetChainId        = 44787
```
```
cd ./endpointproxy/main
go build
./main -p 10090 -cid 44787 -endpoint https://api.s0.b.hmny.io
```

##2. start a proxy process in your program.
```
import "github.com/celer-network/endpoint-proxy/endpointproxy"

endpointproxy.StartProxy("https://api.s0.b.hmny.io", 1666700000, 10090)
```
