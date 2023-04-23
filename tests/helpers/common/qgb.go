package common

import (
	"context"
	"crypto/ecdsa"
	"github.com/celestiaorg/test-infra/testkit/qgbkit"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	util "github.com/ipfs/go-ipfs-util"
	crypto2 "github.com/libp2p/go-libp2p/core/crypto"
	"time"

	"github.com/celestiaorg/test-infra/testkit/appkit"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
)

const (
	// EVMPrivateKeyPassphrase the EVM keystore passphrase that will be used when storing all EVM keys
	EVMPrivateKeyPassphrase = "123"
	// P2PPrivateKeyNickname the P2P private key nickname that will be used when storing all P2P keys
	P2PPrivateKeyNickname = "key"
)

func BuildOrchestrator(ctx context.Context, runenv *runtime.RunEnv, initCtx *run.InitContext) (*qgbkit.QGBKit, error) {
	home := "/.orchestrator"
	runenv.RecordMessage(home)

	_, evmpk, err := GenerateEVMAddress()
	if err != nil {
		return nil, err
	}

	p2ppk, _, err := crypto2.GenerateEd25519Key(util.NewTimeSeededRand())
	if err != nil {
		return nil, err
	}

	cmd := qgbkit.New(home, &p2ppk, evmpk)

	// init orchestrator store
	_, err = cmd.InitService("orchestrator")
	if err != nil {
		return nil, err
	}

	runenv.RecordMessage("inside building orch........")
	// import the corresponding evm private key
	evmpkStr := hexutil.Encode(crypto.FromECDSA(evmpk))[2:]
	out, err := cmd.ListEVMKeys("orchestrator")
	if err != nil {
		return nil, err
	}
	runenv.RecordMessage(out)
	_, err = cmd.ImportEVMKey("orchestrator", evmpkStr, EVMPrivateKeyPassphrase)
	if err != nil {
		return nil, err
	}
	runenv.RecordMessage("after importing EVM key........")

	// import the corresponding p2p private key
	p2ppkRaw, err := p2ppk.Raw()
	if err != nil {
		return nil, err
	}
	_, err = cmd.ImportP2PKey("orchestrator", hexutil.Encode(p2ppkRaw)[2:], P2PPrivateKeyNickname)
	if err != nil {
		return nil, err
	}

	time.Sleep(30 * time.Second)

	return cmd, nil
}

func BuildValidatorWithEVMAddress(ctx context.Context, runenv *runtime.RunEnv, initCtx *run.InitContext, evmAddress *common.Address) (*appkit.AppKit, error) {
	home := "/.celestia-app"
	runenv.RecordMessage(home)

	cmd, keyringName, accAddr, err := InitChainAndMaybeBroadcastGenesis(ctx, runenv, initCtx, home)
	if err != nil {
		return nil, err
	}

	runenv.RecordMessage("Validator is signing its own GenTx")
	_, err = cmd.SignGenTxWithEVMAddress(keyringName, "5000000000utia", "test", home, evmAddress)
	if err != nil {
		return nil, err
	}

	err = BroadcastAndCollectGenTx(ctx, home, accAddr, initCtx, runenv, cmd)
	if err != nil {
		return nil, err
	}

	ip, err := UpdateAndPublishConfig(ctx, home, cmd, initCtx)
	if err != nil {
		return nil, err
	}

	if runenv.IntParam("validator") > 1 {
		err := DiscoverPeers(ctx, home, ip, initCtx, runenv)
		if err != nil {
			return nil, err
		}
	}

	return cmd, nil
}

func GenerateEVMAddress() (*common.Address, *ecdsa.PrivateKey, error) {
	ethPrivateKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, nil, err
	}

	orchEthPublicKey := ethPrivateKey.Public().(*ecdsa.PublicKey)
	evmAddr := crypto.PubkeyToAddress(*orchEthPublicKey)

	return &evmAddr, ethPrivateKey, nil
}

func ECDSAToAddress(ethPrivateKey *ecdsa.PrivateKey) *common.Address {
	orchEthPublicKey := ethPrivateKey.Public().(*ecdsa.PublicKey)
	evmAddr := crypto.PubkeyToAddress(*orchEthPublicKey)

	return &evmAddr
}
