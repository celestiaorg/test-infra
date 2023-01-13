package blockrecon

import (
	"context"
	"github.com/celestiaorg/test-infra/testkit"
	appsync "github.com/celestiaorg/test-infra/tests/helpers/app-sync"
	nodesync "github.com/celestiaorg/test-infra/tests/helpers/node-sync"
	reconstruction "github.com/celestiaorg/test-infra/tests/helpers/reconstruction"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
)

// BlockReconstruction represents all test-cases(1/2/3/4 Full Nodes) that
// are trying to reconstruct the latest block from Light Nodes only
// More information under docs/test-plans/004-Block-Reconstruction
func BlockReconstruction(runenv *runtime.RunEnv, initCtx *run.InitContext) (err error) {
	switch runenv.StringParam("role") {
	case "seed":
		err = appsync.RunSeed(runenv, initCtx)
	case "validator":
		err = nodesync.RunAppValidator(runenv, initCtx)
	case "bridge":
		err = reconstruction.RunBridgeNode(runenv, initCtx)
	case "full":
		err = reconstruction.RunFullNode(runenv, initCtx)
	case "light":
		err = reconstruction.RunLightNode(runenv, initCtx)
	}

	if err != nil {
		runenv.RecordFailure(err)
		initCtx.SyncClient.MustSignalAndWait(context.Background(), testkit.FinishState, runenv.TestInstanceCount)
		return err
	}

	runenv.RecordSuccess()
	return nil
}
