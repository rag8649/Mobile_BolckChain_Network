package consumer

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"strconv"
)

type CollateralResponse struct {
	TotalAmount string `json:"total_amount"`
}

type SupplyResponse struct {
	Supply struct {
		Minted string `json:"minted"`
	} `json:"supply"`
}

// 담보/공급량 조회 후 청산 필요 여부 확인
func CheckLiquidation() {
	// 1. 담보 조회
	collateralResp, err := http.Get("http://localhost:1317/cosmos/reward/v1beta1/collateral")
	if err != nil {
		log.Printf("Collateral API 조회 실패: %v", err)
		return
	}
	defer collateralResp.Body.Close()

	collateralBody, _ := ioutil.ReadAll(collateralResp.Body)
	var collateral CollateralResponse
	if err := json.Unmarshal(collateralBody, &collateral); err != nil {
		log.Printf("Collateral JSON 파싱 실패: %v", err)
		return
	}

	totalCollateral, _ := strconv.ParseFloat(collateral.TotalAmount, 64)

	// 2. 공급량 조회
	supplyResp, err := http.Get("http://localhost:1317/cosmos/reward/v1beta1/supply")
	if err != nil {
		log.Printf("Supply API 조회 실패: %v", err)
		return
	}
	defer supplyResp.Body.Close()

	supplyBody, _ := ioutil.ReadAll(supplyResp.Body)
	var supply SupplyResponse
	if err := json.Unmarshal(supplyBody, &supply); err != nil {
		log.Printf("Supply JSON 파싱 실패: %v", err)
		return
	}

	totalSupply, _ := strconv.ParseFloat(supply.Supply.Minted, 64)

	const MinCollateralRatio = 1.0 // 담보비율

	if totalCollateral/MinCollateralRatio < totalSupply {
		log.Printf("UnderCollateral!!")
		go func() {
			cmd := exec.Command(
				"./build/simd", "tx", "reward", "burn-module-stable", "0",
				"--from", "alice",
				"--chain-id", "learning-chain-1",
				"--home", "./private/.simapp",
				"--keyring-backend", "test",
				"--yes",
				"--broadcast-mode", "block",
			)

			out, err := cmd.CombinedOutput()
			if err != nil {
				log.Printf("[Kafka: REC] burn-module-stable 실행 실패: %v\n출력: %s", err, string(out))
				return
			}
			log.Printf("[Kafka: REC] burn-module-stable 실행 완료\n%s", string(out))
		}()
	}
}
