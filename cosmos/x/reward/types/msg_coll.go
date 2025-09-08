package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgDepositCollateral{}

func NewMsgDepositCollateral(creator, amount string) *MsgDepositCollateral {
	return &MsgDepositCollateral{
		Creator: creator,
		Amount:  amount,
	}
}

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
	if msg.Amount == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "amount cannot be empty")
	}
	return nil
}
