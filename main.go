package main

import (
	"github.com/testground/sdk-go/run"
)

var testcases = map[string]interface{}{
	"init-val": run.InitializedTestCaseFn(initVal),
}

func main() {
	run.InvokeMap(testcases)
}
