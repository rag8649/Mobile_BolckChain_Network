package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/reward/types"
)

// DepositCollateral : stake를 소각하고 KVStore에 담보 총량 반영
func (k Keeper) DepositCollateral(ctx sdk.Context, from string, amount string) error {
	coin, err := sdk.ParseCoinNormalized(amount)
	if err != nil {
		return fmt.Errorf("잘못된 코인 형식: %s", amount)
	}
	if coin.Denom != "stake" {
		return fmt.Errorf("담보는 stake만 가능합니다: %s", coin.Denom)
	}

	fromAddr, err := sdk.AccAddressFromBech32(from)
	if err != nil {
		return err
	}

	// 1. from → 모듈 계정 전송 후 소각
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, fromAddr, types.ModuleName, sdk.NewCoins(coin)); err != nil {
		return err
	}
	if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(coin)); err != nil {
		return err
	}

	// 2. 전체 담보 총량 갱신
	oldAmt := k.GetTotalCollateral(ctx)
	newAmt := oldAmt.Add(coin.Amount)
	k.SetTotalCollateral(ctx, newAmt)

	ctx.Logger().Info("[Collateral] 총량 갱신",
		"prev", oldAmt.String(),
		"add", coin.Amount.String(),
		"new", newAmt.String(),
	)

	return nil
}

// GetTotalCollateral : 전체 담보량을 sdk.Int 로 반환
func (k Keeper) GetTotalCollateral(ctx sdk.Context) sdk.Int {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get([]byte(types.CollateralKey))
	if bz == nil {
		return sdk.ZeroInt()
	}
	amt, ok := sdk.NewIntFromString(string(bz))
	if !ok {
		return sdk.ZeroInt()
	}
	return amt
}

// SetTotalCollateral : sdk.Int 를 string으로 저장
func (k Keeper) SetTotalCollateral(ctx sdk.Context, amt sdk.Int) {
	store := ctx.KVStore(k.storeKey)
	store.Set([]byte(types.CollateralKey), []byte(amt.String()))
}

func (k Keeper) SetSupply(ctx sdk.Context, s types.Supply) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&s)
	store.Set([]byte(types.SupplyKey), bz)
}

func (k Keeper) GetSupply(ctx sdk.Context) types.Supply {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get([]byte(types.SupplyKey))
	if bz == nil {
		return types.Supply{Minted: "0"} // 기본값 "0"
	}
	var s types.Supply
	k.cdc.MustUnmarshal(bz, &s)
	if s.Minted == "" {
		s.Minted = "0"
	}
	return s
}
