package export

import (
	"encoding/json"
	"fmt"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	cliutil "github.com/filecoin-project/lotus/cli/util"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/mitchellh/go-homedir"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
	"io/ioutil"
	"strconv"
	"time"
)

var log = logging.Logger("export")

var ExportsCmd = &cli.Command{
	Name:      "export",
	Usage:     "Export sector metadata",
	ArgsUsage: "[sectorNum1 sectorNum2 ...]",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "miner",
			Usage:    "Filecoin Miner. Such as: f01000",
			Required: true,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := cliutil.ReqContext(cctx)
		start := time.Now()

		if cctx.Args().Len() < 1 {
			return fmt.Errorf("at least one sector must be specified")
		}

		runSectors := make([]uint64, 0)
		for _, sn := range cctx.Args().Slice() {
			sectorNum, err := strconv.ParseUint(sn, 10, 64)
			if err != nil {
				return fmt.Errorf("could not parse sector number: %w", err)
			}
			runSectors = append(runSectors, sectorNum)
		}

		maddr, err := address.NewFromString(cctx.String("miner"))
		if err != nil {
			return xerrors.Errorf("Getting NewFromString err:", err)
		}

		fullNodeApi, closer, err := cliutil.GetFullNodeAPI(cctx)
		if err != nil {
			return xerrors.Errorf("Getting FullNodeAPI err:", err)
		}
		defer closer()

		//Sector size
		mi, err := fullNodeApi.StateMinerInfo(ctx, maddr, types.EmptyTSK)
		if err != nil {
			return xerrors.Errorf("Getting StateMinerInfo err:", err)
		}

		output := &RecoveryParams{
			Miner:      maddr,
			SectorSize: mi.SectorSize,
		}
		sectorInfos := make(SectorInfos, 0)
		failtSectors := make([]uint64, 0)
		for _, sector := range runSectors {
			ts, sectorPreCommitOnChainInfo, err := GetSectorCommitInfoOnChain(ctx, fullNodeApi, maddr, abi.SectorNumber(sector))
			if err != nil {
				log.Errorf("Getting sector (%d) precommit info error: %v ", sector, err)
				continue
			}
			si := &SectorInfo{
				SectorNumber: abi.SectorNumber(sector),
				SealProof:    sectorPreCommitOnChainInfo.Info.SealProof,
				SealedCID:    sectorPreCommitOnChainInfo.Info.SealedCID,
			}

			ticket, err := GetSectorTicketOnChain(ctx, fullNodeApi, maddr, ts, sectorPreCommitOnChainInfo)
			if err != nil {
				log.Errorf("Getting sector (%d) ticket error: %v ", sector, err)
				continue
			}
			si.Ticket = ticket

			sectorInfos = append(sectorInfos, si)
			output.SectorInfos = sectorInfos

			out, err := json.MarshalIndent(output, "", "\t")
			if err != nil {
				return err
			}

			of, err := homedir.Expand("sectors-recovery-" + maddr.String() + ".json")
			if err != nil {
				return err
			}

			if err := ioutil.WriteFile(of, out, 0644); err != nil {
				return err
			}
		}

		end := time.Now()
		fmt.Println("export", len(sectorInfos), "sectors, failt sectors:", failtSectors, ", elapsed:", end.Sub(start))

		return nil
	},
}

type RecoveryParams struct {
	Miner       address.Address
	SectorSize  abi.SectorSize
	SectorInfos SectorInfos
}

type SectorInfo struct {
	SectorNumber abi.SectorNumber
	Ticket       abi.Randomness
	SealProof    abi.RegisteredSealProof
	SealedCID    cid.Cid
}

type SectorInfos []*SectorInfo

func (t SectorInfos) Len() int { return len(t) }

func (t SectorInfos) Swap(i, j int) { t[i], t[j] = t[j], t[i] }

func (t SectorInfos) Less(i, j int) bool {
	return t[i].SectorNumber < t[j].SectorNumber
}
