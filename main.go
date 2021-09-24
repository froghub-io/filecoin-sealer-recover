package main

import (
	"context"
	"github.com/docker/go-units"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
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
			&cli.StringFlag{
				Name:     "sectorNum",
				Usage:    "Sector number to be recovered. Such as: 0",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "sector-size",
				Value: "32GiB",
				Usage: "Size of the sectors in bytes, i.e. 32GiB",
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

			miner := cctx.String("miner")
			sectorNum := cctx.Uint64("sectorNum")
			sealingResult := cctx.String("sealing-result")
			sealingTemp := cctx.String("sealing-temp")
			sectorSizeInt, err := units.RAMInBytes(cctx.String("sector-size"))
			if err != nil {
				return err
			}
			sectorSize := abi.SectorSize(sectorSizeInt)

			maddr, err := address.NewFromString(miner)
			if err != nil {
				return xerrors.Errorf("Getting NewFromString err:", err)
			}
			actorID, err := address.IDFromAddress(maddr)
			if err != nil {
				return xerrors.Errorf("Getting IDFromAddress err:", err)
			}
			fapi, closer, err := cliutil.GetFullNodeAPI(cctx)
			if err != nil {
				return xerrors.Errorf("Getting FullNodeAPI err:", err)
			}
			defer closer()

			ticketValue, sectorPreCommitOnChainInfo, err := recovery.GetOnChainSectorTicket(ctx, fapi, maddr, abi.SectorNumber(sectorNum))
			if err != nil {
				return err
			}

			if err = recovery.RecoverSealedFile(actorID, sectorNum, sectorSize, sealingResult, sealingTemp, abi.SealRandomness(ticketValue), sectorPreCommitOnChainInfo); err != nil {
				return err
			}
			log.Info("recovery sealed success!")
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
