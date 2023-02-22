package testkit

import (
	"net"

	"github.com/celestiaorg/test-infra/testkit/appkit"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/testground/sdk-go/sync"
)

// These topics are used around Celestia App instances
var (
	AccountAddressTopic = sync.NewTopic("account-address", "")
	ValidatorPeerTopic  = sync.NewTopic("validator-info", &appkit.ValidatorNode{})
	SeedNodeTopic       = sync.NewTopic("seeds-info", &appkit.ValidatorNode{})
	InitialGenesisTopic = sync.NewTopic("initial-genesis", "")
	GenesisTxTopic      = sync.NewTopic("genesis-tx", "")
	CurlGenesisState    = sync.NewTopic("genesis-state", "")
	BlockHashTopic      = sync.NewTopic("block-hash", "")
)

// AppNodeInfo is needed for creation of Celestia Bridge instances
// Events based on AppNodeTopic are used for pub/sub of AppNodeInfo
type AppNodeInfo struct {
	ID int
	IP net.IP
}

// BridgeNodeInfo is needed for creation of Celestia Full/Light instances
// Events based on BridgeNodeTopic are used for pub/sub of BridgeNodeInfo
type BridgeNodeInfo struct {
	ID          int
	Maddr       string
	TrustedHash string
	AddrInfo    peer.AddrInfo
}

type FullNodeInfo struct {
	ID    int
	Maddr string
}

// These topics are used around Celestia Bridge/Full/Light instances
var (
	BridgeTotalTopic = sync.NewTopic("bridge-amount", 0)
	AppNodeTopic     = sync.NewTopic("app-info", &AppNodeInfo{})
	BridgeNodeTopic  = sync.NewTopic("bridge-info", &BridgeNodeInfo{})
	FullNodeTopic    = sync.NewTopic("full-info", &FullNodeInfo{})
	FundAccountTopic = sync.NewTopic("account-addr", "")
)

// FinishState should be signaled by those, againts which we are testing
var (
	AppStartedState           = sync.State("app-started")
	BridgeStartedState        = sync.State("bridge-started")
	PastBlocksGeneratedState  = sync.State("past-blocks-generated")
	AccountsFundedState       = sync.State("accounts-funded")
	FinishState               = sync.State("test-finished")
	LightNodesStartedState    = sync.State("light-nodes-started")
	ValidatorReadyTopic       = sync.State("validator-ready")
	FullsFinishedSyncingState = sync.State("fulls-finished-syncing")
)
