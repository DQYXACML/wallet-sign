package ethereum

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/DQYXACML/wallet-sign/chain"
	"github.com/DQYXACML/wallet-sign/leveldb"
	"github.com/DQYXACML/wallet-sign/protobuf/wallet"
	"github.com/DQYXACML/wallet-sign/ssm"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"math/big"
)

const ChainName = "Ethereum"

type ChainAdaptor struct {
	db     *leveldb.Keys
	signer ssm.Signer
}

func (c *ChainAdaptor) SignTransactionMessage(ctx context.Context, req *wallet.SignTransactionMessageRequest) (*wallet.SignTransactionMessageResponse, error) {
	resp := &wallet.SignTransactionMessageResponse{
		Code: wallet.ReturnCode_ERROR,
	}
	privKey, isOk := c.db.GetPrivKey(req.PublicKey)
	if !isOk {
		return nil, errors.New("get private key by public key fail")
	}
	signature, err := c.signer.SignMessage(privKey, req.MessageHash)
	if err != nil {
		log.Error("sign message fail", "err", err)
		return nil, err
	}
	resp.Message = "sign tx message success"
	resp.Signature = signature
	resp.Code = wallet.ReturnCode_SUCCESS
	return resp, nil
}

func (c *ChainAdaptor) GetChainSignMethod(ctx context.Context, req *wallet.GetChainSignMethodRequest) (*wallet.GetChainSignMethodResponse, error) {
	return &wallet.GetChainSignMethodResponse{
		Code:       wallet.ReturnCode_SUCCESS,
		Msg:        "get sign method success",
		SignMethod: "ecdsa",
	}, nil
}

func (c *ChainAdaptor) GetChainSchema(ctx context.Context, req *wallet.GetChainSchemaRequest) (*wallet.GetChainSchemaResponse, error) {
	es := EthereumSchema{
		RequestId: "0",
		DynamicFeeTx: Eip1559DynamicFeeTx{
			ChainId:              "",
			Nonce:                0,
			FromAddress:          common.Address{}.String(),
			ToAddress:            common.Address{}.String(),
			GasLimit:             0,
			Gas:                  0,
			MaxFeePerGas:         "0",
			MaxPriorityFeePerGas: "0",
			Amount:               "0",
			ContractAddress:      "",
		},
		ClassicFeeTx: LegacyFeeTx{
			ChainId:         "0",
			Nonce:           0,
			FromAddress:     common.Address{}.String(),
			ToAddress:       common.Address{}.String(),
			GasLimit:        0,
			GasPrice:        0,
			Amount:          "0",
			ContractAddress: "",
		},
	}
	b, err := json.Marshal(es)
	if err != nil {
		log.Error("marshal fail", "err", err)
	}
	return &wallet.GetChainSchemaResponse{
		Code:    wallet.ReturnCode_SUCCESS,
		Message: "get ethereum sign schema success",
		Schema:  string(b),
	}, nil
}

func (c *ChainAdaptor) CreateKeyPairsExportPublicKeyList(ctx context.Context, req *wallet.CreateKeyPairAndExportPublicKeyRequest) (*wallet.CreateKeyPairAndExportPublicKeyResponse, error) {
	signer := &ssm.ECDSASigner{}
	resp := &wallet.CreateKeyPairAndExportPublicKeyResponse{
		Code: wallet.ReturnCode_ERROR,
	}
	if req.KeyNum > 10000 {
		resp.Message = "Number must be less than 100000"
		return resp, nil
	}

	var keyList []leveldb.Key
	var retKeyList []*wallet.ExportPublicKey

	for i := 0; i < int(req.KeyNum); i++ {
		priKeyStr, pubKeyStr, compressPubkeyStr, err := signer.CreateKeyPair()
		if err != nil {
			resp.Message = "create key pairs fail"
			return resp, nil
		}
		keyItem := leveldb.Key{
			PrivateKey: priKeyStr,
			PubKey:     pubKeyStr,
		}
		pukItem := &wallet.ExportPublicKey{
			CompressPublicKey: compressPubkeyStr,
			PublicKey:         pubKeyStr,
		}
		retKeyList = append(retKeyList, pukItem)
		keyList = append(keyList, keyItem)
	}
	isOk := c.db.StoreKeys(keyList)
	if !isOk {
		log.Error("store keys fail", "isOk", isOk)
		return nil, errors.New("store keys fail")
	}
	resp.Code = wallet.ReturnCode_SUCCESS
	resp.Message = "create keys success"
	resp.PublicKeyList = retKeyList
	return resp, nil
}

func (c *ChainAdaptor) CreateKeyPairsWithAddresses(ctx context.Context, req *wallet.CreateKeyPairsWithAddressesRequest) (*wallet.CreateKeyPairsWithAddressesResponse, error) {
	signer := &ssm.ECDSASigner{}
	resp := &wallet.CreateKeyPairsWithAddressesResponse{
		Code: wallet.ReturnCode_ERROR,
	}
	if req.KeyNum > 10000 {
		resp.Message = "Number must be less than 100000"
		return resp, nil
	}
	var keyList []leveldb.Key
	var retKeyWithAddressList []*wallet.ExportPublicKeyWithAddress

	for i := 0; i < int(req.KeyNum); i++ {
		priKeyStr, pubKeyStr, compressPubkeyStr, err := signer.CreateKeyPair()
		if err != nil {
			resp.Message = "create key pairs fail"
			return resp, nil
		}
		keyItem := leveldb.Key{
			PrivateKey: priKeyStr,
			PubKey:     pubKeyStr,
		}
		publicKeyBytes, err := hex.DecodeString(pubKeyStr)
		pukAddressItem := &wallet.ExportPublicKeyWithAddress{
			PublicKey:         pubKeyStr,
			CompressPublicKey: compressPubkeyStr,
			Address:           hex.EncodeToString(crypto.Keccak256(publicKeyBytes[1:])[12:]),
		}
		retKeyWithAddressList = append(retKeyWithAddressList, pukAddressItem)
		keyList = append(keyList, keyItem)
	}
	isOk := c.db.StoreKeys(keyList)
	if !isOk {
		log.Error("store keys fail", "isOk", isOk)
		return nil, errors.New("store keys fail")
	}
	resp.Code = wallet.ReturnCode_SUCCESS
	resp.Message = "create keys with address success"
	resp.PublicKeyAddresses = retKeyWithAddressList
	return resp, nil
}

func (c *ChainAdaptor) BuildAndSignTransaction(ctx context.Context, req *wallet.BuildAndSignTransactionRequest) (*wallet.BuildAndSignTransactionResponse, error) {
	signer := &ssm.ECDSASigner{}
	resp := &wallet.BuildAndSignTransactionResponse{
		Code: wallet.ReturnCode_ERROR,
	}
	dFeeTx, _, err := c.buildDynamicFeeTx(req.TxBase64Body)
	if err != nil {
		return nil, err
	}
	rawTx, err := CreateEip1559UnSignTx(dFeeTx, dFeeTx.ChainID)
	if err != nil {
		log.Error("create un sign tx fail", "err", err)
		resp.Message = "get un sign tx fail"
		return resp, nil
	}
	privKey, isOk := c.db.GetPrivKey(req.PublicKey)
	if !isOk {
		log.Error("get private key by public key fail", "err", err)
		resp.Message = "get private key by public key fail"
		return resp, nil
	}

	signature, err := signer.SignMessage(privKey, rawTx)
	if err != nil {
		log.Error("sign transaction fail", "err", err)
		resp.Message = "sign transaction fail"
		return resp, nil
	}

	inputSignatureByteList, err := hex.DecodeString(signature)
	if err != nil {
		log.Error("decode signature failed", "err", err)
		resp.Message = "decode signature failed"
		return resp, nil
	}

	eip1559Signer, signedTx, signAndHandledTx, txHash, err := CreateEip1559SignedTx(dFeeTx, inputSignatureByteList, dFeeTx.ChainID)
	if err != nil {
		log.Error("create signed tx fail", "err", err)
		resp.Message = "create signed tx fail"
		return resp, nil
	}
	log.Info("sign transaction success",
		"eip1559Signer", eip1559Signer,
		"signedTx", signedTx,
		"signAndHandledTx", signAndHandledTx,
		"txHash", txHash,
	)
	resp.Code = wallet.ReturnCode_SUCCESS
	resp.Message = "sign whole transaction success"
	resp.SignedTx = signAndHandledTx
	resp.TxHash = txHash
	resp.TxMessageHash = rawTx
	return resp, nil
}

func (c *ChainAdaptor) BuildAndSignBatchTransaction(ctx context.Context, req *wallet.BuildAndSignBatchTransactionRequest) (*wallet.BuildAndSignBatchTransactionResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (c *ChainAdaptor) buildDynamicFeeTx(base64Tx string) (*types.DynamicFeeTx, *Eip1559DynamicFeeTx, error) {
	txReqJsonByte, err := base64.StdEncoding.DecodeString(base64Tx)
	if err != nil {
		log.Error("decode string fail", "err", err)
		return nil, nil, err
	}
	var dynamicFeeTx Eip1559DynamicFeeTx
	if err := json.Unmarshal(txReqJsonByte, &dynamicFeeTx); err != nil {
		log.Error("parse json fail", "err", err)
		return nil, nil, err
	}
	chainID := new(big.Int)
	maxPriorityFeePerGas := new(big.Int)
	maxFeePerGas := new(big.Int)
	amount := new(big.Int)

	if _, ok := chainID.SetString(dynamicFeeTx.ChainId, 10); !ok {
		return nil, nil, fmt.Errorf("invalid chain ID: %s", dynamicFeeTx.ChainId)
	}
	if _, ok := maxPriorityFeePerGas.SetString(dynamicFeeTx.MaxPriorityFeePerGas, 10); !ok {
		return nil, nil, fmt.Errorf("invalid max priority fee: %s", dynamicFeeTx.MaxPriorityFeePerGas)
	}
	if _, ok := maxFeePerGas.SetString(dynamicFeeTx.MaxFeePerGas, 10); !ok {
		return nil, nil, fmt.Errorf("invalid max fee: %s", dynamicFeeTx.MaxFeePerGas)
	}
	if _, ok := amount.SetString(dynamicFeeTx.Amount, 10); !ok {
		return nil, nil, fmt.Errorf("invalid amount: %s", dynamicFeeTx.Amount)
	}

	toAddress := common.HexToAddress(dynamicFeeTx.ToAddress)
	var finalToAddress common.Address
	var finalAmount *big.Int
	var buildData []byte
	log.Info("contract address check",
		"contractAddress", dynamicFeeTx.ContractAddress,
		"isEthTransfer", isEthTransfer(&dynamicFeeTx),
	)
	if isEthTransfer(&dynamicFeeTx) {
		finalToAddress = toAddress
		finalAmount = amount
	} else {
		contractAddress := common.HexToAddress(dynamicFeeTx.ContractAddress)
		buildData = BuildErc20Data(toAddress, amount)
		finalToAddress = contractAddress
		finalAmount = big.NewInt(0)
	}

	dFeeTx := &types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     dynamicFeeTx.Nonce,
		GasTipCap: maxPriorityFeePerGas,
		GasFeeCap: maxFeePerGas,
		Gas:       dynamicFeeTx.GasLimit,
		To:        &finalToAddress,
		Value:     finalAmount,
		Data:      buildData,
	}

	return dFeeTx, &dynamicFeeTx, nil
}

func isEthTransfer(tx *Eip1559DynamicFeeTx) bool {
	if tx.ContractAddress == "" ||
		tx.ContractAddress == "0x0000000000000000000000000000000000000000" ||
		tx.ContractAddress == "0x00" {
		return true
	}
	return false
}

func NewChainAdaptor(db *leveldb.Keys) (chain.IChainAdaptor, error) {
	return &ChainAdaptor{
		db:     db,
		signer: &ssm.ECDSASigner{},
	}, nil
}
