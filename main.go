package main

import (
	"github.com/celestiaorg/test-infra/tests"
	"github.com/testground/sdk-go/run"
)

var testcases = map[string]interface{}{
	"init-val":  run.InitializedTestCaseFn(tests.InitVal),
	"node-sync": run.InitializedTestCaseFn(tests.SyncNodes),
}

func main() {
	run.InvokeMap(testcases)
}
