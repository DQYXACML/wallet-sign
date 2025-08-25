package ethereum

import (
	"encoding/hex"
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"math/big"
)

func BuildErc20Data(toAddress common.Address, amount *big.Int) []byte {
	var data []byte

	transferFnSignature := []byte("transfer(address,uint256)")
	hash := crypto.Keccak256Hash(transferFnSignature)
	methodId := hash[:4]
	dataAddress := common.LeftPadBytes(toAddress.Bytes(), 32)
	dataAmount := common.LeftPadBytes(amount.Bytes(), 32)

	data = append(data, methodId...)
	data = append(data, dataAddress...)
	data = append(data, dataAmount...)

	return data
}

func CreateEip1559UnSignTx(txData *types.DynamicFeeTx, chainId *big.Int) (string, error) {
	tx := types.NewTx(txData)
	// 签名者
	signer := types.LatestSignerForChainID(chainId)
	txHash := signer.Hash(tx)
	return txHash.String(), nil
}

func CreateEip1559SignedTx(txData *types.DynamicFeeTx, signature []byte, chainId *big.Int) (types.Signer, *types.Transaction, string, string, error) {
	tx := types.NewTx(txData)
	signer := types.LatestSignerForChainID(chainId)
	signedTx, err := tx.WithSignature(signer, signature)
	if err != nil {
		return nil, nil, "", "", errors.New("tx with signature fail")
	}
	signedTxData, err := rlp.EncodeToBytes(signedTx)
	if err != nil {
		return nil, nil, "", "", errors.New("encode tx to byte fail")
	}
	return signer, signedTx, "0x" + hex.EncodeToString(signedTxData)[4:], signedTx.Hash().String(), nil
}
