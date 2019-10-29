module github.com/protolambda/go-libp2p-model-net

go 1.13

require (
	github.com/ipfs/go-log v0.0.1
	github.com/jbenet/goprocess v0.1.3
	github.com/libp2p/go-libp2p v0.4.0
	github.com/libp2p/go-libp2p-core v0.2.3
	github.com/libp2p/go-libp2p-netutil v0.1.0
	github.com/libp2p/go-libp2p-peerstore v0.1.3
	github.com/libp2p/go-libp2p-pubsub v0.0.0
	github.com/libp2p/go-libp2p-testing v0.1.0
	github.com/multiformats/go-multiaddr v0.1.1
	gopkg.in/karalabe/cookiejar.v2 v2.0.0-20150724131613-8dcd6a7f4951
)

replace github.com/libp2p/go-libp2p-pubsub => ../proto-pubsub
