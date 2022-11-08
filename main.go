package main

import (
	"github.com/celestiaorg/test-infra/tests"
	"github.com/celestiaorg/test-infra/tests/plans/big-blocks"
	"github.com/celestiaorg/test-infra/tests/plans/block-recon"
	"github.com/testground/sdk-go/run"
)

var testcases = map[string]interface{}{
	// Big Blocks Plan
	"001-val-large-txs":   run.InitializedTestCaseFn(bigblocks.ValSubmitLargeTxs),
	"002-da-sync":         run.InitializedTestCaseFn(bigblocks.SyncNodes),
	"003-full-sync-past":  run.InitializedTestCaseFn(bigblocks.FullSyncPast),
	"004-full-light-past": run.InitializedTestCaseFn(bigblocks.FullLightSyncPast),
	"005-light-das-past":  run.InitializedTestCaseFn(bigblocks.LightDasPast),
	// Pay For Data & Get Shares by Namespace Plan
	"pfd": run.InitializedTestCaseFn(tests.SubmitPFD),
	// Block Reconstruction Plan
	"reconstruction": run.InitializedTestCaseFn(blockrecon.BlockReconstruction),
}

func main() {
	run.InvokeMap(testcases)
}
