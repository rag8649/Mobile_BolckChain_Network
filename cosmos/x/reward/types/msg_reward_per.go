package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// 인터페이스 검증용 — 컴파일 시점 확인
var _ sdk.Msg = &MsgDistributeRewardPercent{}

// ✅ Msg 생성자
func NewMsgDistributeRewardPercent(creator, address, percent string) *MsgDistributeRewardPercent {
	return &MsgDistributeRewardPercent{
		Creator: creator,
		Address: address,
		Percent: percent,
	}
}

// ✅ Route — 모듈 라우트 이름
func (msg *MsgDistributeRewardPercent) Route() string {
	return ModuleName
}

// ✅ Type — 메시지 식별용 문자열
func (msg *MsgDistributeRewardPercent) Type() string {
	return "DistributeRewardPercent"
}

// ✅ GetSigners — 서명자 반환
func (msg *MsgDistributeRewardPercent) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err) // ValidateBasic에서 이미 검사되므로 panic 허용
	}
	return []sdk.AccAddress{creator}
}

// ✅ GetSignBytes — 서명용 바이트 정렬
func (msg *MsgDistributeRewardPercent) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

// ✅ ValidateBasic — 기본 유효성 검사
func (msg *MsgDistributeRewardPercent) ValidateBasic() error {
	// creator 유효성
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	// 수신자 주소 유효성
	if _, err := sdk.AccAddressFromBech32(msg.Address); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid recipient address (%s)", err)
	}

	// percent 값 유효성
	if msg.Percent == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "percent cannot be empty")
	}

	// 0보다 크고 1 이하인지 확인
	per, err := sdk.NewDecFromStr(msg.Percent)
	if err != nil {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "invalid percent format")
	}
	if per.LTE(sdk.ZeroDec()) || per.GT(sdk.OneDec()) {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "percent must be between 0 and 1")
	}

	return nil
}
