package keeper

import (
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/reward/types"
	gogoproto "github.com/gogo/protobuf/proto"
)

func (k Keeper) BurnStableCoin(ctx sdk.Context, target string, amount string) (*types.MsgBurnStableCoinResponse, error) {
	var returnedRecords []*types.RECRecord
	var returnedMetas []*types.RECMeta

	if !strings.HasSuffix(amount, "aeil") {
		amount = amount + "aeil"
	}

	coin, err := sdk.ParseCoinNormalized(amount)
	if err != nil {
		return nil, fmt.Errorf("[Burn] 잘못된 코인 형식: %s, err: %v", amount, err)
	}

	targetAddr, err := sdk.AccAddressFromBech32(target)
	if err != nil {
		return nil, fmt.Errorf("[Burn] 잘못된 주소 형식: %s, err: %v", target, err) //
	} // 1. stable → 모듈 계정 이동 후 소각
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, targetAddr, types.ModuleName, sdk.NewCoins(coin)); err != nil {
		return nil, err
	}
	if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(coin)); err != nil {
		return nil, err
	}
	ctx.Logger().Info("[Burn] Coin 소각 완료", "amount", coin.String())

	// 2. 소각된 stable에 상응하는 REC 개수 계산
	oneREC := sdk.NewInt(1000000) // 정책적으로 정한 비율
	recToReturn := coin.Amount.Quo(oneREC)

	if recToReturn.IsPositive() {
		returned := int64(0)
		store := ctx.KVStore(k.storeKey)

		// RECRecord 반환 + 삭제
		iterRecord := sdk.KVStorePrefixIterator(store, []byte(types.RecRecordKey+"/"))
		defer iterRecord.Close()

		for ; iterRecord.Valid() && returned < recToReturn.Int64(); iterRecord.Next() {
			var rec types.RECRecord
			if err := gogoproto.Unmarshal(iterRecord.Value(), &rec); err == nil {
				ctx.Logger().Info("[Burn] REC 반환",
					"to", target,
					"rec_id", rec.RecId,
					"energy", rec.TotalEnergy,
				)
				returnedRecords = append(returnedRecords, &rec)
			}
			store.Delete(iterRecord.Key())
			returned++
		}

		// RECMeta 반환 + 삭제
		if returned < recToReturn.Int64() {
			iterMeta := sdk.KVStorePrefixIterator(store, []byte(types.RecMetaKey+"/"))
			defer iterMeta.Close()

			for ; iterMeta.Valid() && returned < recToReturn.Int64(); iterMeta.Next() {
				var meta types.RECMeta
				if err := gogoproto.Unmarshal(iterMeta.Value(), &meta); err == nil {
					ctx.Logger().Info("[Burn] RECMeta 반환",
						"to", target,
						"certified_id", meta.CertifiedId,
						"measured_volume", meta.MeasuredVolumeMwh,
					)
					returnedMetas = append(returnedMetas, &meta)
				}
				store.Delete(iterMeta.Key())
				returned++
			}
		}

		if returned < recToReturn.Int64() {
			return nil, fmt.Errorf("[Burn] 담보 부족: 요청=%d, 반환=%d", recToReturn.Int64(), returned)
		}
	}

	// 3. Supply(총 stable 발행량) 감소
	supply := k.GetSupply(ctx)
	oldSupply, _ := sdk.NewIntFromString(supply.Minted)
	newSupply := oldSupply.Sub(coin.Amount)
	if newSupply.IsNegative() {
		newSupply = sdk.ZeroInt()
	}
	supply.Minted = newSupply.String()
	k.SetSupply(ctx, supply)

	ctx.Logger().Info("[Burn] Supply Update",
		"prev", oldSupply.String(),
		"sub", coin.Amount.String(),
		"new", supply.Minted,
	)

	// ✅ 반환된 REC 목록을 응답에 담아서 리턴
	resp := &types.MsgBurnStableCoinResponse{
		RecRecords: returnedRecords,
		RecMetas:   returnedMetas,
	}

	// 🔥 최종 결과 전체 로그 출력
	for _, rec := range resp.RecRecords {
		ctx.Logger().Info("[Burn:Result] RECRecord",
			"rec_id", rec.RecId,
			"issued_at", rec.IssuedAt,
			"block_height", rec.BlockHeight,
			"total_energy", rec.TotalEnergy,
			"contributors", rec.Contributors,
			"source_tx", rec.SourceTx,
		)
	}
	for _, meta := range resp.RecMetas {
		ctx.Logger().Info("[Burn:Result] RECMeta",
			"facility_id", meta.FacilityId,
			"facility_name", meta.FacilityName,
			"location", meta.Location,
			"capacity_mw", meta.CapacityMw,
			"certified_id", meta.CertifiedId,
			"status", meta.Status,
			"timestamp", meta.Timestamp,
		)
	}

	return resp, nil
}
