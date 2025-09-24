package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors" // 👈 이 줄 추가
	lighttype "github.com/cosmos/cosmos-sdk/x/light_tx/types"

	"github.com/cosmos/cosmos-sdk/x/reward/types"
	gogoproto "github.com/gogo/protobuf/proto"
)

type msgServer struct {
	Keeper
	lighttype.UnimplementedMsgServer
}

func NewMsgServerImpl(k Keeper) lighttype.MsgServer {
	return &msgServer{Keeper: k}
}

func (k *msgServer) SendLightTx(goCtx context.Context, msg *lighttype.MsgSendLightTx) (*lighttype.MsgSendLightTxResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	switch payload := msg.Payload.(type) {
	case *lighttype.MsgSendLightTx_Original:
		data := payload.Original
		k.Logger(ctx).Info("📩 Received LightTx (SolarData)",
			"creator", msg.Creator,
			"device_id", data.DeviceId,
			"timestamp", data.Timestamp,
			"total_energy", data.TotalEnergy,
			"latitude", data.Location.Latitude,
			"longitude", data.Location.Longitude,
			"hash", msg.Hash,
			"signature", msg.Signature,
			"pubkey", msg.Pubkey,
		)

		// 이벤트 기록
		ctx.EventManager().EmitEvent(
			sdk.NewEvent("light_tx_solar",
				sdk.NewAttribute("creator", msg.Creator),
				sdk.NewAttribute("device_id", data.DeviceId),
				sdk.NewAttribute("hash", msg.Hash),
				sdk.NewAttribute("timestamp", data.Timestamp),
				sdk.NewAttribute("total_energy", fmt.Sprintf("%f", data.TotalEnergy)),
			),
		)

	case *lighttype.MsgSendLightTx_Rec:
		data := payload.Rec
		k.Logger(ctx).Info("📩 Received LightTx (RECMeta)",
			"creator", msg.Creator,
			"facility_id", data.FacilityId,
			"facility_name", data.FacilityName,
			"location", data.Location,
			"technology_type", data.TechnologyType,
			"capacity_mw", data.CapacityMw,
			"registration_date", data.RegistrationDate,
			"certified_id", data.CertifiedId,
			"issue_date", data.IssueDate,
			"generation_start", data.GenerationStartDate,
			"generation_end", data.GenerationEndDate,
			"measured_volume", data.MeasuredVolume_MWh,
			"retired_date", data.RetiredDate,
			"retirement_purpose", data.RetirementPurpose,
			"status", data.Status,
			"timestamp", data.Timestamp,
			"hash", msg.Hash,
			"signature", msg.Signature,
			"pubkey", msg.Pubkey,
		)

		// RECMeta 직렬화 및 저장
		bz, err := gogoproto.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal RECMeta: %w", err)
		}

		store := ctx.KVStore(k.storeKey)
		key := []byte(fmt.Sprintf("%s/%s", types.RecMetaKey, data.CertifiedId))
		store.Set(key, bz)

		ctx.Logger().Info("[LightTx] RECMeta 저장 완료", "certified_id", data.CertifiedId)

		ctx.EventManager().EmitEvent(
			sdk.NewEvent("light_tx_recmeta",
				sdk.NewAttribute("creator", msg.Creator),
				sdk.NewAttribute("facility_id", data.FacilityId),
				sdk.NewAttribute("facility_name", data.FacilityName),
				sdk.NewAttribute("location", data.Location),
				sdk.NewAttribute("technology_type", data.TechnologyType),
				sdk.NewAttribute("capacity_mw", data.CapacityMw),
				sdk.NewAttribute("registration_date", data.RegistrationDate),
				sdk.NewAttribute("certified_id", data.CertifiedId),
				sdk.NewAttribute("issue_date", data.IssueDate),
				sdk.NewAttribute("generation_start", data.GenerationStartDate),
				sdk.NewAttribute("generation_end", data.GenerationEndDate),
				sdk.NewAttribute("measured_volume", data.MeasuredVolume_MWh),
				sdk.NewAttribute("retired_date", data.RetiredDate),
				sdk.NewAttribute("retirement_purpose", data.RetirementPurpose),
				sdk.NewAttribute("status", data.Status),
				sdk.NewAttribute("timestamp", data.Timestamp),
				sdk.NewAttribute("hash", msg.Hash),
				sdk.NewAttribute("signature", msg.Signature),
				sdk.NewAttribute("pubkey", msg.Pubkey),
			),
		)

	default:
		k.Logger(ctx).Error("Unknown payload type in MsgSendLightTx")
		return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "unknown payload type")
	}

	return &lighttype.MsgSendLightTxResponse{Result: "OK"}, nil
}
