package tx

import (
	"os/exec"
)

func DepositCollateral(amount string) (string, error) {
	cmd := exec.Command(
		"./build/simd", "tx", "reward", "deposit-collateral", amount,
		"--from", "alice",
		"--chain-id", "learning-chain-1",
		"--home", "./private/.simapp",
		"--keyring-backend", "test",
		"--gas", "auto",
		"--gas-adjustment", "1.2",
		"--yes",
	)

	out, err := cmd.CombinedOutput()
	return string(out), err
}

func BurnStableCoin(targetAddr, amount string) (string, error) {
	cmd := exec.Command(
		"./build/simd", "tx", "reward", "burn-stable-coin",
		targetAddr, amount,
		"--from", "alice",
		"--chain-id", "learning-chain-1",
		"--home", "./private/.simapp",
		"--keyring-backend", "test",
		"--gas", "auto",
		"--gas-adjustment", "1.2",
		"--yes",
	)

	out, err := cmd.CombinedOutput()
	return string(out), err
}
