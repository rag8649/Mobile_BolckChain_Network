package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/reward/types"
)

func (k Keeper) BurnStableCoin(ctx sdk.Context, target string, amount string) error {
	coin, err := sdk.ParseCoinNormalized(amount)
	if err != nil {
		return fmt.Errorf("[Burn] 잘못된 코인 형식: %s, err: %v", amount, err)
	}

	targetAddr, err := sdk.AccAddressFromBech32(target)
	if err != nil {
		return fmt.Errorf("[Burn] 잘못된 주소 형식: %s, err: %v", target, err)
	}

	// 1. stable → 모듈 계정 이동 후 소각
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, targetAddr, types.ModuleName, sdk.NewCoins(coin)); err != nil {
		return err
	}
	if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(coin)); err != nil {
		return err
	}

	ctx.Logger().Info("[Burn] StableCoin 소각 완료", "amount", coin.String())

	// 2. 담보 총량 차감 (1 stake = 1,000,000 stable)
	ratio := sdk.NewInt(1)
	stakeToConsume := coin.Amount.Quo(ratio)

	if stakeToConsume.IsPositive() {
		oldAmt := k.GetTotalCollateral(ctx)
		newAmt := oldAmt.Sub(stakeToConsume)
		if newAmt.IsNegative() {
			newAmt = sdk.ZeroInt()
		}
		k.SetTotalCollateral(ctx, newAmt)

		ctx.Logger().Info("[Burn] Collateral 총량 차감",
			"prev", oldAmt.String(),
			"sub", stakeToConsume.String(),
			"new", newAmt.String(),
		)
	}

	// 3. Supply(총 stable 발행량) 감소
	supply := k.GetSupply(ctx)
	oldSupply, _ := sdk.NewIntFromString(supply.Minted)
	newSupply := oldSupply.Sub(coin.Amount)
	if newSupply.IsNegative() {
		newSupply = sdk.ZeroInt()
	}
	supply.Minted = newSupply.String()
	k.SetSupply(ctx, supply)

	ctx.Logger().Info("[Burn] Supply Update",
		"prev", oldSupply.String(),
		"sub", coin.Amount.String(),
		"new", supply.Minted,
	)

	return nil
}
