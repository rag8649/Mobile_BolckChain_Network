package types

import (
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

const (
	ModuleName = "reward"
	StoreKey   = ModuleName
	RouterKey  = ModuleName

	CollateralKey  = "collateralKey"
	SupplyKey      = "supply"
	RecRecordKey   = "rec_record" // 내부 RECRecord
	RecMetaKey     = "rec_meta"   // 외부 RECMeta
	TotalEnergyKey = "total_energy"
	EnergyTxKey    = "EnergyTx_Key"
	// 🔹 발전자별 누적량 prefix
	EnergyByAddressPrefix = "EnergyByAddress/"
)

var (
	ModuleAddress = authtypes.NewModuleAddress(ModuleName).String()
	ModulePerms   = []string{authtypes.Burner, authtypes.Minter}
)

const (
	TxListHeadKey = "TxListHead"
	TxListTailKey = "TxListTail"
	TxNodePrefix  = "TxNode/"
)
