package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// NewMsgBurnModuleStable : 생성자
func NewMsgBurnModuleStable(creator string, amount string) *MsgBurnModuleStable {
	return &MsgBurnModuleStable{
		Creator: creator,
		Amount:  amount,
	}
}

// Route implements sdk.Msg
func (msg *MsgBurnModuleStable) Route() string {
	return RouterKey
}

// Type implements sdk.Msg
func (msg *MsgBurnModuleStable) Type() string {
	return "burn_module_stable"
}

// GetSigners implements sdk.Msg
func (msg *MsgBurnModuleStable) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

// GetSignBytes implements sdk.Msg
func (msg *MsgBurnModuleStable) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic implements sdk.Msg
func (msg *MsgBurnModuleStable) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	if len(msg.Amount) == 0 {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidCoins, "amount cannot be empty")
	}

	if _, ok := sdk.NewIntFromString(msg.Amount); !ok {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidCoins, "amount must be an integer string")
	}

	return nil
}
