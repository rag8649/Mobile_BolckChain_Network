package types

import (
	fmt "fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		WhitelistedAddrs: []string{},
	}
}

func (gs *GenesisState) Validate() error {
	for _, addr := range gs.WhitelistedAddrs {
		if _, err := sdk.AccAddressFromBech32(addr); err != nil {
			return fmt.Errorf("invalid bech32 address in whitelist: %s", addr)
		}
	}
	return nil
}

func DefaultGenesis() *GenesisState {
	return &GenesisState{
		WhitelistedAddrs: []string{},
	}
}
