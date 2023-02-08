package blocksync

import (
	"context"

	"github.com/celestiaorg/test-infra/testkit"
	blocksyncbenchlatest "github.com/celestiaorg/test-infra/tests/helpers/blocksync-bench-latest"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
)

// BlockSyncLatest represents a testcase of 1 validator, 3 bridges, 12 full nodes that
// are trying to sync the latest block from Bridge Nodes and among themselves
// using ShrexSub without falling back on IPLD
// More information under docs/test-plans/005-Block-Sync
func BlockSyncLatest(runenv *runtime.RunEnv, initCtx *run.InitContext) (err error) {
	switch runenv.StringParam("role") {
	case "validator":
		err = blocksyncbenchlatest.RunValidator(runenv, initCtx)
	case "bridge":
		err = blocksyncbenchlatest.RunBridgeNode(runenv, initCtx)
	case "full":
		err = blocksyncbenchlatest.RunFullNode(runenv, initCtx)
	}

	if err != nil {
		runenv.RecordFailure(err)
		initCtx.SyncClient.MustSignalAndWait(context.Background(), testkit.FinishState, runenv.TestInstanceCount)
		return err
	}

	runenv.RecordSuccess()
	return nil
}

func BlockSyncLatestWithHiccups(runenv *runtime.RunEnv, initCtx *run.InitContext) (err error) {
	return
}

func BlockSyncHistorical(runenv *runtime.RunEnv, initCtx *run.InitContext) (err error) {
	return
}

func BlockSyncHistoricalWithHiccups(runenv *runtime.RunEnv, initCtx *run.InitContext) (err error) {
	return
}
