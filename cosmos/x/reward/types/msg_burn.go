package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgBurnStableCoin{}

func NewMsgBurnStableCoin(creator, targetAddr, amount string) *MsgBurnStableCoin {
	return &MsgBurnStableCoin{
		Creator:    creator,
		TargetAddr: targetAddr,
		Amount:     amount,
	}
}

func (msg *MsgBurnStableCoin) Route() string {
	return ModuleName
}

func (msg *MsgBurnStableCoin) Type() string {
	return "BurnStableCoin"
}

func (msg *MsgBurnStableCoin) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err) // CLI에서 이미 ValidateBasic을 거치므로 여기선 panic 허용
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgBurnStableCoin) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgBurnStableCoin) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	if msg.Amount == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "amount cannot be empty")
	}
	return nil
}
