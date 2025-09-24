package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgDepositCollateral{}

func NewMsgDepositCollateralWithMeta(creator string, meta RECMeta) *MsgDepositCollateral {
	return &MsgDepositCollateral{
		Creator: creator,
		RecType: &MsgDepositCollateral_RecMeta{RecMeta: &meta},
	}
}

// NewMsgDepositCollateralWithRecord : 내부 RECRecord 담보 예치
func NewMsgDepositCollateralWithRecord(creator string, rec RECRecord) *MsgDepositCollateral {
	return &MsgDepositCollateral{
		Creator: creator,
		RecType: &MsgDepositCollateral_RecRecord{RecRecord: &rec},
	}
}

// 표준 Cosmos SDK Msg 인터페이스 구현
func (msg *MsgDepositCollateral) Route() string { return ModuleName }
func (msg *MsgDepositCollateral) Type() string  { return "DepositCollateral" }

func (msg *MsgDepositCollateral) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgDepositCollateral) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgDepositCollateral) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if msg.GetRecRecord() == nil && msg.GetRecMeta() == nil {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "either rec_record or rec_meta must be provided")
	}

	if msg.GetRecRecord() != nil && msg.GetRecMeta() != nil {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "only one of rec_record or rec_meta must be provided")
	}

	return nil
}
