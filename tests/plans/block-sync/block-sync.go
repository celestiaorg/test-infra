package blocksync

import (
	"context"

	"github.com/celestiaorg/test-infra/testkit"
	blocksynchistoricalnetpartitions "github.com/celestiaorg/test-infra/tests/helpers/block-sync/historical-net-partitions"
	blocksynchistorical "github.com/celestiaorg/test-infra/tests/helpers/block-sync/historical"
	blocksynclatest "github.com/celestiaorg/test-infra/tests/helpers/block-sync/latest"
	blocksynclatestnetpartitions "github.com/celestiaorg/test-infra/tests/helpers/block-sync/latest-net-partitions"
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
		err = blocksynclatest.RunValidator(runenv, initCtx)
	case "bridge":
		err = blocksynclatest.RunBridgeNode(runenv, initCtx)
	case "full":
		err = blocksynclatest.RunFullNode(runenv, initCtx)
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
func BlockSyncLatestWithNetworkPartitions(runenv *runtime.RunEnv, initCtx *run.InitContext) (err error) {
	switch runenv.StringParam("role") {
	case "validator":
		err = blocksynclatestnetpartitions.RunValidator(runenv, initCtx)
	case "bridge":
		err = blocksynclatestnetpartitions.RunBridgeNode(runenv, initCtx)
	case "full":
		err = blocksynclatestnetpartitions.RunFullNode(runenv, initCtx)
	}

	if err != nil {
		runenv.RecordFailure(err)
		initCtx.SyncClient.MustSignalAndWait(context.Background(), testkit.FinishState, runenv.TestInstanceCount)
		return err
	}

	runenv.RecordSuccess()
	return
}

// BlockSyncHistorical represents a testcase of W validator, X bridges, Y full nodes that
// are trying to sync historical blocks from Bridge Nodes and among themselves
// using either IPLDGetter only or the default CascadeGetter (_see compositions/cluster-k8s/blocksync-historical/*/*-{getter}.toml)
// More information under docs/test-plans/005-Block-Sync
func BlockSyncHistorical(runenv *runtime.RunEnv, initCtx *run.InitContext) (err error) {
	switch runenv.StringParam("role") {
	case "validator":
		err = blocksynchistorical.RunValidator(runenv, initCtx)
	case "bridge":
		err = blocksynchistorical.RunBridgeNode(runenv, initCtx)
	case "full":
		err = blocksynchistorical.RunFullNode(runenv, initCtx)
	case "historical-full":
		err = blocksynchistorical.RunHistoricalFullNode(runenv, initCtx)
	}

	if err != nil {
		runenv.RecordFailure(err)
		initCtx.SyncClient.MustSignalAndWait(context.Background(), testkit.FinishState, runenv.TestInstanceCount)
		return err
	}

	runenv.RecordSuccess()
	return
}

func BlockSyncHistoricalWithNetworkPartitions(runenv *runtime.RunEnv, initCtx *run.InitContext) (err error) {
	switch runenv.StringParam("role") {
	case "validator":
		err = blocksynchistoricalnetpartitions.RunValidator(runenv, initCtx)
	case "bridge":
		err = blocksynchistoricalnetpartitions.RunBridgeNode(runenv, initCtx)
	case "full":
		err = blocksynchistoricalnetpartitions.RunFullNode(runenv, initCtx)
	case "historical-full":
		err = blocksynchistoricalnetpartitions.RunHistoricalFullNode(runenv, initCtx)
	}

	if err != nil {
		runenv.RecordFailure(err)
		initCtx.SyncClient.MustSignalAndWait(context.Background(), testkit.FinishState, runenv.TestInstanceCount)
		return err
	}

	runenv.RecordSuccess()
	return
}
