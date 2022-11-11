package main

import (
	bigblocks "github.com/celestiaorg/test-infra/tests/plans/big-blocks"
	blockrecon "github.com/celestiaorg/test-infra/tests/plans/block-recon"
	dasbenchs "github.com/celestiaorg/test-infra/tests/plans/das-benchmarks"
	pfdgsbn "github.com/celestiaorg/test-infra/tests/plans/pfd-gsbn"
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
	// PayForDataAndGetShares is tracking TestCase key to know
	// when to do shares checker scenario
	"pay-for-data":            run.InitializedTestCaseFn(pfdgsbn.PayForDataAndGetShares),
	"get-shares-by-namespace": run.InitializedTestCaseFn(pfdgsbn.PayForDataAndGetShares),
	// Block Reconstruction Plan
	"reconstruction": run.InitializedTestCaseFn(blockrecon.BlockReconstruction),
	// DAS Benchmarks Plan
	"001-lights-dasing-latest": run.InitializedTestCaseFn(dasbenchs.LightsDasingLatest),
}

func main() {
	run.InvokeMap(testcases)
}
