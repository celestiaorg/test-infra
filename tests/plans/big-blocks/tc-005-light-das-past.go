package bigblocks

import (
	"context"
	"github.com/celestiaorg/test-infra/testkit"
	appsync "github.com/celestiaorg/test-infra/tests/helpers/app-sync"
	nodesync "github.com/celestiaorg/test-infra/tests/helpers/node-sync"
	syncpast "github.com/celestiaorg/test-infra/tests/helpers/sync-past"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
)

// Test-Case #005 - Light nodes are DASing past headers faster than validators produce new ones
// Description is in docs/test-plans/001-Big-Blocks/test-cases
func LightDasPast(runenv *runtime.RunEnv, initCtx *run.InitContext) (err error) {
	switch runenv.StringParam("role") {
	case "seed":
		err = appsync.RunSeed(runenv, initCtx)
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
		runenv.RecordFailure(err)
		initCtx.SyncClient.MustSignalAndWait(context.Background(), testkit.FinishState, runenv.TestInstanceCount)
		return err
	}

	runenv.RecordSuccess()
	return nil
}
