package export

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	cliutil "github.com/filecoin-project/lotus/cli/util"
	"github.com/ipfs/go-cid"
	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
	"io/ioutil"
	"time"
)

var ExportCmd = &cli.Command{
	Name:  "export",
	Usage: "Export sector metadata",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "miner",
			Usage:    "Filecoin Miner. Such as: f01000",
			Required: true,
		},
		&cli.StringFlag{
			Name:  "sectors-metadata",
			Usage: "specify the metadata file for the sectors",
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := cliutil.ReqContext(cctx)
		start := time.Now()

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

		pssb := cctx.String("sectors-metadata")
		if pssb == "" {
			return xerrors.Errorf("Undefined sectors metadata")
		}

		log.Infof("Importing sectors recovery metadata for %s", pssb)

		sectors, err := migrateSectorsMeta(ctx, pssb)
		if err != nil {
			return xerrors.Errorf("migrating sectors metadata: %w", err)
		}

		output := &RecoveryParams{
			Miner:      maddr,
			SectorSize: mi.SectorSize,
		}
		sectorInfos := make(SectorInfos, 0)
		failtSectors := make([]uint64, 0)
		for _, sector := range sectors {
			ts, sectorPreCommitOnChainInfo, err := GetSectorCommitInfoOnChain(ctx, fullNodeApi, maddr, abi.SectorNumber(sector))
			if err != nil {
				log.Errorf("Getting sector (%d) precommit info error: %v ", sector, err)
				continue
			}
			si := &SectorInfo{
				SectorNumber: sector,
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

type SectorNumbers []abi.SectorNumber

func migrateSectorsMeta(ctx context.Context, metadata string) ([]abi.SectorNumber, error) {
	metadata, err := homedir.Expand(metadata)
	if err != nil {
		return []abi.SectorNumber{}, xerrors.Errorf("expanding sectors recovery dir: %w", err)
	}

	b, err := ioutil.ReadFile(metadata)
	if err != nil {
		return []abi.SectorNumber{}, xerrors.Errorf("reading sectors recovery metadata: %w", err)
	}

	var sectors []abi.SectorNumber
	if err := json.Unmarshal(b, &sectors); err != nil {
		return []abi.SectorNumber{}, xerrors.Errorf("unmarshaling sectors recovery metadata: %w", err)
	}

	return sectors, nil
}
