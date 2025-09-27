package tx

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	rtypes "github.com/cosmos/cosmos-sdk/x/reward/types"
)

type TxResponse struct {
	Height    string          `json:"height"`
	TxHash    string          `json:"txhash"`
	RawLog    string          `json:"raw_log"`
	Logs      json.RawMessage `json:"logs"`
	Code      int             `json:"code"`
	Codespace string          `json:"codespace"`
	Data      string          `json:"data"`
	GasUsed   string          `json:"gas_used"`
	GasWanted string          `json:"gas_wanted"`
}

type BurnResultMessage struct {
	Address     string              `json:"address"`
	Status      string              `json:"status"`
	TxHash      string              `json:"tx_hash"`
	RecRecords  []*rtypes.RECRecord `json:"rec_records"`
	RecMetas    []*rtypes.RECMeta   `json:"rec_metas"`
	ErrorReason string              `json:"error_reason,omitempty"`
}

func BurnStableCoin(targetAddr, amount string) (*BurnResultMessage, error) {
	cmd := exec.Command("./build/simd", "tx", "reward", "burn-stable-coin",
		targetAddr, amount,
		"--from", "alice",
		"--chain-id", "learning-chain-1",
		"--keyring-backend", "test",
		"--home", "private/.simapp",
		"--gas", "auto",
		"--gas-adjustment", "1.2",
		"--yes",
		"--broadcast-mode", "block",
		"-o", "json",
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("simd error: %v\noutput: %s", err, string(out))
	}

	fmt.Printf("[Kafka: Burn] 전체 결과: %s\n", string(out))

	// === JSON 부분만 추출 ===
	raw := string(out)
	start := strings.Index(raw, "{")
	if start == -1 {
		return nil, fmt.Errorf("no JSON found in output: %s", raw)
	}
	jsonPart := raw[start:]

	// === TxResponse 파싱 ===
	var txResp TxResponse
	if err := json.Unmarshal([]byte(jsonPart), &txResp); err != nil {
		return nil, fmt.Errorf("failed to parse tx response: %v\njson: %s", err, jsonPart)
	}

	// === raw_log 파싱 (이벤트 안의 REC 데이터 추출) ===
	var parsedLogs []map[string]interface{}
	if err := json.Unmarshal([]byte(txResp.RawLog), &parsedLogs); err != nil {
		return nil, fmt.Errorf("failed to parse raw_log: %v\nraw_log: %s", err, txResp.RawLog)
	}

	var recs []*rtypes.RECRecord
	var metas []*rtypes.RECMeta

	for _, logEntry := range parsedLogs {
		if events, ok := logEntry["events"].([]interface{}); ok {
			for _, ev := range events {
				event := ev.(map[string]interface{})
				switch event["type"] {
				case "rec_record_returned":
					if attrs, ok := event["attributes"].([]interface{}); ok {
						for _, attr := range attrs {
							kv := attr.(map[string]interface{})
							if kv["key"] == "data" {
								raw := kv["value"].(string)
								var rec rtypes.RECRecord
								if err := json.Unmarshal([]byte(raw), &rec); err == nil {
									recs = append(recs, &rec)
								} else {
									fmt.Println("[Kafka: Burn] RECRecord 파싱 실패:", err, "raw=", raw)
								}
							}
						}
					}
				case "rec_meta_returned":
					if attrs, ok := event["attributes"].([]interface{}); ok {
						for _, attr := range attrs {
							kv := attr.(map[string]interface{})
							if kv["key"] == "data" {
								raw := kv["value"].(string)
								var meta rtypes.RECMeta
								if err := json.Unmarshal([]byte(raw), &meta); err == nil {
									metas = append(metas, &meta)
								} else {
									fmt.Println("[Kafka: Burn] RECMeta 파싱 실패:", err, "raw=", raw)
								}
							}
						}
					}
				}
			}
		}
	}

	// === 최종 BurnResultMessage 구성 ===
	return &BurnResultMessage{
		Address:    targetAddr,
		Status:     "success",
		TxHash:     txResp.TxHash,
		RecRecords: recs,
		RecMetas:   metas,
	}, nil
}
