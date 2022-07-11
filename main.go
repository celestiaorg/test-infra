package main

import (
	"github.com/testground/sdk-go/run"
)

var testcases = map[string]interface{}{
	"init-val":  run.InitializedTestCaseFn(initVal),
	"node-sync": run.InitializedTestCaseFn(syncNodes),
}

func main() {
	run.InvokeMap(testcases)
}
