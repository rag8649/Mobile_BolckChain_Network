package keeper

import (
	"context"

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

	// Creator(서명자) → 권한 확인용
	// TargetAddr → 실제 소각 대상
	err := m.Keeper.BurnStableCoin(ctx, msg.TargetAddr, msg.Amount)
	if err != nil {
		return nil, err
	}
	return &types.MsgBurnStableCoinResponse{}, nil
}

func (m msgServer) DepositCollateral(goCtx context.Context, msg *types.MsgDepositCollateral) (*types.MsgDepositCollateralResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := m.Keeper.DepositCollateral(ctx, msg.Creator, msg.Amount); err != nil {
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
