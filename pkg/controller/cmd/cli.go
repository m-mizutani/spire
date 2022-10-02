package cmd

import (
	"fmt"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	"github.com/m-mizutani/goerr"
	"github.com/m-mizutani/spire/pkg/handler"
	"github.com/m-mizutani/spire/pkg/model"
	"github.com/m-mizutani/spire/pkg/types"
	"github.com/m-mizutani/spire/pkg/utils"
	"github.com/m-mizutani/zlog"
	"github.com/urfave/cli/v2"
)

type config struct {
	ifName string
}

func Run(argv []string) error {
	var cfg config
	var (
		logLevel string
	)

	app := cli.App{
		Name:  "spire",
		Usage: "Passive network quality monitoring",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "interface",
				Aliases:     []string{"i"},
				Destination: &cfg.ifName,
			},
			&cli.StringFlag{
				Name:        "log-level",
				Aliases:     []string{"l"},
				Usage:       "logging level [trace|debug|info|warn|error]",
				Value:       "info",
				Destination: &logLevel,
			},
		},
		Before: func(ctx *cli.Context) error {
			utils.RenewLogger(zlog.WithLogLevel(logLevel))
			return nil
		},
		Action: func(c *cli.Context) error {
			logCh := make(chan *model.FlowLog)
			hdlr := handler.New(logCh)

			handle, err := pcap.OpenLive(cfg.ifName, spanLen, true, pcap.BlockForever)
			if err != nil {
				return goerr.Wrap(err)
			}
			defer handle.Close()

			src := gopacket.NewPacketSource(handle, handle.LinkType())

			ticker := time.NewTicker(time.Second)

			go func() {
				for log := range logCh {
					fmt.Printf(
						"%s %v (%s) %5.2fms latency (sent %d bytes / recv %d bytes)\n",
						time.Now().Format("2006-01-02T15:04:05.000"),
						log.Server.Names,
						log.Server.Addr,
						log.Latency*1000,
						log.Client.DataSize,
						log.Server.DataSize,
					)
				}
			}()

			for {
				select {
				case <-ticker.C:
					hdlr.Elapse(1)

				case pkt := <-src.Packets():
					ctx := types.NewContext()
					hdlr.ServePacket(ctx, pkt)
				}
			}
		},
	}

	if err := app.Run(argv); err != nil {
		return err
	}

	return nil
}

const (
	spanLen = 262144
)
