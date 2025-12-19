package keeper

import (
	"encoding/json"
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/reward/types"
	gogoproto "github.com/gogo/protobuf/proto"
)

// 발전량 기록 및 REC 계산
func (k Keeper) AddEnergy(ctx sdk.Context, creator string, amountKwh float64, txHash string) int64 {
	store := ctx.KVStore(k.storeKey)

	// ---- 1. 총 에너지 불러오기 ----
	bz := store.Get([]byte(types.TotalEnergyKey))
	var total float64
	if bz != nil {
		_ = json.Unmarshal(bz, &total)
	}
	total += amountKwh
	store.Set([]byte(types.TotalEnergyKey), []byte(fmt.Sprintf("%f", total)))
	ctx.Logger().Info("[AddEnergy] 누적 total 계산", "total", total)

	// ---- 2. 발전자별 누적량 업데이트 ----
	addrKey := []byte(fmt.Sprintf("%s/%s", types.EnergyByAddressPrefix, creator))
	bzAddr := store.Get(addrKey)
	var addrTotal float64
	if bzAddr != nil {
		_ = json.Unmarshal(bzAddr, &addrTotal)
	}
	addrTotal += amountKwh
	store.Set(addrKey, []byte(fmt.Sprintf("%f", addrTotal)))
	ctx.Logger().Info("[AddEnergy] 발전자별 누적량 갱신", "creator", creator, "addrTotal", addrTotal)

	// ---- 3. Tx 해시 리스트 저장 ----
	bzTxs := store.Get([]byte(types.EnergyTxKey))
	var txList []string
	if bzTxs != nil {
		_ = json.Unmarshal(bzTxs, &txList)
	}
	txList = append(txList, txHash)
	newBz, _ := json.Marshal(txList)
	store.Set([]byte(types.EnergyTxKey), newBz)

	// ---- 4. REC 계산 ----
	recs := int64(total / 1000000.0) // 1REC = 1,000,000Wh
	if recs > 0 {
		remaining := total - float64(recs*1000000.0)
		store.Set([]byte(types.TotalEnergyKey), []byte(fmt.Sprintf("%f", remaining)))
		ctx.Logger().Info("[AddEnergy] REC 발급 가능", "recs", recs, "remaining", remaining)
	}

	return recs
}

func (k Keeper) CreateRECRecord(ctx sdk.Context, count int64) ([]string, error) {
	store := ctx.KVStore(k.storeKey)
	var recIDs []string

	for i := int64(0); i < count; i++ {
		// 1. 모든 발전자별 누적량 조회
		var contributors []*types.Contributor
		iterator := sdk.KVStorePrefixIterator(store, []byte(types.EnergyByAddressPrefix))
		defer iterator.Close()

		for ; iterator.Valid(); iterator.Next() {
			addr := string(iterator.Key()[len(types.EnergyByAddressPrefix):])
			var amount float64
			_ = json.Unmarshal(iterator.Value(), &amount)

			contributors = append(contributors, &types.Contributor{
				Address:   addr,
				EnergyKwh: fmt.Sprintf("%f", amount),
			})
		}

		// 2. Tx 해시 리스트 불러오기
		bzTxs := store.Get([]byte(types.EnergyTxKey))
		var txList []string
		_ = json.Unmarshal(bzTxs, &txList)

		// 3. RECRecord 생성
		recID := fmt.Sprintf("rec-%d-%d", ctx.BlockHeight(), i)
		rec := types.RECRecord{
			RecId:        recID,
			IssuedAt:     ctx.BlockTime().UTC().Format(time.RFC3339),
			BlockHeight:  ctx.BlockHeight(),
			TotalEnergy:  "1000000",
			Contributors: contributors,
			SourceTx:     txList,
		}

		bz, err := gogoproto.Marshal(&rec)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal RECRecord: %w", err)
		}
		key := []byte(fmt.Sprintf("%s/%s", types.RecRecordKey, rec.RecId))
		store.Set(key, bz)

		ctx.Logger().Info("[LightTx] RECRecord 생성", "rec_id", rec.RecId)
		recIDs = append(recIDs, recID)

	}

	return recIDs, nil
}
