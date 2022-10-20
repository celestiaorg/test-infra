package tests

import (
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
	synclatest "github.com/celestiaorg/test-infra/plans/002-das-benchmarks/tests/sync-latest"
)

// Test-Case #001 - X Light Nodes have finished DASing (from one full/bridge node) before block time
// Description is in docs/test-plans/002-das-benchmark/test-cases
func LightsDasingLatest(runenv *runtime.RunEnv, initCtx *run.InitContext) (err error) {
	switch runenv.StringParam("role") {
	case "validator":
		err = synclatest.RunValidator(runenv, initCtx)
	case "bridge":
		err = synclatest.RunBridgeNode(runenv, initCtx)
	case "light":
		err = synclatest.RunLightNode(runenv, initCtx)
	}

	if err != nil {
		return err
	}

	runenv.RecordSuccess()

	return nil
}
