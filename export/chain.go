/*
 * Copyright (C) 2022 FrogHub
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package export

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

func GetSectorTicketOnChain(ctx context.Context, fullNodeApi v0api.FullNode, maddr address.Address, ts *types.TipSet, preCommitInfo *miner.SectorPreCommitOnChainInfo) (abi.Randomness, error) {
	buf := new(bytes.Buffer)
	if err := maddr.MarshalCBOR(buf); err != nil {
		return nil, xerrors.Errorf("Address MarshalCBOR err:", err)
	}

	ticket, err := fullNodeApi.StateGetRandomnessFromTickets(ctx, crypto.DomainSeparationTag_SealRandomness, preCommitInfo.Info.SealRandEpoch, buf.Bytes(), ts.Key())
	if err != nil {
		return nil, xerrors.Errorf("Getting Randomness err:", err)
	}

	return ticket, err
}

func GetSectorCommitInfoOnChain(ctx context.Context, fullNodeApi v0api.FullNode, maddr address.Address, sid abi.SectorNumber) (*types.TipSet, *miner.SectorPreCommitOnChainInfo, error) {
	si, err := fullNodeApi.StateSectorGetInfo(ctx, maddr, sid, types.EmptyTSK)
	if err != nil {
		return nil, nil, err
	}
	if si == nil {
		//Provecommit not submitted
		preCommitInfo, err := fullNodeApi.StateSectorPreCommitInfo(ctx, maddr, sid, types.EmptyTSK)
		if err != nil {
			return nil, nil, xerrors.Errorf("Getting sector PreCommit info err:", err)
		}

		ts, err := fullNodeApi.ChainGetTipSetByHeight(ctx, preCommitInfo.PreCommitEpoch, types.EmptyTSK)
		if err != nil {
			return nil, nil, err
		}
		if ts == nil {
			return nil, nil, xerrors.Errorf("Height(%d) Tipset Not Found")
		}
		return ts, &preCommitInfo, err
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

	return ts, &preCommitInfo, err
}
