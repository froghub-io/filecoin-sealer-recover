package recovery

import (
	"context"
	"fmt"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/extern/sector-storage/ffiwrapper"
	"github.com/filecoin-project/lotus/extern/sector-storage/ffiwrapper/basicfs"
	sealing "github.com/filecoin-project/lotus/extern/storage-sealing"
	"github.com/filecoin-project/specs-storage/storage"
	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"golang.org/x/xerrors"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"
)

func RecoverSealedFile(ctx context.Context, fullNodeApi v0api.FullNode, maddr address.Address, actorID uint64, sectors []int, parallel uint, sealingResult string, sealingTemp string) error {
	// Sector size
	mi, err := fullNodeApi.StateMinerInfo(ctx, maddr, types.EmptyTSK)
	if err != nil {
		return xerrors.Errorf("Getting StateMinerInfo err:", err)
	}

	wg := &sync.WaitGroup{}
	limiter := make(chan bool, parallel)
	var p1LastTaskTime time.Time
	for _, sector := range sectors {
		wg.Add(1)
		limiter <- true
		go func(sector int64) {
			defer func() {
				wg.Done()
				<-limiter
			}()

			//Control PC1 running interval
			for {
				if time.Now().Add(-time.Minute * 10).After(p1LastTaskTime) {
					break
				}
				<-time.After(p1LastTaskTime.Sub(time.Now()))
			}
			p1LastTaskTime = time.Now()

			ts, sectorPreCommitOnChainInfo, err := GetSectorCommitInfoOnChain(ctx, fullNodeApi, maddr, abi.SectorNumber(sector))
			if err != nil {
				log.Errorf("Getting sector (%d) precommit info error: %v ", sector, err)
			}

			ticket, err := GetSectorTicketOnChain(ctx, fullNodeApi, maddr, ts, sectorPreCommitOnChainInfo)
			if err != nil {
				log.Errorf("Getting sector (%d) ticket error: %v ", sector, err)
			}

			sdir, err := homedir.Expand(sealingTemp)
			if err != nil {
				log.Errorf("Sector (%d) ,expands the path error: %v", sector, err)
			}
			mkdirAll(sdir)
			tempDir, err := ioutil.TempDir(sdir, fmt.Sprintf("recover-%d", sector))
			if err != nil {
				log.Errorf("Sector (%d) ,creates a new temporary directory error: %v", sector, err)
			}
			if err := os.MkdirAll(tempDir, 0775); err != nil {
				log.Errorf("Sector (%d) ,creates a directory named path error: %v", sector, err)
			}

			sb, err := ffiwrapper.New(&basicfs.Provider{
				Root: tempDir,
			})
			if err != nil {
				log.Errorf("Sector (%d) ,new ffi Sealer error: %v", sector, err)
			}

			sid := storage.SectorRef{
				ID: abi.SectorID{
					Miner:  abi.ActorID(actorID),
					Number: abi.SectorNumber(sector),
				},
				ProofType: sectorPreCommitOnChainInfo.Info.SealProof,
			}

			log.Infof("Start recover sector(%d,%d), registeredSealProof: %d, ticket: %x", actorID, sector, sectorPreCommitOnChainInfo.Info.SealProof, ticket)

			log.Infof("Start running AP, sector (%d)", sector)
			pi, err := sb.AddPiece(context.TODO(), sid, nil, abi.PaddedPieceSize(mi.SectorSize).Unpadded(), sealing.NewNullReader(abi.UnpaddedPieceSize(mi.SectorSize)))
			if err != nil {
				log.Errorf("Sector (%d) ,running AP  error: %v", sector, err)
			}
			var pieces []abi.PieceInfo
			pieces = append(pieces, pi)
			log.Infof("Complete AP, sector (%d)", sector)

			log.Infof("Start running PreCommit1, sector (%d)", sector)
			pc1o, err := sb.SealPreCommit1(context.TODO(), sid, abi.SealRandomness(ticket), []abi.PieceInfo{pi})
			if err != nil {
				log.Errorf("Sector (%d) , running PreCommit1  error: %v", sector, err)
			}
			log.Infof("Complete PreCommit1, sector (%d)", sector)

			err = sealPreCommit2AndCheck(ctx, sb, sid, pc1o, sectorPreCommitOnChainInfo.Info.SealedCID.String())
			if err != nil {
				log.Errorf("Sector (%d) , running PreCommit2  error: %v", sector, err)
			}

			go func() {
				err = MoveStorage(ctx, sid, tempDir, sealingResult)
				if err != nil {
					log.Errorf("Sector (%d) , running MoveStorage  error: %v", sector, err)
				}
			}()

			log.Infof("Complete sector (%d)", sector)
		}(int64(sector))
	}
	wg.Wait()

	return nil
}

var pc2Lock sync.Mutex

func sealPreCommit2AndCheck(ctx context.Context, sb *ffiwrapper.Sealer, sid storage.SectorRef, phase1Out storage.PreCommit1Out, sealedCID string) error {
	pc2Lock.Lock()
	log.Infof("Start running PreCommit2, sector (%d)", sid.ID)

	cids, err := sb.SealPreCommit2(ctx, sid, phase1Out)
	if err != nil {
		pc2Lock.Unlock()
		return err
	}
	pc2Lock.Unlock()
	log.Infof("Complete PreCommit2, sector (%d)", sid.ID)

	//check CID with chain
	if sealedCID != cids.Sealed.String() {
		return xerrors.Errorf("sealed cid mismatching!!! (sealedCID: %v, newSealedCID: %v)", sealedCID, cids.Sealed.String())
	}
	return nil
}

func MoveStorage(ctx context.Context, sector storage.SectorRef, tempDir string, sealingResult string) error {
	//del unseal
	if err := os.RemoveAll(tempDir + "/unsealed"); err != nil {
		return xerrors.Errorf("SectorID: %d, del unseal error：%s", sector.ID, err)
	}
	sectorNum := "s-t0" + sector.ID.Miner.String() + "-" + sector.ID.Number.String()

	//del layer
	files, _ := ioutil.ReadDir(tempDir + "/cache/" + sectorNum)
	for _, f := range files {
		if strings.Contains(f.Name(), "layer") || strings.Contains(f.Name(), "tree-c") || strings.Contains(f.Name(), "tree-d") {
			if err := os.RemoveAll(tempDir + "/cache/" + sectorNum + "/" + f.Name()); err != nil {
				return xerrors.Errorf("SectorID: %d, del layer error：%s", sector.ID, err)
			}
		}
	}

	//move to storage
	mkdirAll(sealingResult)
	mkdirAll(sealingResult + "/cache")
	mkdirAll(sealingResult + "/sealed")
	if err := move(tempDir+"/cache/"+sectorNum, sealingResult+"/cache/"+sectorNum); err != nil {
		return xerrors.Errorf("SectorID: %d, move cache error：%s", sector.ID, err)
	}
	if err := move(tempDir+"/sealed/"+sectorNum, sealingResult+"/sealed/"+sectorNum); err != nil {
		return xerrors.Errorf("SectorID: %d, move sealed error：%s", sector.ID, err)
	}

	return nil
}
