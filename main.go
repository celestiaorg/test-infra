package main

import (
	"github.com/celestiaorg/test-infra/tests/plans/big-blocks"
	"github.com/celestiaorg/test-infra/tests/plans/block-recon"
	"github.com/celestiaorg/test-infra/tests/plans/pfd-gsbn"
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
}

func main() {
	run.InvokeMap(testcases)
}
