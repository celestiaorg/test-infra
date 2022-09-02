/*
Package common is a helper around redundant creation of App or Node part

In order to eliminate the boilerplate code of creating a validators' set,
please use `common.BuildValidator`. This Func does:
- InitChain
- Add-Gen-Account
- Collect-GenTxs
- Add-Persistent-Peers
In addition, the func returns initialized cobra cmd, so you can continue
operating with the validator

appcmd, err := common.BuildValidator(ctx, runenv, initCtx)
appcmd.PayForData(...)
*/
package common
