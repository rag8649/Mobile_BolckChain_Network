package keeper

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/reward/types"
)

type msgServer struct {
	Keeper
}

func NewMsgServerImpl(k Keeper) types.MsgServer {
	return &msgServer{Keeper: k}
}
func (m *msgServer) RewardSolarPower(goCtx context.Context, msg *types.MsgRewardSolarPower) (*types.MsgRewardSolarPowerResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// 여기서 내부 로직 호출
	err := m.Keeper.RewardSolarPower(ctx, msg.Address, msg.Amount)
	if err != nil {
		return nil, err
	}

	return &types.MsgRewardSolarPowerResponse{}, nil
}

func (m msgServer) BurnStableCoin(goCtx context.Context, msg *types.MsgBurnStableCoin) (*types.MsgBurnStableCoinResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Keeper 호출
	resp, err := m.Keeper.BurnStableCoin(ctx, msg.TargetAddr, msg.Amount)
	if err != nil {
		return nil, err
	}

	// === 이벤트 발행 (CLI JSON 로그에 노출) ===
	ctx.EventManager().EmitEvent(
		sdk.NewEvent("burn_stable_coin",
			sdk.NewAttribute("target", msg.TargetAddr),
			sdk.NewAttribute("amount", msg.Amount),
			sdk.NewAttribute("returned_records", fmt.Sprintf("%d", len(resp.RecRecords))),
			sdk.NewAttribute("returned_metas", fmt.Sprintf("%d", len(resp.RecMetas))),
		),
	)

	// RECRecord 상세 이벤트
	for _, rec := range resp.RecRecords {
		bz, _ := json.Marshal(rec)
		ctx.EventManager().EmitEvent(
			sdk.NewEvent("rec_record_returned",
				sdk.NewAttribute("rec_id", rec.RecId),
				sdk.NewAttribute("data", string(bz)),
			),
		)
	}

	// RECMeta 상세 이벤트
	for _, meta := range resp.RecMetas {
		bz, _ := json.Marshal(meta)
		ctx.EventManager().EmitEvent(
			sdk.NewEvent("rec_meta_returned",
				sdk.NewAttribute("certified_id", meta.CertifiedId),
				sdk.NewAttribute("data", string(bz)),
			),
		)
	}

	// 최종 응답 반환
	return resp, nil
}

// DistributeRewardPercent : 모듈 계좌 잔액의 일정 비율을 address에게 지급
func (m msgServer) DistributeRewardPercent(goCtx context.Context, msg *types.MsgDistributeRewardPercent) (*types.MsgDistributeRewardPercentResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := m.Keeper.DistributeRewardPercent(ctx, msg); err != nil {
		return nil, err
	}

	return &types.MsgDistributeRewardPercentResponse{}, nil
}

func (m msgServer) DepositCollateral(goCtx context.Context, msg *types.MsgDepositCollateral) (*types.MsgDepositCollateralResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := m.Keeper.DepositCollateral(ctx, msg); err != nil {
		return nil, err
	}

	return &types.MsgDepositCollateralResponse{}, nil
}

func (m msgServer) RemoveCollateral(goCtx context.Context, msg *types.MsgRemoveCollateral) (*types.MsgRemoveCollateralResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := m.Keeper.RemoveCollateral(ctx, msg.Creator, msg.Amount); err != nil {
		return nil, err
	}

	return &types.MsgRemoveCollateralResponse{}, nil
}

func (m msgServer) BurnModuleStable(goCtx context.Context, msg *types.MsgBurnModuleStable) (*types.MsgBurnModuleStableResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Keeper 함수 호출
	err := m.Keeper.BurnModuleStableCoins(ctx)
	if err != nil {
		return nil, fmt.Errorf("burn failed: %w", err)
	}

	return &types.MsgBurnModuleStableResponse{}, nil
}
func (m msgServer) AddEnergy(goCtx context.Context, msg *types.MsgAddEnergy) (*types.MsgAddEnergyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Wh 단위 amount 파싱
	whAmt, ok := sdk.NewIntFromString(msg.Amount)
	if !ok {
		return nil, fmt.Errorf("invalid amount: %s", msg.Amount)
	}

	// Keeper 호출
	recs := m.Keeper.AddEnergy(ctx, msg.To, float64(whAmt.Int64()), msg.TxHash)

	// 모든 발전자별 누적량 조회 (contributor list 만들기)
	store := ctx.KVStore(m.Keeper.storeKey)
	var contributors []*types.Contributor

	iterator := sdk.KVStorePrefixIterator(store, []byte(types.EnergyByAddressPrefix))
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		addr := string(iterator.Key()[len(types.EnergyByAddressPrefix):]) // prefix 제거
		var amount float64
		_ = json.Unmarshal(iterator.Value(), &amount)

		contributors = append(contributors, &types.Contributor{
			Address:   addr,
			EnergyKwh: fmt.Sprintf("%f", amount),
		})
	}

	// contributors 를 JSON 문자열로 변환
	contribJSON, _ := json.Marshal(contributors)

	// 이벤트 발행
	ctx.EventManager().EmitEvent(
		sdk.NewEvent("add_energy",
			sdk.NewAttribute("creator", msg.Creator),
			sdk.NewAttribute("to", msg.To),
			sdk.NewAttribute("amount", msg.Amount),
			sdk.NewAttribute("tx_hash", msg.TxHash),
			sdk.NewAttribute("recs", fmt.Sprintf("%d", recs)),
			sdk.NewAttribute("contributors", string(contribJSON)), // 🔥 JSON 문자열
		),
	)

	return &types.MsgAddEnergyResponse{Recs: recs}, nil
}

func (m msgServer) CreateRECRecord(goCtx context.Context, msg *types.MsgCreateRECRecord) (*types.MsgCreateRECRecordResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Keeper 호출 → RECRecord 발급
	recIDs, err := m.Keeper.CreateRECRecord(ctx, msg.Count)
	if err != nil {
		return nil, err
	}
	// 이벤트 발행
	ctx.EventManager().EmitEvent(
		sdk.NewEvent("create_rec_record",
			sdk.NewAttribute("creator", msg.Creator),
			sdk.NewAttribute("count", fmt.Sprintf("%d", msg.Count)),
			sdk.NewAttribute("rec_id", recIDs[0]),
		),
	)

	// 단순 완료 응답 (필요시 RECRecord 반환 로직 추가 가능)
	return &types.MsgCreateRECRecordResponse{RecIds: recIDs}, nil
}

func (m msgServer) AppendTxHash(goCtx context.Context, msg *types.MsgAppendTxHash) (*types.MsgAppendTxHashResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// proto 정의에 맞춘 TxNode 생성
	node := types.TxNodeTx{
		TxHashes: msg.TxHashes,
		Creator:  msg.NodeCreator, // LinkedList 노드 생성자
		RecId:    msg.RecId,       // proto에 추가된 rec_id
		Next:     msg.Next,
	}

	if err := m.Keeper.AppendTxNode(ctx, node); err != nil {
		return nil, err
	}

	// 이벤트 발행
	ctx.EventManager().EmitEvent(
		sdk.NewEvent("append_tx_hash",
			sdk.NewAttribute("signer", msg.Creator),           // from=alice
			sdk.NewAttribute("node_creator", msg.NodeCreator), // LinkedList node 생성자
			sdk.NewAttribute("rec_id", msg.RecId),
			sdk.NewAttribute("hashes", strings.Join(msg.TxHashes, ",")),
		),
	)

	// proto 응답 구조에 맞게 node 반환
	return &types.MsgAppendTxHashResponse{
		Node: &node,
	}, nil
}
