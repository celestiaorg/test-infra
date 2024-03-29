package qgb

import (
	"context"
	"github.com/celestiaorg/test-infra/testkit"
	appsync "github.com/celestiaorg/test-infra/tests/helpers/app-sync"
	qgbsync "github.com/celestiaorg/test-infra/tests/helpers/qgb-sync"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
	"time"
)

// RunQGB Runs a QGB network with a relayer relaying to the network specified in config.
func RunQGB(runenv *runtime.RunEnv, initCtx *run.InitContext) (err error) {
	switch runenv.TestGroupID {
	case "orchestrators":
		err = qgbsync.RunValidatorWithOrchestrator(runenv, initCtx)
	case "relayers":
		err = qgbsync.RunValidatorWithRelayer(runenv, initCtx)
	case "seeds":
		err = appsync.RunSeed(runenv, initCtx)
	}

	if err != nil {
		runenv.RecordFailure(err)
		initCtx.SyncClient.MustSignalAndWait(context.Background(), testkit.FinishState, runenv.TestInstanceCount)
		return err
	}

	runenv.RecordSuccess()
	time.Sleep(15 * time.Minute)
	return err
}
