package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RemoveCollateral : 전체 담보량에서 특정 금액을 차감
func (k Keeper) RemoveCollateral(ctx sdk.Context, from string, amount string) error {
	coin, err := sdk.ParseCoinNormalized(amount)
	if err != nil {
		return fmt.Errorf("잘못된 코인 형식: %s", amount)
	}
	if coin.Denom != "stake" {
		return fmt.Errorf("담보는 stake만 가능합니다: %s", coin.Denom)
	}

	// 총 담보 차감
	oldAmt := k.GetTotalCollateral(ctx)
	newAmt := oldAmt.Sub(coin.Amount)
	if newAmt.IsNegative() {
		return fmt.Errorf("차감 불가: 기존 담보 부족 (현재=%s, 요청=%s)", oldAmt.String(), coin.Amount.String())
	}

	k.SetTotalCollateral(ctx, newAmt)

	ctx.Logger().Info("[Collateral] 총량 차감",
		"from", from,
		"prev", oldAmt.String(),
		"sub", coin.Amount.String(),
		"new", newAmt.String(),
	)

	return nil
}
