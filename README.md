# go-libp2p-model-net

Based on the [mock-net](https://github.com/libp2p/go-libp2p/tree/master/p2p/net/mock) of go-libp2p.
Extracting/modding the time-components: controlled go-routines, no sleeps, scheduling instead of tickers. that change the simulation results.
This means that the model does not have to run in real time, and can be scaled up with simply more memory: there are no resource stealing or cpu concerns as everything is controlled and scheduled by the model.

However, the go-libp2p stack was never designed for this, so getting pure deterministic behavior is still a work in progress.

The model net also intends to provide an interface for running experiments on it, and implementing actors for each modeled node.

**highly experimental, here be dragons**

## License

MIT, [LICENSE](./LICENSE)

Based on mock-net from go-libp2p, also licensed under MIT, [here](./mocknet/LICENSE)
