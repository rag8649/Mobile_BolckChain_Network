package institution

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	institutionkeeper "github.com/cosmos/cosmos-sdk/x/institution/keeper"
	"github.com/cosmos/cosmos-sdk/x/institution/types"
)

func InitGenesis(ctx sdk.Context, k institutionkeeper.Keeper, data types.GenesisState) {
	for _, addr := range data.WhitelistedAddrs {
		k.AddToWhitelist(ctx, sdk.AccAddress(addr)) // Keeper 메서드
	}
}

func ExportGenesis(ctx sdk.Context, k institutionkeeper.Keeper) *types.GenesisState {
	return &types.GenesisState{
		WhitelistedAddrs: k.GetAllWhitelisted(ctx),
	}
}
