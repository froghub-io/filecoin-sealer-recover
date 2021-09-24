package recovery

import (
	"bytes"
	"context"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/lotus/chain/actors/builtin/miner"
	"github.com/filecoin-project/lotus/chain/types"
	"golang.org/x/xerrors"
)

func GetOnChainSectorTicket(ctx context.Context, fullNodeApi v0api.FullNode, maddr address.Address, sid abi.SectorNumber) (abi.Randomness, *miner.SectorPreCommitOnChainInfo, error) {
	si, err := fullNodeApi.StateSectorGetInfo(ctx, maddr, sid, types.EmptyTSK)
	if err != nil {
		return nil, nil, err
	}
	if si == nil {
		return nil, nil, xerrors.Errorf("Miner(%v) Sector(%d) info Not Found", maddr, sid)
	}
	ts, err := fullNodeApi.ChainGetTipSetByHeight(ctx, si.Activation, types.EmptyTSK)
	if err != nil {
		return nil, nil, err
	}
	if ts == nil {
		return nil, nil, xerrors.Errorf("Height(%d) Tipset Not Found", si.Activation)
	}

	preCommitInfo, err := fullNodeApi.StateSectorPreCommitInfo(ctx, maddr, sid, ts.Key())
	if err != nil {
		return nil, nil, xerrors.Errorf("Getting sector PreCommit info err:", err)
	}

	ticketEpoch := preCommitInfo.Info.SealRandEpoch

	buf := new(bytes.Buffer)
	if err := maddr.MarshalCBOR(buf); err != nil {
		return nil, nil, xerrors.Errorf("Address MarshalCBOR err:", err)
	}

	ticket, err := fullNodeApi.ChainGetRandomnessFromTickets(ctx, ts.Key(), crypto.DomainSeparationTag_SealRandomness, ticketEpoch, buf.Bytes())
	if err != nil {
		return nil, nil, xerrors.Errorf("Getting Randomness err:", err)
	}

	return ticket, &preCommitInfo, err
}
