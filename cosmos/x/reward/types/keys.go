package types

import (
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

const (
	ModuleName = "reward"
	StoreKey   = ModuleName
	RouterKey  = ModuleName

	CollateralKey = "collateralKey"
	SupplyKey     = "supply"
)

var (
	ModuleAddress = authtypes.NewModuleAddress(ModuleName).String()
	ModulePerms   = []string{authtypes.Burner, authtypes.Minter}
)
