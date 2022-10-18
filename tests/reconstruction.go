package tests

import (
	nodesync "github.com/celestiaorg/test-infra/tests/node-sync"
	"github.com/celestiaorg/test-infra/tests/reconstruction"

	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
)

// TODO(@Bidon15): Add description
func BlockReconstruction(runenv *runtime.RunEnv, initCtx *run.InitContext) (err error) {
	switch runenv.StringParam("role") {
	case "validator":
		err = nodesync.RunAppValidator(runenv, initCtx)
	case "bridge":
		err = reconstruction.RunBridgeNode(runenv, initCtx)
	case "full":
		err = reconstruction.RunFullNode(runenv, initCtx)
	case "light":
		err = reconstruction.RunLightNode(runenv, initCtx)
	}

	if err != nil {
		return err
	}
	runenv.RecordSuccess()
	return nil
}
