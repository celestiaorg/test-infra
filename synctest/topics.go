package synctest

import (
	"net"

	"github.com/testground/sdk-go/sync"
)

type AppId struct {
	ID int
	IP net.IP
}

type BridgeId struct {
	ID          int
	Maddr       string
	TrustedHash string
	Amount      int
}

var (
	AppNodeTopic    = sync.NewTopic("app-id", &AppId{})
	BridgeNodeTopic = sync.NewTopic("bridge-id", &BridgeId{})
)
