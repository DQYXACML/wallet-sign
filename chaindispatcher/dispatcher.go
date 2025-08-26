package chaindispatcher

import (
	"context"
	"encoding/base64"
	"github.com/DQYXACML/wallet-sign/chain"
	"github.com/DQYXACML/wallet-sign/chain/bitcoin"
	"github.com/DQYXACML/wallet-sign/chain/ethereum"
	"github.com/DQYXACML/wallet-sign/chain/solana"
	"github.com/DQYXACML/wallet-sign/config"
	"github.com/DQYXACML/wallet-sign/leveldb"
	"github.com/DQYXACML/wallet-sign/protobuf/wallet"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/keycard-go/hexutils"
	"google.golang.org/grpc"
	"runtime/debug"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	AccessToken string = "DappLinkTheWeb3202402290001"
	WalletKey   string = "DappLinkWalletServicesRiskKeyxxxxxxxKey"
	RisKKey     string = "DappLinkWalletServicesRiskKeyxxxxxxxKey"
)

type CommonRequest interface {
	GetConsumerToken() string
	GetChainName() string
}

type CommonReply = wallet.GetChainSignMethodResponse
type ChainType = string
type ChainDispatcher struct {
	registry map[string]chain.IChainAdaptor
}

func (c *ChainDispatcher) GetChainSignMethod(ctx context.Context, request *wallet.GetChainSignMethodRequest) (*wallet.GetChainSignMethodResponse, error) {
	resp := c.preHandler(request)
	if resp != nil {
		return &wallet.GetChainSignMethodResponse{
			Code: resp.Code,
			Msg:  resp.Msg,
		}, nil
	}
	return c.registry[request.GetChainName()].GetChainSignMethod(ctx, request)
}

func (c *ChainDispatcher) GetChainSchema(ctx context.Context, request *wallet.GetChainSchemaRequest) (*wallet.GetChainSchemaResponse, error) {
	resp := c.preHandler(request)
	if resp != nil {
		return &wallet.GetChainSchemaResponse{
			Code:    resp.Code,
			Message: resp.Msg,
		}, nil
	}
	return c.registry[request.GetChainName()].GetChainSchema(ctx, request)
}

func (c *ChainDispatcher) CreateKeyPairsExportPublicKeyList(ctx context.Context, request *wallet.CreateKeyPairAndExportPublicKeyRequest) (*wallet.CreateKeyPairAndExportPublicKeyResponse, error) {
	resp := c.preHandler(request)
	if resp != nil {
		return &wallet.CreateKeyPairAndExportPublicKeyResponse{
			Code:    resp.Code,
			Message: resp.Msg,
		}, nil
	}
	return c.registry[request.ChainName].CreateKeyPairsExportPublicKeyList(ctx, request)
}

func (c *ChainDispatcher) CreateKeyPairsWithAddresses(ctx context.Context, request *wallet.CreateKeyPairsWithAddressesRequest) (*wallet.CreateKeyPairsWithAddressesResponse, error) {
	resp := c.preHandler(request)
	if resp != nil {
		return &wallet.CreateKeyPairsWithAddressesResponse{
			Code:    resp.Code,
			Message: resp.Msg,
		}, nil
	}
	return c.registry[request.ChainName].CreateKeyPairsWithAddresses(ctx, request)
}

func (c *ChainDispatcher) BuildAndSignTransaction(ctx context.Context, request *wallet.BuildAndSignTransactionRequest) (*wallet.BuildAndSignTransactionResponse, error) {
	resp := c.preHandler(request)
	if resp != nil {
		return &wallet.BuildAndSignTransactionResponse{
			Code:    resp.Code,
			Message: resp.Msg,
		}, nil
	}
	txReqJsonByte, err := base64.StdEncoding.DecodeString(request.TxBase64Body)
	if err != nil {
		return &wallet.BuildAndSignTransactionResponse{
			Code:    wallet.ReturnCode_ERROR,
			Message: "decode base64 string fail",
		}, nil
	}
	RiskKeyHash := crypto.Keccak256(append(txReqJsonByte, []byte(RisKKey)...))
	RiskKeyHashStr := hexutils.BytesToHex(RiskKeyHash)
	if RiskKeyHashStr != request.RiskKeyHash {
		return &wallet.BuildAndSignTransactionResponse{
			Code:    wallet.ReturnCode_ERROR,
			Message: "riskKey hash check Fail",
		}, nil
	}
	WalletKeyHash := crypto.Keccak256(append(txReqJsonByte, []byte(WalletKey)...))
	WalletKeyHashStr := hexutils.BytesToHex(WalletKeyHash)
	if WalletKeyHashStr != request.WalletKeyHash {
		return &wallet.BuildAndSignTransactionResponse{
			Code:    wallet.ReturnCode_ERROR,
			Message: "wallet key hash Check Fail",
		}, nil
	}
	return c.registry[request.ChainName].BuildAndSignTransaction(ctx, request)
}

func (c *ChainDispatcher) BuildAndSignBatchTransaction(ctx context.Context, request *wallet.BuildAndSignBatchTransactionRequest) (*wallet.BuildAndSignBatchTransactionResponse, error) {
	resp := c.preHandler(request)
	if resp != nil {
		return &wallet.BuildAndSignBatchTransactionResponse{
			Code:    resp.Code,
			Message: resp.Msg,
		}, nil
	}
	return c.registry[request.ChainName].BuildAndSignBatchTransaction(ctx, request)
}

func (c *ChainDispatcher) SignTransactionMessage(ctx context.Context, request *wallet.SignTransactionMessageRequest) (*wallet.SignTransactionMessageResponse, error) {
	resp := c.preHandler(request)
	if resp != nil {
		return &wallet.SignTransactionMessageResponse{
			Code:    resp.Code,
			Message: resp.Msg,
		}, nil
	}
	return c.registry[request.ChainName].SignTransactionMessage(ctx, request)
}

func (c *ChainDispatcher) Interceptor(ctx context.Context, request interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	defer func() {
		if e := recover(); e != nil {
			log.Error("panic error", "msg", e)
			log.Debug(string(debug.Stack()))
			err = status.Errorf(codes.Internal, "Panic err: %v", e)
		}
	}()

	pos := strings.LastIndex(info.FullMethod, "/")
	consumerToken := request.(CommonRequest).GetConsumerToken()
	method := info.FullMethod[pos+1:]
	chainName := request.(CommonRequest).GetChainName()
	log.Info(method, "chain", chainName, "consumerToken", consumerToken, "req", request)
	resp, err = handler(ctx, request)
	log.Debug("Finish handling", "resp", resp, "err", err)
	return
}

func (c *ChainDispatcher) preHandler(req interface{}) (resp *CommonReply) {
	consumerToken := req.(CommonRequest).GetConsumerToken()
	if consumerToken != AccessToken {
		return &CommonReply{
			Code: wallet.ReturnCode_ERROR,
			Msg:  "Invalid consumer token",
		}
	}
	chainName := req.(CommonRequest).GetChainName()
	log.Debug("chain name", "chain", chainName, "req", req)
	if _, ok := c.registry[chainName]; !ok {
		return &CommonReply{
			Code: wallet.ReturnCode_ERROR,
			Msg:  "unsupported chain",
		}
	}
	return nil
}

func NewChainDispatcher(conf *config.Config) (*ChainDispatcher, error) {
	dispatcher := &ChainDispatcher{
		registry: make(map[string]chain.IChainAdaptor),
	}
	chainAdaptorFactoryMap := map[ChainType]func(db *leveldb.Keys) (chain.IChainAdaptor, error){
		bitcoin.ChainName:  bitcoin.NewChainAdaptor,
		ethereum.ChainName: ethereum.NewChainAdaptor,
		solana.ChainName:   solana.NewChainAdaptor,
	}
	supportedChains := []string{
		bitcoin.ChainName,
		ethereum.ChainName,
		solana.ChainName,
	}

	db, err := leveldb.NewKeyStore(conf.LevelDbPath)
	if err != nil {
		log.Error("new key store level db", "err", err)
		return nil, err
	}

	for _, c := range conf.Chains {
		if factory, ok := chainAdaptorFactoryMap[c]; ok {
			adaptor, err := factory(db)
			if err != nil {
				log.Crit("failed to setup chain", "chain", c, "error", err)
			}
			dispatcher.registry[c] = adaptor
		} else {
			log.Error("unsupported chain", "chain", c, "supportedChains", supportedChains)
		}
	}
	return dispatcher, nil
}
