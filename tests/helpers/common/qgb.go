package common

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"github.com/celestiaorg/test-infra/testkit/qgbkit"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	util "github.com/ipfs/go-ipfs-util"
	crypto2 "github.com/libp2p/go-libp2p/core/crypto"
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

	// import the corresponding evm private key
	evmpkStr := hexutil.Encode(crypto.FromECDSA(evmpk))[2:]
	_, err = cmd.ImportEVMKey("orchestrator", evmpkStr, EVMPrivateKeyPassphrase)
	if err != nil {
		return nil, err
	}

	// import the corresponding p2p private key
	p2ppkRaw, err := p2ppk.Raw()
	if err != nil {
		return nil, err
	}
	_, err = cmd.ImportP2PKey("orchestrator", hexutil.Encode(p2ppkRaw)[2:], P2PPrivateKeyNickname)
	if err != nil {
		return nil, err
	}

	return cmd, nil
}

func BuildRelayer(ctx context.Context, runenv *runtime.RunEnv, initCtx *run.InitContext) (*qgbkit.QGBKit, error) {
	home := "/.relayer"
	runenv.RecordMessage(home)

	privateKeyStr := runenv.StringParam("funded-evm-private-key")
	if privateKeyStr == "" {
		return nil, fmt.Errorf("empty funded evm private key. please add it to configuration")
	}
	evmpk, err := crypto.HexToECDSA(privateKeyStr)
	if err != nil {
		return nil, err
	}

	p2ppk, _, err := crypto2.GenerateEd25519Key(util.NewTimeSeededRand())
	if err != nil {
		return nil, err
	}

	cmd := qgbkit.New(home, &p2ppk, evmpk)

	// init orchestrator store
	_, err = cmd.InitService("relayer")
	if err != nil {
		return nil, err
	}

	// import the corresponding evm private key
	evmpkStr := hexutil.Encode(crypto.FromECDSA(evmpk))[2:]
	_, err = cmd.ImportEVMKey("relayer", evmpkStr, EVMPrivateKeyPassphrase)
	if err != nil {
		return nil, err
	}

	// import the corresponding p2p private key
	p2ppkRaw, err := p2ppk.Raw()
	if err != nil {
		return nil, err
	}
	_, err = cmd.ImportP2PKey("relayer", hexutil.Encode(p2ppkRaw)[2:], P2PPrivateKeyNickname)
	if err != nil {
		return nil, err
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
