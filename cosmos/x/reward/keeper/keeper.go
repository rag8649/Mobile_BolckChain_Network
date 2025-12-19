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

	if k.bankKeeper == nil {
		panic("bankKeeper is nil")
	}

	// 1. Wh 입력 변환
	whAmt, ok := sdk.NewIntFromString(amount)
	if !ok {
		return fmt.Errorf("잘못된 amount 형식: %s", amount)
	}

	// 2. Wh → stable 변환 (1 stable = 1Wh)
	stableUnit := sdk.NewInt(1)
	coinAmt := whAmt.Quo(stableUnit) // 발행할 stable 수량
	ctx.Logger().Info("[RewardSolarPower] Coin 수량 계산", "coinAmt", coinAmt.String())

	// 3. 담보 조회 (REC 단위)
	collateralAmt, err := k.GetTotalCollateral(ctx) // REC 개수
	if err != nil {
		ctx.Logger().Error("[RewardSolarPower] 담보 조회 실패")
	}
	supply := k.GetSupply(ctx)
	minted, _ := sdk.NewIntFromString(supply.Minted)

	newTotal := minted.Add(coinAmt)

	// 담보 가치 = REC 개수 × 1000000 stable (1REC = 1,000,000 stable)
	collateralValueStable := collateralAmt.Mul(sdk.NewInt(1000000))

	if newTotal.GT(collateralValueStable) {
		return fmt.Errorf("[RewardSolarPower] 발행량 초과: 담보 부족 (collateral=%s REC → %s aeil, minted=%s, requested=%s)",
			collateralAmt.String(), collateralValueStable.String(), minted.String(), coinAmt.String())
	}

	// 4. stable 발행 및 전송
	coinsTotal := sdk.NewCoins(sdk.NewCoin("aeil", coinAmt))

	// 수수료 10%
	feeAmt := coinAmt.ToDec().Mul(sdk.NewDecWithPrec(1, 1)).TruncateInt() // coinAmt * 0.1
	feeCoins := sdk.NewCoins(sdk.NewCoin("aeil", feeAmt))

	// 사용자 금액
	userAmt := coinAmt.Sub(feeAmt)
	userCoins := sdk.NewCoins(sdk.NewCoin("aeil", userAmt))

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

	return nil
}
