package plans

import (
	"context"
	"github.com/celestiaorg/test-infra/testkit"
	"github.com/celestiaorg/test-infra/tests/helpers/flood"
	nodesync "github.com/celestiaorg/test-infra/tests/helpers/node-sync"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
)

func SyncNodes(runenv *runtime.RunEnv, initCtx *run.InitContext) (err error) {
	switch runenv.StringParam("role") {
	case "validator":
		err = flood.RunAppValidator(runenv, initCtx)
	case "bridge":
		err = nodesync.RunBridgeNode(runenv, initCtx)
	case "light":
		err = nodesync.RunLightNode(runenv, initCtx)
	}

	if err != nil {
		runenv.RecordFailure(err)
		initCtx.SyncClient.MustSignalAndWait(context.Background(), testkit.FinishState, runenv.TestInstanceCount)
		return err
	}

	runenv.RecordSuccess()
	return nil
}
