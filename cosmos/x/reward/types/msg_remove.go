package types

import (
	fmt "fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgRemoveCollateral{}

func NewMsgRemoveCollateral(creator string, amount string) *MsgRemoveCollateral {
	return &MsgRemoveCollateral{
		Creator: creator,
		Amount:  amount,
	}
}

func (msg *MsgRemoveCollateral) Route() string { return RouterKey }
func (msg *MsgRemoveCollateral) Type() string  { return "RemoveCollateral" }

func (msg *MsgRemoveCollateral) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgRemoveCollateral) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgRemoveCollateral) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return err
	}
	if len(msg.Amount) == 0 {
		return fmt.Errorf("amount cannot be empty")
	}
	return nil
}
