package main

import (
	"github.com/celestiaorg/test-infra/tests"
	"github.com/testground/sdk-go/run"
)

var testcases = map[string]interface{}{
	"001-val-large-txs":   run.InitializedTestCaseFn(tests.ValSubmitLargeTxs),
	"002-da-sync":         run.InitializedTestCaseFn(tests.SyncNodes),
	"003-full-sync-past":  run.InitializedTestCaseFn(tests.FullSyncPast),
	"004-full-light-past": run.InitializedTestCaseFn(tests.FullLightSyncPast),
	"005-light-das-past":  run.InitializedTestCaseFn(tests.LightDasPast),
	"pfd":                 run.InitializedTestCaseFn(tests.SubmitPFD),
}

func main() {
	run.InvokeMap(testcases)
}
