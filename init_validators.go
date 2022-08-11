package main

import (
	appsync "github.com/celestiaorg/test-infra/app-sync"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
)

// This test-case is a 101 on how Celestia-App only should be started.
// In this test-case, we are testing the following scenario:
// 1. Every instance can create an account
// 2. The orchestrator(described more below) funds the account at genesis
//    and sends the initial genesis.json to the rest of the validators' set
// 3. After receiving the initial genesis.json, validators are signing the
//    genesis transaction(gentx)
// 4. Validators collects all genesis transactions
// 5. The chain is started
func initVal(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	var err error

	switch runenv.TestGroupID {
	case "validators":
		err = appsync.RunValidator(runenv, initCtx)
	case "seeds":
		err = appsync.RunSeed(runenv, initCtx)
	}

	return err
}
