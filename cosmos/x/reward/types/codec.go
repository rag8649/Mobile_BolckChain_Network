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
	GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin

	// 이미 발행된 코인 전송
	SendCoinsFromModuleToAccount(ctx sdk.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx sdk.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error

	// 코인 발행 및 소각
	MintCoins(ctx sdk.Context, moduleName string, amt sdk.Coins) error
	BurnCoins(ctx sdk.Context, moduleName string, amt sdk.Coins) error
}

// Amino Codec 등록
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgRewardSolarPower{}, "reward/MsgRewardSolarPower", nil)
	cdc.RegisterConcrete(&MsgBurnStableCoin{}, "reward/MsgBurnStableCoin", nil)
	cdc.RegisterConcrete(&MsgDepositCollateral{}, "reward/MsgDepositCollateral", nil)
	cdc.RegisterConcrete(&MsgRemoveCollateral{}, "reward/MsgRemoveCollateral", nil)
	cdc.RegisterConcrete(&MsgBurnModuleStable{}, "reward/MsgBurnModuleStable", nil)
	cdc.RegisterConcrete(&MsgAddEnergy{}, "reward/MsgAddEnergy", nil)
	cdc.RegisterConcrete(&MsgCreateRECRecord{}, "reward/MsgCreateRECRecord", nil)
	cdc.RegisterConcrete(&MsgAppendTxHash{}, "reward/MsgAppendTxHash", nil)
	cdc.RegisterConcrete(&MsgDistributeRewardPercent{}, "reward/MsgDistributeRewardPercent", nil)

	// ✅ REC 구조체 추가 등록
	cdc.RegisterConcrete(&RECRecord{}, "reward/RECRecord", nil)
	cdc.RegisterConcrete(&RECMeta{}, "reward/RECMeta", nil)
	cdc.RegisterConcrete(&Contributor{}, "reward/Contributor", nil)
}

// Legacy Codec 등록
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgRewardSolarPower{}, "reward/RewardSolarPower", nil)
	cdc.RegisterConcrete(&MsgBurnStableCoin{}, "reward/MsgBurnStableCoin", nil)
	cdc.RegisterConcrete(&MsgDepositCollateral{}, "reward/MsgDepositCollateral", nil)
	cdc.RegisterConcrete(&MsgRemoveCollateral{}, "reward/MsgRemoveCollateral", nil)
	cdc.RegisterConcrete(&MsgBurnModuleStable{}, "reward/MsgBurnModuleStable", nil)
	cdc.RegisterConcrete(&MsgAddEnergy{}, "reward/MsgAddEnergy", nil)
	cdc.RegisterConcrete(&MsgCreateRECRecord{}, "reward/MsgCreateRECRecord", nil)
	cdc.RegisterConcrete(&MsgAppendTxHash{}, "reward/MsgAppendTxHash", nil)
	cdc.RegisterConcrete(&MsgDistributeRewardPercent{}, "reward/MsgDistributeRewardPercent", nil)
	// ✅ REC 구조체 추가 등록
	cdc.RegisterConcrete(&RECRecord{}, "reward/RECRecord", nil)
	cdc.RegisterConcrete(&RECMeta{}, "reward/RECMeta", nil)
	cdc.RegisterConcrete(&Contributor{}, "reward/Contributor", nil)
}

// Proto Codec 등록
func RegisterInterfaces(reg cdctypes.InterfaceRegistry) {
	reg.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgRewardSolarPower{},
		&MsgBurnStableCoin{},
		&MsgDepositCollateral{},
		&MsgRemoveCollateral{},
		&MsgBurnModuleStable{},
		&MsgAddEnergy{},
		&MsgCreateRECRecord{},
		&MsgAppendTxHash{},
		&MsgDistributeRewardPercent{},
	)
}
