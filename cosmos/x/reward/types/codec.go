package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)

// BankKeeper defines the expected bank keeper (noalias)
type BankKeeper interface {
	// 이미 발행된 코인 전송
	SendCoinsFromModuleToAccount(ctx sdk.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx sdk.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error

	// 코인 발행 및 소각
	MintCoins(ctx sdk.Context, moduleName string, amt sdk.Coins) error
	BurnCoins(ctx sdk.Context, moduleName string, amt sdk.Coins) error
}

func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgRewardSolarPower{}, "reward/MsgRewardSolarPower", nil)
	cdc.RegisterConcrete(&MsgBurnStableCoin{}, "reward/MsgBurnStableCoin", nil)
	cdc.RegisterConcrete(&MsgDepositCollateral{}, "reward/MsgDepositCollateral", nil)
	cdc.RegisterConcrete(&MsgRemoveCollateral{}, "reward/MsgRemoveCollateral", nil) // ✅ 수정
}

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgRewardSolarPower{}, "reward/RewardSolarPower", nil)
	cdc.RegisterConcrete(&MsgBurnStableCoin{}, "reward/MsgBurnStableCoin", nil)
	cdc.RegisterConcrete(&MsgDepositCollateral{}, "reward/MsgDepositCollateral", nil)
	cdc.RegisterConcrete(&MsgRemoveCollateral{}, "reward/MsgRemoveCollateral", nil) // ✅ 추가
}

func RegisterInterfaces(reg cdctypes.InterfaceRegistry) {
	reg.RegisterImplementations((*sdk.Msg)(nil),
		&MsgRewardSolarPower{},
		&MsgBurnStableCoin{},
		&MsgDepositCollateral{},
		&MsgRemoveCollateral{}, // ✅ 추가
	)
}
