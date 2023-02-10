package blocksync

import (
	"context"

	"github.com/celestiaorg/test-infra/testkit"
	blocksyncbenchlatest "github.com/celestiaorg/test-infra/tests/helpers/blocksyncbench-latest"
	blocksyncbenchlatesthiccup "github.com/celestiaorg/test-infra/tests/helpers/blocksyncbench-latest-hiccup"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
)

// BlockSyncLatest represents a testcase of W validator, X bridges, Y full nodes that
// are trying to sync the latest block from Bridge Nodes and among themselves
// using either ShrexGetter only, IPLDGetter only or the default CascadeGetter (_see compositions/cluster-k8s/blocksync-latest/*/*-{getter}.toml)
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

// BlockSyncLatest represents a testcase of W validator, X bridges, Y full nodes that
// are trying to sync the latest block from Bridge Nodes and among themselves
// using either ShrexGetter only, IPLDGetter only or the default CascadeGetter (_see compositions/cluster-k8s/blocksync-latest/*/*-{getter}.toml)
// However with a configuration hiccup height, such that when reached, all full nodes are disconnected
// from all bridge nodes except for a given chosen few (_configurable with the key `full-node-entry-points`)
// More information under docs/test-plans/005-Block-Sync
func BlockSyncLatestWithHiccups(runenv *runtime.RunEnv, initCtx *run.InitContext) (err error) {
	switch runenv.StringParam("role") {
	case "validator":
		err = blocksyncbenchlatesthiccup.RunValidator(runenv, initCtx)
	case "bridge":
		err = blocksyncbenchlatesthiccup.RunBridgeNode(runenv, initCtx)
	case "full":
		err = blocksyncbenchlatesthiccup.RunFullNode(runenv, initCtx)
	}

	if err != nil {
		runenv.RecordFailure(err)
		initCtx.SyncClient.MustSignalAndWait(context.Background(), testkit.FinishState, runenv.TestInstanceCount)
		return err
	}

	runenv.RecordSuccess()
	return
}

func BlockSyncHistorical(runenv *runtime.RunEnv, initCtx *run.InitContext) (err error) {
	return
}

func BlockSyncHistoricalWithHiccups(runenv *runtime.RunEnv, initCtx *run.InitContext) (err error) {
	return
}
