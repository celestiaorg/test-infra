package dasbenchs

import (
	"context"

	"github.com/celestiaorg/test-infra/testkit"
	dasbenchmarks "github.com/celestiaorg/test-infra/tests/helpers/das-benchmarks"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
)

// Test-Case #001 - X Light Nodes have finished DASing (from one full/bridge node) before block time
// Description is in docs/test-plans/002-das-benchmark/test-cases
func LightsDasingLatest(runenv *runtime.RunEnv, initCtx *run.InitContext) (err error) {
	switch runenv.StringParam("role") {
	case "validator":
		err = dasbenchmarks.RunValidator(runenv, initCtx)
	case "bridge":
		err = dasbenchmarks.RunBridgeNode(runenv, initCtx)
	case "light":
		err = dasbenchmarks.RunLightNode(runenv, initCtx)
	}

	if err != nil {
		runenv.RecordFailure(err)
		initCtx.SyncClient.SignalEntry(context.Background(), testkit.FinishState)
		return
	}

	runenv.RecordSuccess()
	return
}
