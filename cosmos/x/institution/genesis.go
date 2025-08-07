package institution

import (
	"fmt"

	cryptosecp256k1 "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	institutionkeeper "github.com/cosmos/cosmos-sdk/x/institution/keeper"
	"github.com/cosmos/cosmos-sdk/x/institution/types"
	institutiontypes "github.com/cosmos/cosmos-sdk/x/institution/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func InitGenesis(
	ctx sdk.Context,
	k institutionkeeper.Keeper,
	stakingKeeper stakingkeeper.Keeper,
	bankKeeper bankkeeper.Keeper,
	data institutiontypes.GenesisState,
) {
	stakeAmt := sdk.NewInt(1000000000) // 1,000 STAKE

	for _, addrStr := range data.WhitelistedAddrs {
		addr, err := sdk.AccAddressFromBech32(addrStr)
		if err != nil {
			panic(fmt.Sprintf("❌ invalid bech32 address: %v", err))
		}
		valAddr := sdk.ValAddress(addr)

		pubKey := dummyPubKey() // 임시 테스트용

		validator, err := stakingtypes.NewValidator(valAddr, pubKey, stakingtypes.Description{
			Moniker: "AutoValidator_" + addr.String(),
		})
		if err != nil {
			panic(fmt.Sprintf("❌ validator 생성 실패: %v", err))
		}

		validator.Status = stakingtypes.Bonded
		validator.Tokens = stakeAmt
		validator.DelegatorShares = sdk.NewDecFromInt(stakeAmt)

		stakingKeeper.SetValidator(ctx, validator)
		stakingKeeper.SetValidatorByConsAddr(ctx, validator)
		stakingKeeper.AfterValidatorCreated(ctx, valAddr)

		bondedPool := stakingKeeper.GetBondedPool(ctx)
		coin := sdk.NewCoin("stake", stakeAmt)

		if err := bankKeeper.MintCoins(ctx, stakingtypes.ModuleName, sdk.NewCoins(coin)); err != nil {
			panic(fmt.Sprintf("❌ mint error: %v", err))
		}
		if err := bankKeeper.SendCoinsFromModuleToModule(ctx,
			stakingtypes.ModuleName,
			bondedPool.GetName(),
			sdk.NewCoins(coin),
		); err != nil {
			panic(fmt.Sprintf("❌ send to bonded pool error: %v", err))
		}

		ctx.Logger().Info("✅ AutoValidator 생성 시도", "address", addr.String())
	}
}

func ExportGenesis(ctx sdk.Context, k institutionkeeper.Keeper) *types.GenesisState {
	return &types.GenesisState{
		WhitelistedAddrs: k.GetAllWhitelisted(ctx),
	}
}

func dummyPubKey() cryptotypes.PubKey {
	privKey := cryptosecp256k1.GenPrivKey()
	return privKey.PubKey()
}
