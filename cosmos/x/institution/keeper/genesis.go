package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/institution/types"
)

func (k Keeper) InitGenesis(ctx sdk.Context, gs types.GenesisState) {
	for _, addr := range gs.WhitelistedAddrs {
		accAddr, err := sdk.AccAddressFromBech32(addr)
		if err != nil {
			panic(err)
		}
		k.AddToWhitelist(ctx, accAddr)
	}
}
