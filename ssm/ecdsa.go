package ssm

import (
	"encoding/hex"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
)

type ECDSASigner struct{}

func (ecdsa *ECDSASigner) CreateKeyPair() (string, string, string, error) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		log.Error("generate key fail", "err", err)
		return EmptyHexString, EmptyHexString, EmptyHexString, err
	}
	priKeyStr := hex.EncodeToString(crypto.FromECDSA(privateKey))
	pubKeyStr := hex.EncodeToString(crypto.FromECDSAPub(&privateKey.PublicKey))
	compressPubkeyStr := hex.EncodeToString(crypto.CompressPubkey(&privateKey.PublicKey))
	return priKeyStr, pubKeyStr, compressPubkeyStr, nil
}

func (ecdsa *ECDSASigner) SignMessage(privateKey string, txMsg string) (string, error) {
	txHash := common.HexToHash(txMsg)
	privateKeyByte, err := hex.DecodeString(privateKey)
	if err != nil {
		log.Error("decode private key fail", "err", err)
		return EmptyHexString, err
	}
	privKeyEcdsa, err := crypto.ToECDSA(privateKeyByte)
	if err != nil {
		log.Error("Byte private key to ecdsa key fail", "err", err)
		return EmptyHexString, err
	}
	signatureByte, err := crypto.Sign(txHash[:], privKeyEcdsa)
	if err != nil {
		log.Error("sign transaction fail", "err", err)
		return EmptyHexString, err
	}
	return hex.EncodeToString(signatureByte), nil
}

func (ecdsa *ECDSASigner) VerifySignature(publicKey, txHash, signature string) (bool, error) {
	pubKeyBytes, err := hex.DecodeString(publicKey)
	if err != nil {
		log.Error("Error converting public key to bytes", err)
		return false, err
	}
	txHashBytes, err := hex.DecodeString(txHash)
	if err != nil {
		log.Error("Error converting transaction hash to bytes", err)
		return false, err
	}
	sigBytes, err := hex.DecodeString(signature)
	if err != nil {
		log.Error("Error converting signature to bytes", err)
		return false, err
	}
	return crypto.VerifySignature(pubKeyBytes, txHashBytes, sigBytes), nil
}
