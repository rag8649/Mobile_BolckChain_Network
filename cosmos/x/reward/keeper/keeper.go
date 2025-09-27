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
		panic("bankKeeper is nil")
	}

	// 1. Wh 입력 변환
	whAmt, ok := sdk.NewIntFromString(amount)
	if !ok {
		return fmt.Errorf("잘못된 amount 형식: %s", amount)
	}
	ctx.Logger().Info("[RewardSolarPower] amount 변환 성공 (Wh)", "whAmt", whAmt.String())

	// 2. Wh → stable 변환 (1 stable = 1Wh)
	stableUnit := sdk.NewInt(1)
	stableAmt := whAmt.Quo(stableUnit) // 발행할 stable 수량
	ctx.Logger().Info("[RewardSolarPower] Stable 수량 계산", "stableAmt", stableAmt.String())

	// 3. 담보 조회 (REC 단위)
	collateralAmt, err := k.GetTotalCollateral(ctx) // REC 개수
	if err != nil {
		ctx.Logger().Error("[RewardSolarPower] 담보 조회 실패")
	}
	supply := k.GetSupply(ctx)
	minted, _ := sdk.NewIntFromString(supply.Minted)

	newTotal := minted.Add(stableAmt)

	// 담보 가치 = REC 개수 × 1000 stable (1REC = 1,000,000 stable)
	collateralValueStable := collateralAmt.Mul(sdk.NewInt(1000000))

	if newTotal.GT(collateralValueStable) {
		return fmt.Errorf("[RewardSolarPower] 발행량 초과: 담보 부족 (collateral=%s REC → %s stable, minted=%s, requested=%s)",
			collateralAmt.String(), collateralValueStable.String(), minted.String(), stableAmt.String())
	}

	// 4. stable 발행 및 전송
	coinsTotal := sdk.NewCoins(sdk.NewCoin("stable", stableAmt))

	// 수수료 10%
	feeAmt := stableAmt.ToDec().Mul(sdk.NewDecWithPrec(1, 1)).TruncateInt() // stableAmt * 0.1
	feeCoins := sdk.NewCoins(sdk.NewCoin("stable", feeAmt))

	// 사용자 금액
	userAmt := stableAmt.Sub(feeAmt)
	userCoins := sdk.NewCoins(sdk.NewCoin("stable", userAmt))

	toAddr, err := sdk.AccAddressFromBech32(to)
	if err != nil {
		return err
	}

	// 발행
	if err := k.bankKeeper.MintCoins(ctx, types.ModuleName, coinsTotal); err != nil {
		return err
	}

	// 사용자에게 지급
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, toAddr, userCoins); err != nil {
		return err
	}

	ctx.Logger().Info("[RewardSolarPower] 수수료 적립 완료", "feeCoins", feeCoins.String())

	// 발행량 갱신
	supply.Minted = newTotal.String()
	k.SetSupply(ctx, supply)

	ctx.Logger().Info("[RewardSolarPower] 함수 종료")
	return nil
}
