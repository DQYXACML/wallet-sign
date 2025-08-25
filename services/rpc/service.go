package rpc

import (
	"context"
	"fmt"
	"github.com/DQYXACML/wallet-sign/chaindispatcher"
	"github.com/DQYXACML/wallet-sign/config"
	"github.com/DQYXACML/wallet-sign/protobuf/wallet"
	"github.com/ethereum/go-ethereum/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"net"
	"sync/atomic"
)

const MaxReceivedMessageSize = 1024 * 1024 * 30000

type RpcService struct {
	conf *config.Config
	wallet.UnimplementedWalletServiceServer
	stopped atomic.Bool
}

func (s *RpcService) Stop(ctx context.Context) error {
	s.stopped.Store(true)
	return nil
}

func (s *RpcService) Stopped() bool {
	return s.stopped.Load()
}

func NewRpcService() (*RpcService, error) {
	rpcService := &RpcService{}
	return rpcService, nil
}

func (s *RpcService) Start(ctx context.Context) error {
	go func(s *RpcService) {
		addr := fmt.Sprintf("%s:%d", s.conf.RpcServer.Host, s.conf.RpcServer.Port)
		opt := grpc.MaxRecvMsgSize(MaxReceivedMessageSize)

		dispatcher, err := chaindispatcher.NewChainDispatcher(s.conf)
		if err != nil {
			log.Error("new chain dispatcher fail", "err", err)
			return
		}
		gs := grpc.NewServer(opt, grpc.UnaryInterceptor(dispatcher.Interceptor))

		wallet.RegisterWalletServiceServer(gs, dispatcher)
		listener, err := net.Listen("tcp", addr)
		if err != nil {
			log.Error("Could not start tcp listener. ")
			return
		}
		reflection.Register(gs) // grpcui -plaintext 127.0.0.1:port
		log.Info("Grpc info", "port", s.conf.RpcServer.Port, "address", listener.Addr())

		if err := gs.Serve(listener); err != nil {
			log.Error("Could not GRPC services")
		}
	}(s)
	return nil
}
