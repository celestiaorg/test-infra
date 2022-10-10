package tests

import (
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
)

// Test-Case #001 - X Light Nodes have finished DASing (from one full/bridge node) before block time
// Description is in docs/test-plans/002-DAS-Benchmark/test-cases
func lightDasLatest(runenv *runtime.RunEnv, initCtx *run.InitContext) (err error) {
	switch runenv.StringParam("role") {
	case "bridge":
		// err = syncpast.RunBridgeNode(runenv, initCtx)
	case "full":
		// err = nodesync.RunFullNode(runenv, initCtx)
	case "light":
		// err = syncpast.RunLightNode(runenv, initCtx)
	}

	if err != nil {
		return err
	}

	runenv.RecordSuccess()

	return nil
}
