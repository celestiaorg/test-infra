package tests

import (
	fundaccounts "github.com/celestiaorg/test-infra/plans/001-big-blocks/tests/fund-accs"

	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
)

// TODO(@Bidon15): Will be change once we have #85 is finished
func SubmitPFD(runenv *runtime.RunEnv, initCtx *run.InitContext) (err error) {
	switch runenv.StringParam("role") {
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
		return err
	}
	runenv.RecordSuccess()
	return nil
}
