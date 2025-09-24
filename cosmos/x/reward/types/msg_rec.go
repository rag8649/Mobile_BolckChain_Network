package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgAddEnergy{}
var _ sdk.Msg = &MsgCreateRECRecord{}

// New 생성자
func NewMsgAddEnergy(creator, to, amount, txHash string) *MsgAddEnergy {
	return &MsgAddEnergy{
		Creator: creator,
		To:      to,
		Amount:  amount,
		TxHash:  txHash,
	}
}
func NewMsgCreateRECRecord(creator string, count int64) *MsgCreateRECRecord {
	return &MsgCreateRECRecord{
		Creator: creator,
		Count:   count,
	}
}

// sdk.Msg 인터페이스 구현 (AddEnergy)
func (msg *MsgAddEnergy) Route() string { return RouterKey }
func (msg *MsgAddEnergy) Type() string  { return "AddEnergy" }
func (msg *MsgAddEnergy) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{addr}
}
func (msg *MsgAddEnergy) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}
func (msg *MsgAddEnergy) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator: %s", err)
	}
	if msg.Amount == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "amount required")
	}
	return nil
}

// sdk.Msg 인터페이스 구현 (CreateRECRecord)
func (msg *MsgCreateRECRecord) Route() string { return RouterKey }
func (msg *MsgCreateRECRecord) Type() string  { return "CreateRECRecord" }
func (msg *MsgCreateRECRecord) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{addr}
}
func (msg *MsgCreateRECRecord) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}
func (msg *MsgCreateRECRecord) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator: %s", err)
	}
	if msg.Count <= 0 {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "count must be > 0")
	}
	return nil
}
