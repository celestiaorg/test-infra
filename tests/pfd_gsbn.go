package tests

import (
	fundaccounts "github.com/celestiaorg/test-infra/tests/fund-accs"

	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
)

// TODO(@Bidon15): Description
func SubmitPFDandGSBN(runenv *runtime.RunEnv, initCtx *run.InitContext) (err error) {
	switch runenv.StringParam("role") {
	case "validator":
		err = fundaccounts.RunAppValidator(runenv, initCtx)
	case "bridge":
		err = fundaccounts.RunBridgeNode(runenv, initCtx)
		// case "full":
		// 	err = nodesync.RunFullNode(runenv, initCtx)
		// case "light":
		// 	err = syncpast.RunLightNode(runenv, initCtx)
	}

	if err != nil {
		return err
	}
	runenv.RecordSuccess()
	return nil
}
