package recovery

import (
	"context"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/actors/builtin/miner"
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
)

func RecoverSealedFile(actorID uint64, sectorNum uint64, sectorSize abi.SectorSize, sealingResult string, sealingTemp string, ticket abi.SealRandomness, sectorPreCommitOnChainInfo *miner.SectorPreCommitOnChainInfo) error {
	sdir, err := homedir.Expand(sealingTemp)
	if err != nil {
		return err
	}
	mkdirAll(sdir)

	tempDir, err := ioutil.TempDir(sdir, "recover")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(tempDir, 0775); err != nil {
		return err
	}

	sbfs := &basicfs.Provider{
		Root: tempDir,
	}

	sb, err := ffiwrapper.New(sbfs)
	if err != nil {
		return err
	}

	sid := storage.SectorRef{
		ID: abi.SectorID{
			Miner:  abi.ActorID(actorID),
			Number: abi.SectorNumber(sectorNum),
		},
		ProofType: sectorPreCommitOnChainInfo.Info.SealProof,
	}

	log.Infof("Start recover sector(%d,%d), registeredSealProof: %d, ticket: %x", actorID, sectorNum, sectorPreCommitOnChainInfo.Info.SealProof, ticket)

	var pieces []abi.PieceInfo

	log.Info("Start running AP")
	pi, err := sb.AddPiece(context.TODO(), sid, nil, abi.PaddedPieceSize(sectorSize).Unpadded(), sealing.NewNullReader(abi.UnpaddedPieceSize(sectorSize)))
	if err != nil {
		return xerrors.Errorf("running AP err:", err)
	}
	log.Info("Complete AP")

	pieces = append(pieces, pi)
	log.Info("Start running PreCommit1")
	pc1o, err := sb.SealPreCommit1(context.TODO(), sid, ticket, []abi.PieceInfo{pi})
	if err != nil {
		return xerrors.Errorf("running PreCommit1 err:", err)
	}
	log.Info("Complete PreCommit1")

	log.Info("Start running PreCommit2")
	err = sealPreCommit2(sb, tempDir, sealingResult, sid, pc1o, sectorPreCommitOnChainInfo.Info.SealedCID.String())
	if err != nil {
		return xerrors.Errorf("running PreCommit2 err:", err)
	}
	log.Info("Complete PreCommit2")
	return nil
}

func sealPreCommit2(sb *ffiwrapper.Sealer, tempDir string, sealingResult string, sector storage.SectorRef, phase1Out storage.PreCommit1Out, sealedCID string) error {
	cids, err := sb.SealPreCommit2(context.TODO(), sector, phase1Out)
	if err != nil {
		return err
	}

	//del unseal
	if err := os.RemoveAll(tempDir + "/unsealed"); err != nil {
		log.Error("del unseal", err)
	}
	sectorNum := "s-t0" + sector.ID.Miner.String() + "-" + sector.ID.Number.String()

	//del layer
	files, _ := ioutil.ReadDir(tempDir + "/cache/" + sectorNum)
	for _, f := range files {
		if strings.Contains(f.Name(), "layer") || strings.Contains(f.Name(), "tree-c") || strings.Contains(f.Name(), "tree-d") {
			if err := os.RemoveAll(tempDir + "/cache/" + sectorNum + "/" + f.Name()); err != nil {
				log.Error("del layer", err)
			}
		}
	}

	//mv
	mkdirAll(sealingResult)
	mkdirAll(sealingResult + "/cache")
	mkdirAll(sealingResult + "/sealed")
	if err := move(tempDir+"/cache/"+sectorNum, sealingResult+"/cache/"+sectorNum); err != nil {
		log.Error("cache", err)
	}
	if err := move(tempDir+"/sealed/"+sectorNum, sealingResult+"/sealed/"+sectorNum); err != nil {
		log.Error("sealed", err)
	}

	//check
	if sealedCID != cids.Sealed.String() {
		return xerrors.Errorf("sealed cid mismatching!!! (sealedCID: %v, newSealedCID: %v)", sealedCID, cids.Sealed.String())
	}
	return nil
}
