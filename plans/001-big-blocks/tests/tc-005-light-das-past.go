package tests

import (
	nodesync "github.com/celestiaorg/test-infra/plans/001-big-blocks/tests/node-sync"
	syncpast "github.com/celestiaorg/test-infra/plans/001-big-blocks/tests/sync-past"

	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
)

// Test-Case #005 - Light nodes are DASing past headers faster than validators produce new ones
// Description is in docs/test-plans/001-Big-Blocks/test-cases
func LightDasPast(runenv *runtime.RunEnv, initCtx *run.InitContext) (err error) {
	switch runenv.StringParam("role") {
	case "validator":
		err = nodesync.RunAppValidator(runenv, initCtx)
	case "bridge":
		err = syncpast.RunBridgeNode(runenv, initCtx)
	case "full":
		err = nodesync.RunFullNode(runenv, initCtx)
	case "light":
		err = syncpast.RunLightNode(runenv, initCtx)
	}

	if err != nil {
		return err
	}
	runenv.RecordSuccess()
	return nil
}
