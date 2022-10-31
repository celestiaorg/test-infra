package tests

import (
	appsync "github.com/celestiaorg/test-infra/tests/app-sync"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
)

// Test-Case #001 - Validators submit large txs
// Description is in docs/test-plans/001-Big-Blocks/test-cases
func ValSubmitLargeTxs(runenv *runtime.RunEnv, initCtx *run.InitContext) (err error) {
	switch runenv.TestGroupID {
	case "validators":
		err = appsync.RunValidator(runenv, initCtx)
	// we don't have seeds rn. More info in the func
	case "seeds":
		err = appsync.RunSeed(runenv, initCtx)
	}

	return err
}
