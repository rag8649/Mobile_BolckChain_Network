package keeper

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/reward/types"
)

type Keeper struct {
	storeKey      sdk.StoreKey
	cdc           codec.BinaryCodec
	bankKeeper    types.BankKeeper // 인터페이스로 정의 필요
	AccountKeeper types.AccountKeeper
}

func NewKeeper(cdc codec.BinaryCodec, key sdk.StoreKey, bankKeeper types.BankKeeper, accountKeeper types.AccountKeeper) Keeper {
	return Keeper{
		storeKey:      key,
		cdc:           cdc,
		bankKeeper:    bankKeeper,
		AccountKeeper: accountKeeper,
	}
}

func (k Keeper) RewardSolarPower(ctx sdk.Context, to string, amount string) error {
	ctx.Logger().Info("[RewardSolarPower] 함수 호출 시작",
		"to", to,
		"amount_raw", amount,
	)

	if k.bankKeeper == nil {
		ctx.Logger().Error("[RewardSolarPower] bankKeeper is nil")
		panic("bankKeeper is nil")
	}

	amt, ok := sdk.NewIntFromString(amount)
	if !ok {
		ctx.Logger().Error("[RewardSolarPower] amount 변환 실패", "amount", amount)
		return fmt.Errorf("잘못된 amount 형식: %s", amount)
	}
	ctx.Logger().Info("[RewardSolarPower] amount 변환 성공", "amt", amt.String())

	// 1. 담보/발행량 조회
	collateralAmt := k.GetTotalCollateral(ctx)
	ctx.Logger().Info("[RewardSolarPower] 담보 조회 완료", "collateralAmt", collateralAmt.String())

	supply := k.GetSupply(ctx)
	minted, _ := sdk.NewIntFromString(supply.Minted)
	ctx.Logger().Info("[RewardSolarPower] 현재 발행량 조회", "minted", minted.String())

	// 2. 발행 후 총량
	newTotal := minted.Add(amt)
	ctx.Logger().Info("[RewardSolarPower] 신규 발행량 계산",
		"minted", minted.String(),
		"amt", amt.String(),
		"newTotal", newTotal.String(),
	)

	// 3. 담보 비율 체크
	if newTotal.GT(collateralAmt.Mul(sdk.NewInt(1))) {
		ctx.Logger().Error("[RewardSolarPower] 🚫 발행량 초과",
			"collateralAmt", collateralAmt.String(),
			"minted", minted.String(),
			"requested", amt.String(),
			"newTotal", newTotal.String(),
		)
		return fmt.Errorf("[RewardSolarPower] 발행량 초과: 담보 부족 (collateral=%s, minted=%s, requested=%s)",
			collateralAmt.String(), minted.String(), amt.String())
	}

	// 4. stable 발행 및 전송
	coins := sdk.NewCoins(sdk.NewCoin("stable", amt))
	toAddr, err := sdk.AccAddressFromBech32(to)
	if err != nil {
		ctx.Logger().Error("[RewardSolarPower] 주소 변환 실패", "to", to, "err", err)
		return err
	}
	ctx.Logger().Info("[RewardSolarPower] 주소 변환 성공", "toAddr", toAddr.String())

	if err := k.bankKeeper.MintCoins(ctx, types.ModuleName, coins); err != nil {
		ctx.Logger().Error("[RewardSolarPower] MintCoins 실패", "coins", coins.String(), "err", err)
		return err
	}
	ctx.Logger().Info("[RewardSolarPower] MintCoins 성공", "coins", coins.String())

	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, toAddr, coins); err != nil {
		ctx.Logger().Error("[RewardSolarPower] SendCoins 실패", "toAddr", toAddr.String(), "err", err)
		return err
	}
	ctx.Logger().Info("[RewardSolarPower] SendCoins 성공", "toAddr", toAddr.String(), "coins", coins.String())

	// 5. 발행량 갱신
	supply.Minted = newTotal.String()
	k.SetSupply(ctx, supply)
	ctx.Logger().Info("[RewardSolarPower] 발행량 갱신 완료",
		"newTotal", newTotal.String(),
	)

	ctx.Logger().Info("[RewardSolarPower] 함수 종료")
	return nil
}
