package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/reward/types"
)

// DistributeRewardPercent : 모듈 계좌 잔액에서 percent만큼을 address에게 지급
// func (k Keeper) DistributeRewardPercent(ctx sdk.Context, msg *types.MsgDistributeRewardPercent) error {
// 	// 1️⃣ Creator 유효성 검사
// 	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
// 		return fmt.Errorf("invalid creator address: %w", err)
// 	}

// 	// 2️⃣ 모듈 계좌 조회
// 	moduleAddr := k.AccountKeeper.GetModuleAddress(types.ModuleName)
// 	if moduleAddr == nil {
// 		return sdkerrors.Wrapf(sdkerrors.ErrUnknownAddress, "module account %s not found", types.ModuleName)
// 	}

// 	// ✅ denom을 reward 모듈이 사용하는 코인 이름으로 설정하세요 (예: "urec" 또는 "ustable")
// 	const denom = "aeil"

// 	// 3️⃣ 모듈 잔고 조회
// 	balance := k.bankKeeper.GetBalance(ctx, moduleAddr, denom)
// 	if balance.Amount.IsZero() {
// 		return sdkerrors.Wrap(sdkerrors.ErrInsufficientFunds, "module account has no balance")
// 	} else {
// 		// 8️⃣ 로그 + 이벤트
// 		ctx.Logger().Info("[Reward] 모듈 잔고", balance.Amount)
// 	}

// 	// 4️⃣ percent 문자열 → sdk.Dec 변환
// 	per, err := sdk.NewDecFromStr(msg.Percent)
// 	if err != nil {
// 		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "invalid percent format")
// 	}
// 	if per.LTE(sdk.ZeroDec()) || per.GT(sdk.OneDec()) {
// 		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "percent must be between 0 and 1")
// 	}

// 	// 5️⃣ 지급 금액 계산
// 	rewardAmt := sdk.NewDecFromInt(balance.Amount).Mul(per).TruncateInt()
// 	if !rewardAmt.IsPositive() {
// 		return sdkerrors.Wrap(sdkerrors.ErrInsufficientFunds, "reward amount is zero")
// 	}

// 	rewardCoin := sdk.NewCoin(denom, rewardAmt)
// 	rewardCoins := sdk.NewCoins(rewardCoin)

// 	// 6️⃣ 수신자 주소 변환
// 	addr, err := sdk.AccAddressFromBech32(msg.Address)
// 	if err != nil {
// 		return sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, "invalid recipient address")
// 	}

// 	// 7️⃣ 모듈 → address로 송금
// 	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, rewardCoins); err != nil {
// 		return err
// 	}

// 	// 8️⃣ 로그 + 이벤트
// 	ctx.Logger().Info("[Reward] DistributeRewardPercent 완료",
// 		"creator", msg.Creator,
// 		"recipient", msg.Address,
// 		"percent", msg.Percent,
// 		"distributed_amount", rewardCoin.String(),
// 	)

// 	ctx.EventManager().EmitEvents(sdk.Events{
// 		sdk.NewEvent(
// 			sdk.EventTypeMessage,
// 			sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
// 			sdk.NewAttribute(sdk.AttributeKeySender, msg.Creator),
// 			sdk.NewAttribute("recipient", msg.Address),
// 			sdk.NewAttribute("percent", msg.Percent),
// 			sdk.NewAttribute("distributed_amount", rewardCoin.String()),
// 		),
// 	})

// 	return nil
// }

func (k Keeper) DistributeRewardPercent(ctx sdk.Context, msg *types.MsgDistributeRewardPercent) error {
	// 1️⃣ Creator 유효성 검사
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return fmt.Errorf("invalid creator address: %w", err)
	}

	// 2️⃣ 모듈 계좌 조회
	moduleAddr := k.AccountKeeper.GetModuleAddress(types.ModuleName)
	if moduleAddr == nil {
		return sdkerrors.Wrapf(sdkerrors.ErrUnknownAddress, "module account %s not found", types.ModuleName)
	}

	const denom = "aeil"

	// 3️⃣ 모듈 잔고 조회
	balance := k.bankKeeper.GetBalance(ctx, moduleAddr, denom)
	if balance.Amount.LT(sdk.NewInt(10000)) {
		return sdkerrors.Wrap(sdkerrors.ErrInsufficientFunds, "module account has insufficient balance (< 10000aeil)")
	} else {
		ctx.Logger().Info("[Reward] 모듈 잔고", "amount", balance.Amount.String())
	}

	// ✅ 지급 금액을 고정값으로 설정 (10,000 aeil)
	rewardAmt := sdk.NewInt(10000)
	rewardCoin := sdk.NewCoin(denom, rewardAmt)
	rewardCoins := sdk.NewCoins(rewardCoin)

	// 6️⃣ 수신자 주소 변환
	addr, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, "invalid recipient address")
	}

	// 7️⃣ 모듈 → address로 송금
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, rewardCoins); err != nil {
		return err
	}

	// 8️⃣ 로그 + 이벤트
	ctx.Logger().Info("[Reward] DistributeRewardFixed 완료",
		"creator", msg.Creator,
		"recipient", msg.Address,
		"distributed_amount", rewardCoin.String(),
	)

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
			sdk.NewAttribute(sdk.AttributeKeySender, msg.Creator),
			sdk.NewAttribute("recipient", msg.Address),
			sdk.NewAttribute("distributed_amount", rewardCoin.String()),
		),
	})

	return nil
}
