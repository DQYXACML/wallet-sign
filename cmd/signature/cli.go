package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/DQYXACML/wallet-sign/common/cliapp"
	"github.com/DQYXACML/wallet-sign/config"
	"github.com/DQYXACML/wallet-sign/flags"
	"github.com/DQYXACML/wallet-sign/services/rpc"

	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/log"
)

func runRpc(ctx *cli.Context, shutdown context.CancelCauseFunc) (cliapp.Lifecycle, error) {
	fmt.Println("running grpc services...")
	var f = flag.String("c", "config.yml", "config path")
	flag.Parse()
	cfg, err := config.NewConfig(*f)
	if err != nil {
		log.Error("new config fail", "err", err)
		return nil, err
	}
	return rpc.NewRpcService(cfg)
}

func NewCli() *cli.App {
	flags := flags.Flags
	return &cli.App{
		Version:              "v0.0.1-beta",
		Description:          "wallet sign rpc service",
		EnableBashCompletion: true,
		Commands: []*cli.Command{
			{
				Name:        "rpc",
				Flags:       flags,
				Description: "Run rpc services",
				Action:      cliapp.LifecycleCmd(runRpc),
			},
			{
				Name:        "version",
				Description: "Show project version",
				Action: func(ctx *cli.Context) error {
					cli.ShowVersion(ctx)
					return nil
				},
			},
		},
	}
}
