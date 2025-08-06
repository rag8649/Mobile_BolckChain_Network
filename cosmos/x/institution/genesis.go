package institution

import (
	"fmt"

	"encoding/base64"

	sdked25519 "github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
	data institutiontypes.GenesisState,
) {
	// Whitelist에 등록된 주소들을 순회
	for _, addrStr := range data.WhitelistedAddrs {
		addr, err := sdk.AccAddressFromBech32(addrStr)
		if err != nil {
			panic(fmt.Sprintf("invalid bech32 address: %v", err))
		}

		valAddr := sdk.ValAddress(addr)

		// consensus pubkey는 테스트용으로 적당한 값 사용 (실제 네트워크에선 노드 pubkey로 교체)
		pubKey := dummyPubKey() // 아래에 함수 정의됨

		validator, err := stakingtypes.NewValidator(valAddr, pubKey, stakingtypes.Description{
			Moniker: "AutoValidator_" + addr.String(),
		})
		if err != nil {
			panic(fmt.Sprintf("failed to create validator: %v", err))
		}

		validator.Tokens = sdk.NewInt(1000000)
		validator.DelegatorShares = sdk.OneDec()
		validator.Status = stakingtypes.Bonded

		stakingKeeper.SetValidator(ctx, validator)
		stakingKeeper.SetValidatorByConsAddr(ctx, validator)
		stakingKeeper.AfterValidatorCreated(ctx, valAddr)
	}
}

func ExportGenesis(ctx sdk.Context, k institutionkeeper.Keeper) *types.GenesisState {
	return &types.GenesisState{
		WhitelistedAddrs: k.GetAllWhitelisted(ctx),
	}
}

func dummyPubKey() cryptotypes.PubKey {
	keyBytes, _ := base64.StdEncoding.DecodeString("A3t6F7oyW9V8sxhj9F+JD7zy6x6rF9KmL9L9Om3qJQsW")
	var pk sdked25519.PubKey
	copy(pk.Key[:], keyBytes[:]) // Key는 [32]byte
	return &pk
}
