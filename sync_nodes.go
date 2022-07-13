package main

import (
	"github.com/celestiaorg/test-infra/synctest"

	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
)

func syncNodes(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	var err error

	switch in := runenv.TestGroupID; in {
	case "app":
		err = synctest.RunAppValidator(runenv, initCtx)
	case "bridge":
		err = synctest.RunBridgeNode(runenv, initCtx)
		// case "full":
		// 	err = synctest.RunFullNode(runenv, initCtx)
		// case "light":
		// 	err = synctest.RunLightNode(runenv, initCtx)
	}

	if err != nil {
		return err
	}
	return nil
}
