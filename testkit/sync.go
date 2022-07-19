package testkit

import (
	"github.com/celestiaorg/test-infra/testkit/appkit"
	"github.com/testground/sdk-go/sync"
)

var (
	AccountAddressTopic   = sync.NewTopic("account-address", "")
	ValidatorPeerTopic    = sync.NewTopic("validator-info", &appkit.ValidatorNode{})
	InitialGenenesisTopic = sync.NewTopic("initial-genesis", "")
	GenesisTxTopic        = sync.NewTopic("genesis-tx", "")
	BlockHashTopic        = sync.NewTopic("block-hash", "")
)
