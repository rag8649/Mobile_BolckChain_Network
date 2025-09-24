package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/reward/types"
	gogoproto "github.com/gogo/protobuf/proto" // ✅ gogo protobuf import
)

// DepositCollateral : RECRecord 또는 RECMeta를 KVStore에 저장
func (k Keeper) DepositCollateral(ctx sdk.Context, msg *types.MsgDepositCollateral) error {
	// Creator 유효성 확인
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return fmt.Errorf("invalid creator address: %w", err)
	}

	store := ctx.KVStore(k.storeKey)

	// RECRecord 저장
	if rec := msg.GetRecRecord(); rec != nil {
		bz, err := gogoproto.Marshal(rec)
		if err != nil {
			return fmt.Errorf("failed to marshal RECRecord: %w", err)
		}
		key := []byte(fmt.Sprintf("%s/%s", types.RecRecordKey, rec.RecId))
		store.Set(key, bz)

		ctx.Logger().Info("[Collateral] RECRecord 저장",
			"rec_id", rec.RecId,
			"block_height", rec.BlockHeight,
			"total_energy", rec.TotalEnergy,
		)
		return nil
	}

	// RECMeta 저장
	if meta := msg.GetRecMeta(); meta != nil {
		bz, err := gogoproto.Marshal(meta)
		if err != nil {
			return fmt.Errorf("failed to marshal RECMeta: %w", err)
		}
		key := []byte(fmt.Sprintf("%s/%s", types.RecMetaKey, meta.CertifiedId))
		store.Set(key, bz)

		ctx.Logger().Info("[Collateral] RECMeta 저장",
			"certified_id", meta.CertifiedId,
			"facility_id", meta.FacilityId,
			"measured_volume", meta.MeasuredVolumeMwh,
		)
		return nil
	}

	return fmt.Errorf("invalid message: neither RECRecord nor RECMeta provided")
}

// GetAllRECRecords : 저장된 모든 RECRecord 조회
func (k Keeper) GetAllRECRecords(ctx sdk.Context) ([]*types.RECRecord, error) {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, []byte(types.RecRecordKey+"/"))
	defer iterator.Close()

	var records []*types.RECRecord
	for ; iterator.Valid(); iterator.Next() {
		var rec types.RECRecord
		if err := gogoproto.Unmarshal(iterator.Value(), &rec); err != nil {
			return nil, err
		}
		records = append(records, &rec)
	}
	return records, nil
}

// GetAllRECMetas : 저장된 모든 RECMeta 조회
func (k Keeper) GetAllRECMetas(ctx sdk.Context) ([]*types.RECMeta, error) {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, []byte(types.RecMetaKey+"/"))
	defer iterator.Close()

	var metas []*types.RECMeta
	for ; iterator.Valid(); iterator.Next() {
		var meta types.RECMeta
		if err := gogoproto.Unmarshal(iterator.Value(), &meta); err != nil {
			return nil, err
		}
		metas = append(metas, &meta)
	}
	return metas, nil
}

// GetTotalCollateral : 저장된 REC 개수를 sdk.Int로 반환
func (k Keeper) GetTotalCollateral(ctx sdk.Context) (sdk.Int, error) {
	count := int64(0)

	store := ctx.KVStore(k.storeKey)

	iterRecord := sdk.KVStorePrefixIterator(store, []byte(types.RecRecordKey+"/"))
	defer iterRecord.Close()
	for ; iterRecord.Valid(); iterRecord.Next() {
		count++
	}

	iterMeta := sdk.KVStorePrefixIterator(store, []byte(types.RecMetaKey+"/"))
	defer iterMeta.Close()
	for ; iterMeta.Valid(); iterMeta.Next() {
		count++
	}

	return sdk.NewInt(count), nil
}

// SetSupply : Supply 구조체를 KVStore에 저장
func (k Keeper) SetSupply(ctx sdk.Context, s types.Supply) {
	store := ctx.KVStore(k.storeKey)
	bz, err := gogoproto.Marshal(&s)
	if err != nil {
		panic(fmt.Errorf("failed to marshal Supply: %w", err))
	}
	store.Set([]byte(types.SupplyKey), bz)
}

// GetSupply : KVStore에서 Supply 구조체를 불러오기
func (k Keeper) GetSupply(ctx sdk.Context) types.Supply {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get([]byte(types.SupplyKey))
	if bz == nil {
		return types.Supply{Minted: "0"} // 기본값
	}

	var s types.Supply
	if err := gogoproto.Unmarshal(bz, &s); err != nil {
		panic(fmt.Errorf("failed to unmarshal Supply: %w", err))
	}

	if s.Minted == "" {
		s.Minted = "0"
	}
	return s
}
