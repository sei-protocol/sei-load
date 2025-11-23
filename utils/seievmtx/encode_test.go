package seievmtx

import (
	"bytes"
	"math/big"
	"testing"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestEncodeCosmosTxFromEthTx_Legacy(t *testing.T) {
	setupBech32(t)

	priv, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	fromAddr := crypto.PubkeyToAddress(priv.PublicKey)

	chainID := big.NewInt(1337)
	nonce := uint64(1)
	to := fromAddr
	value := big.NewInt(1234567890)
	gasLimit := uint64(21000)
	gasPrice := big.NewInt(2_000_000_000)
	data := []byte("hello-sei")

	legacy := &ethtypes.LegacyTx{
		Nonce:    nonce,
		To:       &to,
		Value:    value,
		Gas:      gasLimit,
		GasPrice: gasPrice,
		Data:     data,
	}
	ethTx := ethtypes.NewTx(legacy)
	signedEthTx, err := ethtypes.SignTx(ethTx, ethtypes.LatestSignerForChainID(chainID), priv)
	if err != nil {
		t.Fatalf("failed to sign tx: %v", err)
	}

	bz, err := EncodeCosmosTxFromEthTx(signedEthTx)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}
	if len(bz) == 0 {
		t.Fatalf("empty encoded tx bytes")
	}

	txCfg := DefaultTxConfig()
	sdkTx, err := txCfg.TxDecoder()(bz)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	msg := MustGetEVMTransactionMessage(sdkTx)
	recoveredEthTx, _ := msg.AsTransaction()
	if recoveredEthTx == nil {
		t.Fatalf("failed to reconstruct ethereum tx from message")
	}

	if signedEthTx.Hash() != recoveredEthTx.Hash() {
		t.Fatalf("hash mismatch: have %s want %s", recoveredEthTx.Hash(), signedEthTx.Hash())
	}
	if !bytes.Equal(signedEthTx.Data(), recoveredEthTx.Data()) {
		t.Fatalf("data mismatch")
	}
}

func TestEncodeCosmosTxFromEthTx_DynamicFee(t *testing.T) {
	setupBech32(t)

	priv, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	chainID := big.NewInt(900)
	nonce := uint64(0)
	to := crypto.PubkeyToAddress(priv.PublicKey)
	value := big.NewInt(42)
	gasLimit := uint64(50_000)
	maxFee := big.NewInt(30_000_000_000)
	maxPriority := big.NewInt(1_500_000_000)
	data := []byte{0xde, 0xad, 0xbe, 0xef}

	dyn := &ethtypes.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		GasTipCap: maxPriority,
		GasFeeCap: maxFee,
		Gas:       gasLimit,
		To:        &to,
		Value:     value,
		Data:      data,
	}
	ethTx := ethtypes.NewTx(dyn)
	signedEthTx, err := ethtypes.SignTx(ethTx, ethtypes.LatestSignerForChainID(chainID), priv)
	if err != nil {
		t.Fatalf("failed to sign tx: %v", err)
	}

	bz, err := EncodeCosmosTxFromEthTx(signedEthTx)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}
	if len(bz) == 0 {
		t.Fatalf("empty encoded tx bytes")
	}

	txCfg := DefaultTxConfig()
	sdkTx, err := txCfg.TxDecoder()(bz)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	msg := MustGetEVMTransactionMessage(sdkTx)
	recoveredEthTx, _ := msg.AsTransaction()
	if recoveredEthTx == nil {
		t.Fatalf("failed to reconstruct ethereum tx from message")
	}

	if signedEthTx.Hash() != recoveredEthTx.Hash() {
		t.Fatalf("hash mismatch: have %s want %s", recoveredEthTx.Hash(), signedEthTx.Hash())
	}
}

func setupBech32(t *testing.T) {}
