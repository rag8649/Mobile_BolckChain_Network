package consumer

import (
	"github.com/cosmos/cosmos-sdk/simapp"
	rewardkeeper "github.com/cosmos/cosmos-sdk/x/reward/keeper"
)

var rewardKeeper rewardkeeper.Keeper
var myApp *simapp.SimApp

func SetKeeper(rk rewardkeeper.Keeper) {
	rewardKeeper = rk
}

func SetApp(a *simapp.SimApp) {
	myApp = a
}
