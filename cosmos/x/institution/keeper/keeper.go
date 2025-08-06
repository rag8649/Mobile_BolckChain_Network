package keeper

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	"github.com/cosmos/cosmos-sdk/x/institution/types"
)

type Keeper struct {
	cdc      codec.Codec
	storeKey sdk.StoreKey
	ak       authkeeper.AccountKeeper
}

func NewKeeper(cdc codec.Codec, storeKey sdk.StoreKey, ak authkeeper.AccountKeeper) Keeper {
	return Keeper{
		cdc:      cdc,
		storeKey: storeKey,
		ak:       ak,
	}
}

// ✅ whitelist 등록
func (k Keeper) SetWhitelisted(ctx sdk.Context, addr sdk.AccAddress) {
	store := ctx.KVStore(k.storeKey)
	store.Set([]byte(types.WhitelistedPrefix+addr.String()), []byte{1})
}

// ✅ whitelist 여부 확인
func (k Keeper) IsWhitelisted(ctx sdk.Context, addr sdk.AccAddress) bool {
	store := ctx.KVStore(k.storeKey)
	return store.Has([]byte(types.WhitelistedPrefix + addr.String()))
}

// ✅ 전체 whitelist 조회 (ExportGenesis용)
func (k Keeper) GetAllWhitelisted(ctx sdk.Context) []string {
	store := ctx.KVStore(k.storeKey)

	iterator := sdk.KVStorePrefixIterator(store, []byte(types.WhitelistedPrefix))
	defer iterator.Close()

	var addresses []string
	for ; iterator.Valid(); iterator.Next() {
		key := iterator.Key()
		addr := string(key[len(types.WhitelistedPrefix):])
		addresses = append(addresses, addr)
	}
	return addresses
}

// ✅ InitGenesis 구현

// ✅ ExportGenesis 구현
func (k Keeper) ExportGenesis(ctx sdk.Context) types.GenesisState {
	addrs := k.GetAllWhitelisted(ctx)
	return types.GenesisState{
		WhitelistedAddrs: addrs,
	}
}

func (k Keeper) AddToWhitelist(ctx sdk.Context, addr sdk.AccAddress) {
	store := ctx.KVStore(k.storeKey)
	store.Set([]byte("whitelist_"+addr.String()), []byte{1})
}
