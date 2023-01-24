package pfdgsbn

import (
	"context"
	"github.com/celestiaorg/test-infra/testkit"
	appsync "github.com/celestiaorg/test-infra/tests/helpers/app-sync"
	fundaccounts "github.com/celestiaorg/test-infra/tests/helpers/fund-accs"

	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
)

// PayForBlobAndGetShares func is a combination of 2 test-cases, where we want to
// TC-1: Do pay for data only
// TC-2: Do pay for data and get the shares to verify against the pushed data
// in each of the RunXXX method, we are tracking runenv.TestCase to see when to kick-in
// GetSharesByNamespace Checker
func PayForBlobAndGetShares(runenv *runtime.RunEnv, initCtx *run.InitContext) (err error) {
	switch runenv.StringParam("role") {
	case "seed":
		err = appsync.RunSeed(runenv, initCtx)
	case "validator":
		err = fundaccounts.RunAppValidator(runenv, initCtx)
	case "bridge":
		err = fundaccounts.RunBridgeNode(runenv, initCtx)
	case "full":
		err = fundaccounts.RunFullNode(runenv, initCtx)
	case "light":
		err = fundaccounts.RunLightNode(runenv, initCtx)
	}

	if err != nil {
		runenv.RecordFailure(err)
		initCtx.SyncClient.MustSignalAndWait(context.Background(), testkit.FinishState, runenv.TestInstanceCount)
		return err
	}

	runenv.RecordSuccess()
	return nil
}
