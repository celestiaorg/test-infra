package tests

import (
	nodesync "github.com/celestiaorg/test-infra/tests/node-sync"

	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
)

func SyncNodes(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	var err error

	vn := runenv.IntParam("validator")
	bn := runenv.IntParam("bridge")
	fn := runenv.IntParam("full")
	ln := runenv.IntParam("light")

	if int(initCtx.GlobalSeq) <= vn {
		err = nodesync.RunAppValidator(runenv, initCtx)
	} else if int(initCtx.GlobalSeq) > vn && int(initCtx.GlobalSeq) <= (vn+bn) {
		err = nodesync.RunBridgeNode(runenv, initCtx)
	} else if int(initCtx.GlobalSeq) > (vn+bn) && int(initCtx.GlobalSeq) <= (vn+bn+fn) {
		err = nodesync.RunFullNode(runenv, initCtx)
	} else if int(initCtx.GlobalSeq) > (vn+bn+fn) && int(initCtx.GlobalSeq) <= (vn+bn+fn+ln) {
		err = nodesync.RunLightNode(runenv, initCtx)
	}

	if err != nil {
		return err
	}
	runenv.RecordSuccess()
	return nil
}
