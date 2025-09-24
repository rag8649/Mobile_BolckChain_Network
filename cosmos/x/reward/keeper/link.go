package keeper

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/reward/types"
)

func (k Keeper) AppendTxNode(ctx sdk.Context, node types.TxNodeTx) error {

	store := ctx.KVStore(k.storeKey)

	// 1. 지금까지 KVStore에 저장된 모든 TxHashes 조회
	allHashes := k.GetAllTxHashes(ctx)
	node.TxHashes = allHashes // 새 노드에 기록

	// 2. 새 노드 직렬화 및 저장
	bz, err := json.Marshal(node)
	if err != nil {
		return err
	}
	nodeKey := []byte(fmt.Sprintf("%s%s", types.TxNodePrefix, node.RecId))
	store.Set(nodeKey, bz)

	// 3. head가 비어 있으면 → 첫 노드
	head := store.Get([]byte(types.TxListHeadKey))
	if head == nil {
		store.Set([]byte(types.TxListHeadKey), []byte(node.RecId))
		store.Set([]byte(types.TxListTailKey), []byte(node.RecId))
		// KVStore 초기화
		k.resetEnergyStore(ctx, store)
		return nil
	}

	// 4. 기존 tail 불러오기
	tail := string(store.Get([]byte(types.TxListTailKey)))
	tailKey := []byte(fmt.Sprintf("%s%s", types.TxNodePrefix, tail))
	tailNodeBz := store.Get(tailKey)
	if tailNodeBz == nil {
		return fmt.Errorf("tail node not found")
	}

	var tailNode types.TxNodeTx
	if err := json.Unmarshal(tailNodeBz, &tailNode); err != nil {
		return err
	}
	tailNode.Next = node.RecId

	// 5. tail 갱신
	newTailBz, _ := json.Marshal(tailNode)
	store.Set(tailKey, newTailBz)
	store.Set([]byte(types.TxListTailKey), []byte(node.RecId))

	// 6. KVStore 초기화
	k.resetEnergyStore(ctx, store)

	return nil
}

func (k Keeper) resetEnergyStore(ctx sdk.Context, store sdk.KVStore) {
	// 발전자별 누적량 초기화
	iterator := sdk.KVStorePrefixIterator(store, []byte(types.EnergyByAddressPrefix))
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		store.Delete(iterator.Key())
	}
	// 전체 Tx 해시 리스트 초기화
	store.Delete([]byte(types.EnergyTxKey))

	// ctx.Logger().Info("[TxList] 에너지 KVStore 초기화 완료")
}

// 모든 TxHash 조회 (EnergyTxKey 기준)
func (k Keeper) GetAllTxHashes(ctx sdk.Context) []string {
	store := ctx.KVStore(k.storeKey)

	bz := store.Get([]byte(types.EnergyTxKey))
	if bz == nil {
		ctx.Logger().Info("[DEBUG] GetAllTxHashes: KVStore 비어있음")
		return []string{}
	}

	var txList []string
	if err := json.Unmarshal(bz, &txList); err != nil {
		ctx.Logger().Error("[DEBUG] GetAllTxHashes: JSON 언마샬 실패", "err", err)
		return []string{}
	}

	// ctx.Logger().Info("[DEBUG] GetAllTxHashes: 조회 결과", "txs", txList)
	return txList
}
