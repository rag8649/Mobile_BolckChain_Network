package types

import (
	fmt "fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgAppendTxHash{}

// ✅ recID 추가
func NewMsgAppendTxHash(creator string, txHashes []string, nodeCreator string, recID string, next string) *MsgAppendTxHash {
	return &MsgAppendTxHash{
		Creator:     creator,
		TxHashes:    txHashes,
		NodeCreator: nodeCreator,
		RecId:       recID, // ✅ 새 필드
		Next:        next,
	}
}

func (msg *MsgAppendTxHash) Route() string { return RouterKey }
func (msg *MsgAppendTxHash) Type() string  { return "AppendTxHash" }

func (msg *MsgAppendTxHash) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{addr}
}

func (msg *MsgAppendTxHash) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return fmt.Errorf("invalid creator address: %w", err)
	}
	if msg.NodeCreator == "" {
		return fmt.Errorf("node_creator cannot be empty")
	}
	if msg.RecId == "" {
		return fmt.Errorf("rec_id cannot be empty") // ✅ 필수값 검증
	}
	return nil
}
