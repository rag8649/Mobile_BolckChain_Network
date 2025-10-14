package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/reward/types"
)

// BurnModuleStableCoins : 모듈 계좌의 stable 코인 전체 소각
func (k Keeper) BurnModuleStableCoins(ctx sdk.Context) error {
	// 1. 모듈 계좌 주소 조회
	moduleAddr := k.AccountKeeper.GetModuleAddress(types.ModuleName)
	if moduleAddr == nil {
		return fmt.Errorf("[BurnModuleCoin] module account not found")
	}

	// 2. mcnl 코인 잔고 조회
	stableBalance := k.bankKeeper.GetBalance(ctx, moduleAddr, "mcnl")
	if stableBalance.Amount.IsZero() {
		ctx.Logger().Info("[BurnModuleCoin] 소각할 mcnl 코인 없음")
		return nil
	}

	// 3. 소각 실행
	coins := sdk.NewCoins(stableBalance)
	if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, coins); err != nil {
		return fmt.Errorf("[BurnModuleCoin] failed to burn: %w", err)
	}
	// 🔹 공급량 업데이트
	supply := k.GetSupply(ctx)
	minted, _ := sdk.NewIntFromString(supply.Minted)
	newTotal := minted.Sub(stableBalance.Amount)
	if newTotal.IsNegative() {
		newTotal = sdk.NewInt(0)
	}
	supply.Minted = newTotal.String()
	k.SetSupply(ctx, supply)

	ctx.Logger().Info("[BurnModuleCoin] 소각 및 공급량 갱신 완료",
		"burned", stableBalance.String(),
		"newTotal", newTotal.String(),
	)
	return nil
}
