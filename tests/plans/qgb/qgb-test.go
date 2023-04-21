package qgb

import (
	"context"
	"github.com/celestiaorg/test-infra/testkit"
	appsync2 "github.com/celestiaorg/test-infra/tests/helpers/qgb-sync"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
)

// Test-Case #001 - Validators submit large txs
// Description is in docs/test-plans/001-Big-Blocks/test-cases
func RunQGB(runenv *runtime.RunEnv, initCtx *run.InitContext) (err error) {
	switch runenv.TestGroupID {
	case "orchestrators":
		err = appsync2.RunValidatorWithOrchestrator(runenv, initCtx)
	}

	if err != nil {
		runenv.RecordFailure(err)
		initCtx.SyncClient.MustSignalAndWait(context.Background(), testkit.FinishState, runenv.TestInstanceCount)
		return err
	}

	runenv.RecordSuccess()
	return err
}
