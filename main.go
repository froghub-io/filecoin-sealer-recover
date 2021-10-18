package main

import (
	"context"
	"github.com/filecoin-project/go-address"
	cliutil "github.com/filecoin-project/lotus/cli/util"
	"github.com/froghub-io/filecoin-sealer-recover/recovery"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
	"os"
)

func main() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
	app := &cli.App{
		Name:    "sealer-recovery",
		Usage:   "Filecoin sealer recovery",
		Version: BuildVersion,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "miner",
				Usage:    "Filecoin miner. Such as: f01000",
				Required: true,
			},
			&cli.IntSliceFlag{
				Name:     "sectors",
				Usage:    "Sector number to be recovered. Such as: 0",
				Required: true,
			},
			&cli.UintFlag{
				Name:  "parallel",
				Usage: "Number of parallel P1",
				Value: 1,
			},
			&cli.StringFlag{
				Name:  "sealing-result",
				Value: "~/sector",
				Usage: "Recover sector result path",
			},
			&cli.StringFlag{
				Name:  "sealing-temp",
				Value: "~/temp",
				Usage: "Temporarily generated during sector recovery",
			},
		},
		Action: func(cctx *cli.Context) error {
			log.Info("Start sealer recovery!")

			ctx := cliutil.DaemonContext(cctx)
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			maddr, err := address.NewFromString(cctx.String("miner"))
			if err != nil {
				return xerrors.Errorf("Getting NewFromString err:", err)
			}
			actorID, err := address.IDFromAddress(maddr)
			if err != nil {
				return xerrors.Errorf("Getting IDFromAddress err:", err)
			}

			fullapi, closer, err := cliutil.GetFullNodeAPI(cctx)
			if err != nil {
				return xerrors.Errorf("Getting FullNodeAPI err:", err)
			}
			defer closer()

			if err = recovery.RecoverSealedFile(ctx, fullapi, maddr, actorID, cctx.IntSlice("sectors"), cctx.Uint("parallel"), cctx.String("sealing-result"), cctx.String("sealing-temp")); err != nil {
				return err
			}
			log.Info("Complete recovery sealed!")
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Warnf("%+v", err)
		return
	}

}

// BuildVersion is the local build version
const BuildVersion = "1.0.0"
