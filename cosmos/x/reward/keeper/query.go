package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/reward/types"
)

var _ types.QueryServer = Querier{}

type Querier struct {
	Keeper
}

// Collateral 쿼리
func (q Querier) Collateral(ctx context.Context, req *types.QueryCollateralRequest) (*types.QueryCollateralResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// 단일 총 담보량 가져오기
	collateral := q.GetTotalCollateral(sdkCtx)

	return &types.QueryCollateralResponse{
		TotalAmount: collateral.String(),
	}, nil
}

// Supply 쿼리
func (q Querier) Supply(ctx context.Context, req *types.QuerySupplyRequest) (*types.QuerySupplyResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	supply := q.GetSupply(sdkCtx) // ← GetSupply 함수 필요
	return &types.QuerySupplyResponse{Supply: supply}, nil
}
