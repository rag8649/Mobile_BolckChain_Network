package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/reward/types"
)

// RemoveCollateral : 특정 stable 양에 상응하는 REC를 삭제
func (k Keeper) RemoveCollateral(ctx sdk.Context, from string, amount string) error {
	coin, err := sdk.ParseCoinNormalized(amount)
	if err != nil {
		return fmt.Errorf("[Remove] 잘못된 코인 형식: %s", amount)
	}
	if coin.Denom != "mcnl" {
		return fmt.Errorf("[Remove] 담보 차감은 mcnl 기준만 가능합니다: %s", coin.Denom)
	}

	// 1. stable → REC 개수 환산 (1 REC = 1,000,000 stable)
	oneREC := sdk.NewInt(1000) // 정책적으로 정의
	recToDelete := coin.Amount.Quo(oneREC)
	if recToDelete.IsZero() {
		return fmt.Errorf("[Remove] REC 단위 미만은 삭제 불가: %s", amount)
	}

	store := ctx.KVStore(k.storeKey)
	deleted := int64(0)

	// 2. RECRecord 삭제
	iterRecord := sdk.KVStorePrefixIterator(store, []byte(types.RecRecordKey+"/"))
	defer iterRecord.Close()
	for ; iterRecord.Valid() && deleted < recToDelete.Int64(); iterRecord.Next() {
		store.Delete(iterRecord.Key())
		deleted++
	}

	// 3. RECMeta 삭제
	if deleted < recToDelete.Int64() {
		iterMeta := sdk.KVStorePrefixIterator(store, []byte(types.RecMetaKey+"/"))
		defer iterMeta.Close()
		for ; iterMeta.Valid() && deleted < recToDelete.Int64(); iterMeta.Next() {
			store.Delete(iterMeta.Key())
			deleted++
		}
	}

	// 4. 결과 확인
	if deleted < recToDelete.Int64() {
		return fmt.Errorf("[Remove] 담보 부족: 요청=%d, 실제삭제=%d", recToDelete.Int64(), deleted)
	}

	ctx.Logger().Info("[Remove] Collateral REC 삭제 완료",
		"from", from,
		"requested", recToDelete.String(),
		"deleted", deleted,
	)

	return nil
}
