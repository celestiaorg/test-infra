package main

import (
	"github.com/celestiaorg/test-infra/plans/002-das-benchmarks/tests"
	"github.com/testground/sdk-go/run"
)

var testcases = map[string]interface{}{
	"001-lights-dasing-latest":   run.InitializedTestCaseFn(tests.LightsDasingLatest),
}

func main() {
	run.InvokeMap(testcases)
}

