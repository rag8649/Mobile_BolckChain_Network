package keeper

import (
	"context"
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/reward/types"
)

var _ types.QueryServer = Querier{}

type Querier struct {
	Keeper
}

// Collateral 쿼리
func (q Querier) Collateral(ctx context.Context, req *types.QueryCollateralRequest) (*types.QueryCollateralResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// 단일 총 담보량 가져오기
	collateral, err := q.GetTotalCollateral(sdkCtx)
	if err != nil {

	}

	return &types.QueryCollateralResponse{
		TotalAmount: collateral.String(),
	}, nil
}

// Supply 쿼리
func (q Querier) Supply(ctx context.Context, req *types.QuerySupplyRequest) (*types.QuerySupplyResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	supply := q.GetSupply(sdkCtx) // ← GetSupply 함수 필요
	return &types.QuerySupplyResponse{Supply: supply}, nil
}

// RECList : 담보로 등록된 모든 REC 조회
func (q Querier) RECList(ctx context.Context, req *types.QueryRECListRequest) (*types.QueryRECListResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// 모든 RECRecord 조회
	recRecords, err := q.GetAllRECRecords(sdkCtx)
	if err != nil {
		return nil, err
	}

	// 모든 RECMeta 조회
	recMetas, err := q.GetAllRECMetas(sdkCtx)
	if err != nil {
		return nil, err
	}

	return &types.QueryRECListResponse{
		RecRecords: recRecords,
		RecMetas:   recMetas,
	}, nil
}

// TxNodeList : LinkedList 전체 조회
func (q Querier) TxNodeList(ctx context.Context, req *types.QueryTxNodeListRequest) (*types.QueryTxNodeListResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(q.storeKey)

	head := store.Get([]byte(types.TxListHeadKey))
	if head == nil {
		// 빈 리스트면 빈 응답 반환
		return &types.QueryTxNodeListResponse{Nodes: []*types.TxNodeTx{}}, nil
	}

	var nodes []*types.TxNodeTx
	cur := string(head)

	for cur != "" {
		nodeKey := []byte(fmt.Sprintf("%s%s", types.TxNodePrefix, cur))
		bz := store.Get(nodeKey)
		if bz == nil {
			break
		}

		var node types.TxNodeTx
		if err := json.Unmarshal(bz, &node); err != nil {
			return nil, err
		}

		nodes = append(nodes, &node)
		cur = node.Next
	}

	return &types.QueryTxNodeListResponse{Nodes: nodes}, nil
}
