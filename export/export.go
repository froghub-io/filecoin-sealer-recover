package export

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/lotus/chain/types"
	cliutil "github.com/filecoin-project/lotus/cli/util"
	"github.com/ipfs/go-cid"
	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
	"io/ioutil"
	"sort"
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

		sectorInfos := make(SectorInfos, 0)
		failtSectors := make([]uint64, 0)
		for _, sector := range sectors {
			si, err := fullNodeApi.StateSectorGetInfo(ctx, maddr, sector, types.EmptyTSK)
			if err != nil {
				log.Errorf("Sector (%d), StateSectorGetInfo error: %v", sector, err)
				failtSectors = append(failtSectors, uint64(sector))
				continue
			}

			if si == nil {
				//ProveCommit not submitted
				preCommitInfo, err := fullNodeApi.StateSectorPreCommitInfo(ctx, maddr, sector, types.EmptyTSK)
				if err != nil {
					log.Errorf("Sector (%d), StateSectorPreCommitInfo error: %v", sector, err)
					failtSectors = append(failtSectors, uint64(sector))
					continue
				}
				sectorInfos = append(sectorInfos, &SectorInfo{
					SectorNumber: sector,
					SealProof:    preCommitInfo.Info.SealProof,
					Activation:   preCommitInfo.PreCommitEpoch,
					SealedCID:    preCommitInfo.Info.SealedCID,
				})
				continue
			}

			sectorInfos = append(sectorInfos, &SectorInfo{
				SectorNumber: sector,
				SealProof:    si.SealProof,
				Activation:   si.Activation,
				SealedCID:    si.SealedCID,
			})
		}

		//sort by sectorInfo.Activation
		//walk back from the execTs instead of HEAD, to save time.
		sort.Sort(sectorInfos)

		buf := new(bytes.Buffer)
		if err := maddr.MarshalCBOR(buf); err != nil {
			return xerrors.Errorf("Address MarshalCBOR err:", err)
		}

		output := &RecoveryParams{
			Miner:       maddr,
			SectorSize:  mi.SectorSize,
			SectorInfos: sectorInfos,
		}
		outputSectorInfos := make(SectorInfos, 0)

		tsk := types.EmptyTSK
		for _, sectorInfo := range sectorInfos {
			ts, err := fullNodeApi.ChainGetTipSetByHeight(ctx, sectorInfo.Activation, tsk)
			tsk = ts.Key()

			ticket, err := fullNodeApi.StateGetRandomnessFromTickets(ctx, crypto.DomainSeparationTag_SealRandomness, sectorInfo.Activation, buf.Bytes(), tsk)
			if err != nil {
				log.Errorf("Sector (%d), Getting Randomness  error: %v", sectorInfo.SectorNumber, err)
				failtSectors = append(failtSectors, uint64(sectorInfo.SectorNumber))
				continue
			}
			sectorInfo.Ticket = ticket

			outputSectorInfos = append(outputSectorInfos, sectorInfo)
			output.SectorInfos = outputSectorInfos

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
	Activation   abi.ChainEpoch
	Ticket       abi.Randomness
	SealProof    abi.RegisteredSealProof
	SealedCID    cid.Cid
}

type SectorInfos []*SectorInfo

func (t SectorInfos) Len() int { return len(t) }

func (t SectorInfos) Swap(i, j int) { t[i], t[j] = t[j], t[i] }

func (t SectorInfos) Less(i, j int) bool {
	if t[i].Activation != t[j].Activation {
		return t[i].Activation > t[j].Activation
	}

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
