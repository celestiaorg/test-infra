package main

import (
	"github.com/celestiaorg/test-infra/tests"
	"github.com/testground/sdk-go/run"
)

var testcases = map[string]interface{}{
	"001-val-large-txs": run.InitializedTestCaseFn(tests.ValSubmitLargeTxs),
	"node-sync":         run.InitializedTestCaseFn(tests.SyncNodes),
}

func main() {
	run.InvokeMap(testcases)
}
