/*
Package common is a helper around redundant creation of Network, App and Node part

As testground doesn't have native support of uint64 conversion from .toml files,
GetBandwidthValue in network.go is used to adjust the bandwidth per instance
for each of the test-case

In order to eliminate the boilerplate code of creating a validators' set,
please use `common.BuildValidator`. This Func does:
- InitChain
- Add-Gen-Account
- Collect-GenTxs
- Add-Persistent-Peers
In addition, the func returns initialized cobra cmd, so you can continue
operating with the validator

Default: network.LinkShape{
	Bandwidth: common.GetBandwidthValue(runenv.StringParam("bandwidth")),
}

appcmd, err := common.BuildValidator(ctx, runenv, initCtx)
appcmd.PayForData(...)
*/
package common
