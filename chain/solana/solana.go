package solana

import (
	"context"
	"encoding/json"
	"github.com/DQYXACML/wallet-sign/chain"
	"github.com/DQYXACML/wallet-sign/leveldb"
	"github.com/DQYXACML/wallet-sign/protobuf/wallet"
	"github.com/DQYXACML/wallet-sign/ssm"
	"github.com/ethereum/go-ethereum/log"
)

const ChainName = "Solana"

type ChainAdaptor struct {
	db     *leveldb.Keys
	signer ssm.Signer
}

func (c *ChainAdaptor) GetChainSignMethod(ctx context.Context, req *wallet.GetChainSignMethodRequest) (*wallet.GetChainSignMethodResponse, error) {
	return &wallet.GetChainSignMethodResponse{
		Code:       wallet.ReturnCode_SUCCESS,
		Msg:        "get sign method success",
		SignMethod: "eddsa",
	}, nil
}

func (c *ChainAdaptor) GetChainSchema(ctx context.Context, req *wallet.GetChainSchemaRequest) (*wallet.GetChainSchemaResponse, error) {
	ss := SolanaSchema{
		Nonce:           "",
		GasPrice:        "",
		GasTipCap:       "",
		GasFeeCap:       "",
		Gas:             0,
		ContractAddress: "",
		FromAddress:     "",
		ToAddress:       "",
		TokenId:         "",
		Value:           "",
	}
	b, err := json.Marshal(ss)
	if err != nil {
		log.Error("marshal fail", "err", err)
	}
	return &wallet.GetChainSchemaResponse{
		Code:    wallet.ReturnCode_SUCCESS,
		Message: "get solana sign schema success",
		Schema:  string(b),
	}, nil
}

func (c *ChainAdaptor) CreateKeyPairsExportPublicKeyList(ctx context.Context, req *wallet.CreateKeyPairAndExportPublicKeyRequest) (*wallet.CreateKeyPairAndExportPublicKeyResponse, error) {
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
		priKeyStr, pubKeyStr, compressPubKeyStr, err := c.signer.CreateKeyPair()
		if err != nil {
			if req.KeyNum > 10000 {
				resp.Message = "create key pair fail"
				return resp, nil
			}
		}
		keyItem := leveldb.Key{
			PrivateKey: priKeyStr,
			PubKey:     pubKeyStr,
		}
		pukItem := &wallet.ExportPublicKey{
			PublicKey:         pubKeyStr,
			CompressPublicKey: compressPubKeyStr,
		}
		retKeyList = append(retKeyList, pukItem)
		keyList = append(keyList, keyItem)
	}

	isOk := c.db.StoreKeys(keyList)
	if !isOk {
		resp.Message = "store keys fail"
		return resp, nil
	}

	resp.Code = wallet.ReturnCode_SUCCESS
	resp.Message = "create keys success"
	resp.PublicKeyList = retKeyList
	return resp, nil
}

func (c *ChainAdaptor) CreateKeyPairsWithAddresses(ctx context.Context, req *wallet.CreateKeyPairsWithAddressesRequest) (*wallet.CreateKeyPairsWithAddressesResponse, error) {
	resp := &wallet.CreateKeyPairsWithAddressesResponse{
		Code: wallet.ReturnCode_ERROR,
	}
	if req.KeyNum > 10000 {
		resp.Message = "Number must be less than 100000"
		return resp, nil
	}
	var keyList []leveldb.Key
	var retKeyList []*wallet.ExportPublicKeyWithAddress
	for i := 0; i < int(req.KeyNum); i++ {
		priKeyStr, pubKeyStr, compressPubKeyStr, err := c.signer.CreateKeyPair()
		if err != nil {
			if req.KeyNum > 10000 {
				resp.Message = "create key pair fail"
				return resp, nil
			}
		}
		keyItem := leveldb.Key{
			PrivateKey: priKeyStr,
			PubKey:     pubKeyStr,
		}
		address, err := PubKeyHexToAddress(pubKeyStr)
		if err != nil {
			resp.Message = "public key to address fail"
			return resp, nil
		}
		pukItem := &wallet.ExportPublicKeyWithAddress{
			PublicKey:         pubKeyStr,
			CompressPublicKey: compressPubKeyStr,
			Address:           address,
		}
		retKeyList = append(retKeyList, pukItem)
		keyList = append(keyList, keyItem)
	}
	isOk := c.db.StoreKeys(keyList)
	if !isOk {
		resp.Message = "store keys fail"
		return resp, nil
	}
	resp.Code = wallet.ReturnCode_SUCCESS
	resp.Message = "create keys success"
	resp.PublicKeyAddresses = retKeyList
	return resp, nil
}

func (c *ChainAdaptor) BuildAndSignTransaction(ctx context.Context, req *wallet.BuildAndSignTransactionRequest) (*wallet.BuildAndSignTransactionResponse, error) {
	resp := &wallet.BuildAndSignTransactionResponse{
		Code: wallet.ReturnCode_ERROR,
	}

	return resp, nil
}

func (c *ChainAdaptor) BuildAndSignBatchTransaction(ctx context.Context, req *wallet.BuildAndSignBatchTransactionRequest) (*wallet.BuildAndSignBatchTransactionResponse, error) {
	resp := &wallet.BuildAndSignBatchTransactionResponse{
		Code: wallet.ReturnCode_ERROR,
	}

	return resp, nil
}

func isSOLTransfer(coinAddress string) bool {
	return coinAddress == "" ||
		coinAddress == "So11111111111111111111111111111111111111112"
}

func NewChainAdaptor(db *leveldb.Keys) (chain.IChainAdaptor, error) {
	return &ChainAdaptor{
		db:     db,
		signer: &ssm.EdDSASigner{},
	}, nil
}
