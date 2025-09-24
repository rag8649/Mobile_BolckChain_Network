package keeper

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/fullnode_bridge/config"
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

// RewardSolarPower 수정본
func (k Keeper) RewardSolarPower(ctx sdk.Context, to string, amount string) error {
	ctx.Logger().Info("[RewardSolarPower] 함수 호출 시작",
		"to", to,
		"amount_raw", amount,
	)

	if k.bankKeeper == nil {
		ctx.Logger().Error("[RewardSolarPower] bankKeeper is nil")
		panic("bankKeeper is nil")
	}

	// Wh 단위 입력값 변환
	whAmt, ok := sdk.NewIntFromString(amount)
	if !ok {
		ctx.Logger().Error("[RewardSolarPower] amount 변환 실패", "amount", amount)
		return fmt.Errorf("잘못된 amount 형식: %s", amount)
	}
	ctx.Logger().Info("[RewardSolarPower] amount 변환 성공 (Wh)", "whAmt", whAmt.String())
	// 1REC = 1,000,000 Wh
	recUnit := sdk.NewInt(1_000_000)

	// REC 개수 = Wh / 1,000,000
	recCount := whAmt.Quo(recUnit)

	// REC 가격 (원화)
	priceInt := sdk.NewInt(int64(config.CurrentRECPrice))

	// 총 원화 가치 = REC 개수 * REC 가격
	totalValueKRW := recCount.Mul(priceInt)

	// stable 변환 (1 stable = 100원)
	stableUnit := sdk.NewInt(100)
	stableAmt := totalValueKRW.Quo(stableUnit)
	ctx.Logger().Info("[RewardSolarPower] Stable 수량 계산", "stableAmt", stableAmt.String())

	// === 기존 로직 (담보 체크, 발행 등) ===
	collateralAmt, err := k.GetTotalCollateral(ctx)
	if err != nil {
		ctx.Logger().Error("[RewardSolarPower] 담보 조회 실패")
	}
	supply := k.GetSupply(ctx)
	minted, _ := sdk.NewIntFromString(supply.Minted)

	newTotal := minted.Add(stableAmt)

	// 담보 가치 = 담보 REC 수량 × REC 가격
	collateralValue := collateralAmt.Mul(priceInt)

	if newTotal.GT(collateralValue) {
		return fmt.Errorf("[RewardSolarPower] 발행량 초과: 담보 부족 (collateralValue=%s, minted=%s, requested=%s)",
			collateralValue.String(), minted.String(), stableAmt.String())
	}

	// 4. stable 발행 및 전송
	coinsTotal := sdk.NewCoins(sdk.NewCoin("stable", stableAmt))

	// 수수료 10%
	feeAmt := stableAmt.ToDec().Mul(sdk.NewDecWithPrec(1, 1)).TruncateInt() // stableAmt * 0.1
	feeCoins := sdk.NewCoins(sdk.NewCoin("stable", feeAmt))

	// 사용자에게 줄 금액
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

	// 사용자에게 전송
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, toAddr, userCoins); err != nil {
		return err
	}

	// 수수료는 모듈 계좌에 남김
	ctx.Logger().Info("[RewardSolarPower] 수수료 적립 완료", "feeCoins", feeCoins.String())

	// 발행량 갱신
	supply.Minted = newTotal.String()
	k.SetSupply(ctx, supply)

	ctx.Logger().Info("[RewardSolarPower] 함수 종료")
	return nil
}
