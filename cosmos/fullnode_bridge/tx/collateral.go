package tx

import (
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/cosmos/cosmos-sdk/fullnode_bridge/types"
)

func BurnStableCoin(targetAddr, amount string) (*types.BurnResultMessage, error) {
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
		"--output", "json",
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("simd error: %v\noutput: %s", err, string(out))
	}

	// 1. CLI 출력(JSON)을 BurnResultMessage로 파싱
	var resp types.BurnResultMessage
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse burn response: %v\noutput: %s", err, string(out))
	}

	// 2. 상태 성공 표시
	resp.Status = "success"
	return &resp, nil
}
