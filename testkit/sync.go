package testkit

import "github.com/testground/sdk-go/sync"

var (
	AccountAddressTopic   = sync.NewTopic("account-address", "")
	InitialGenenesisTopic = sync.NewTopic("initial-genesis", "")
	GenesisTxTopic        = sync.NewTopic("genesis-tx", "")
	BlockHashTopic        = sync.NewTopic("block-hash", "")
)
