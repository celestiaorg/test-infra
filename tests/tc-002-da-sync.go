package tests

import (
	nodesync "github.com/celestiaorg/test-infra/tests/node-sync"

	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
)

func SyncNodes(runenv *runtime.RunEnv, initCtx *run.InitContext) (err error) {
	vn := runenv.IntParam("validator")
	bn := runenv.IntParam("bridge")
	fn := runenv.IntParam("full")
	ln := runenv.IntParam("light")

	// TODO(@Bidon15): how do we assign LN per BN with non-detirministic assignment of GlobalSeq
	// due to roles and compositions
	switch runenv.StringParam("role") {
	case "validator":
		err = nodesync.RunAppValidator(runenv, initCtx)
	case "bridge":
		err = nodesync.RunBridgeNode(runenv, initCtx)
	case "full":
		err = nodesync.RunFullNode(runenv, initCtx)
	case "light":
		err = nodesync.RunLightNode(runenv, initCtx)
	}

	if err != nil {
		return err
	}
	runenv.RecordSuccess()
	return nil
}
