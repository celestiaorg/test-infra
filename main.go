package main

import (
	"github.com/celestiaorg/test-infra/tests/plans"
	bigblocks "github.com/celestiaorg/test-infra/tests/plans/big-blocks"
	blockrecon "github.com/celestiaorg/test-infra/tests/plans/block-recon"
	blocksync "github.com/celestiaorg/test-infra/tests/plans/block-sync"
	pfdgsbn "github.com/celestiaorg/test-infra/tests/plans/pfd-gsbn"
	"github.com/celestiaorg/test-infra/tests/plans/robusta"
	"github.com/celestiaorg/test-infra/tests/plans/qgb"
	"github.com/testground/sdk-go/run"
)

var testcases = map[string]interface{}{
	// Big Blocks Plan
	"001-val-large-txs":   bigblocks.ValSubmitLargeTxs,
	"002-da-sync":         bigblocks.SyncNodes,
	"003-full-sync-past":  bigblocks.FullSyncPast,
	"004-full-light-past": bigblocks.FullLightSyncPast,
	"005-light-das-past":  bigblocks.LightDasPast,
	// Pay For Blob & Get Shares by Namespace Plan
	// PayForBlobAndGetShares is tracking TestCase key to know
	// when to do shares checker scenario
	"pay-for-blob":            pfdgsbn.PayForBlobAndGetShares,
	"get-shares-by-namespace": pfdgsbn.PayForBlobAndGetShares,
	// Block Reconstruction Plan
	"reconstruction": blockrecon.BlockReconstruction,
	// BlockSync Benchmarks - Syncing Latest
	"blocksync-latest": blocksync.BlockSyncLatest,
	// Robusta Nightly Plan
	"flood-robusta-nightly-1": robusta.RunRobusta,
	"flood-internal":          plans.SyncNodes,
	"qgb-test":       qgb.RunQGB,
}

func main() {
	run.InvokeMap(testcases)
}
