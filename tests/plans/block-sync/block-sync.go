package blocksync

import (
	"context"

	"github.com/celestiaorg/test-infra/testkit"
	"github.com/celestiaorg/test-infra/testkit/optlkit"
	blocksyncbenchhistorical "github.com/celestiaorg/test-infra/tests/helpers/blocksyncbench-historical"
	blocksyncbenchlatest "github.com/celestiaorg/test-infra/tests/helpers/blocksyncbench-latest"
	blocksyncbenchlatestnetpartition "github.com/celestiaorg/test-infra/tests/helpers/blocksyncbench-latest-net-parition"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
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
func BlockSyncLatestWithNetworkPartitions(runenv *runtime.RunEnv, initCtx *run.InitContext) (err error) {
	switch runenv.StringParam("role") {
	case "validator":
		err = blocksyncbenchlatestnetpartition.RunValidator(runenv, initCtx)
	case "bridge":
		err = blocksyncbenchlatestnetpartition.RunBridgeNode(runenv, initCtx)
	case "full":
		err = blocksyncbenchlatestnetpartition.RunFullNode(runenv, initCtx)
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
	optlOpts := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpoint(runenv.StringParam("otel-collector-address")),
		otlpmetrichttp.WithInsecure(),
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	meterProvider, stopFn, err := optlkit.SetupMeter(ctx, initCtx, "BlockSyncHistorical", runenv.StringParam("role"), optlOpts)
	if err != nil {
		runenv.RecordFailure(err)
		return err
	}

	blockSyncMeter := (*meterProvider).Meter("blocksync-historical")

	switch runenv.StringParam("role") {
	case "validator":
		err = blocksyncbenchhistorical.RunValidator(runenv, initCtx, blockSyncMeter)
	case "bridge":
		err = blocksyncbenchhistorical.RunBridgeNode(runenv, initCtx, blockSyncMeter)
	case "full":
		err = blocksyncbenchhistorical.RunFullNode(runenv, initCtx, blockSyncMeter)
	}

	if err != nil {
		runenv.RecordFailure(err)
		initCtx.SyncClient.MustSignalAndWait(context.Background(), testkit.FinishState, runenv.TestInstanceCount)
		return err
	}

	err = stopFn(ctx)
	if err != nil {
		runenv.RecordFailure(err)
		return err
	}

	runenv.RecordSuccess()
	return
}

func BlockSyncHistoricalWithHiccups(runenv *runtime.RunEnv, initCtx *run.InitContext) (err error) {
	return
}
