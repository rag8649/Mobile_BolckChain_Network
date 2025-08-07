package institution

import (
	"fmt"

	cryptosecp256k1 "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	distributionkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	institutionkeeper "github.com/cosmos/cosmos-sdk/x/institution/keeper"
	"github.com/cosmos/cosmos-sdk/x/institution/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func InitGenesis(
	ctx sdk.Context,
	k institutionkeeper.Keeper,
	stakingKeeper stakingkeeper.Keeper,
	bankKeeper bankkeeper.Keeper,
	distrKeeper distributionkeeper.Keeper,
	data types.GenesisState,
) {
	for _, addrStr := range data.WhitelistedAddrs {
		accAddr, err := sdk.AccAddressFromBech32(addrStr)
		if err != nil {
			panic(fmt.Sprintf("Invalid address: %v", err))
		}
		valAddr := sdk.ValAddress(accAddr)

		// 1. 화이트리스트 등록
		k.AddToWhitelist(ctx, accAddr)

		// 2. Validator 생성
		pubKey := dummyPubKey()
		description := stakingtypes.NewDescription("AutoValidator_"+accAddr.String(), "", "", "", "")
		validator, err := stakingtypes.NewValidator(valAddr, pubKey, description)
		if err != nil {
			panic(fmt.Sprintf("Failed to create validator: %v", err))
		}
		validator.Status = stakingtypes.Bonded
		validator.Tokens = sdk.NewInt(1000000000) // 1000stake
		validator.DelegatorShares = sdk.NewDec(1000000000)

		stakingKeeper.SetValidator(ctx, validator)
		stakingKeeper.SetValidatorByPowerIndex(ctx, validator)
		stakingKeeper.SetValidatorByConsAddr(ctx, validator)

		// 3. Genesis에서는 Hook이 자동 호출되지 않으므로 distribution state 초기화
		distrKeeper.SetValidatorOutstandingRewards(ctx, valAddr, distrtypes.ValidatorOutstandingRewards{Rewards: sdk.NewDecCoins()})
		distrKeeper.SetValidatorAccumulatedCommission(ctx, valAddr, distrtypes.ValidatorAccumulatedCommission{Commission: sdk.NewDecCoins()})
		distrKeeper.SetValidatorHistoricalRewards(ctx, valAddr, 0, distrtypes.NewValidatorHistoricalRewards(sdk.NewDecCoins(), 1))
		distrKeeper.SetValidatorCurrentRewards(ctx, valAddr, distrtypes.NewValidatorCurrentRewards(sdk.NewDecCoins(), 1))

		// 4. Token mint → account → staking
		amount := sdk.NewInt64Coin("stake", 1000000000)
		coins := sdk.NewCoins(amount)

		if err := bankKeeper.MintCoins(ctx, minttypes.ModuleName, coins); err != nil {
			panic(err)
		}
		if err := bankKeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, accAddr, coins); err != nil {
			panic(err)
		}

		// 5. 위임 (self-delegation)
		_, err = stakingKeeper.Delegate(
			ctx,
			accAddr,
			amount.Amount,
			stakingtypes.Bonded,
			validator,
			true,
		)
		if err != nil {
			panic(fmt.Sprintf("delegate error: %v", err))
		}
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
