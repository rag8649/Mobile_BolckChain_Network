package tx

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	rtypes "github.com/cosmos/cosmos-sdk/x/reward/types"
	"github.com/gogo/protobuf/proto"
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

// func BurnStableCoin(targetAddr, amount string) (*BurnResultMessage, error) {
// 	cmd := exec.Command("./build/simd", "tx", "reward", "burn-stable-coin",
// 		targetAddr, amount,
// 		"--from", "alice",
// 		"--chain-id", "learning-chain-1",
// 		"--keyring-backend", "test",
// 		"--home", "private/.simapp",
// 		"--gas", "auto",
// 		"--gas-adjustment", "1.2",
// 		"--yes",
// 		"--broadcast-mode", "block",
// 		"-o", "json",
// 	)

// 	out, err := cmd.CombinedOutput()
// 	if err != nil {
// 		return nil, fmt.Errorf("[Kafka: Burn] simd error: %v\noutput: %s", err, string(out))
// 	}

// 	// === JSON 부분만 추출 ===
// 	raw := string(out)
// 	start := strings.Index(raw, "{")
// 	if start == -1 {
// 		return nil, fmt.Errorf("[Kafka: Burn] no JSON found in output: %s", raw)
// 	}
// 	jsonPart := raw[start:]

// 	// === TxResponse 파싱 ===
// 	var txResp TxResponse
// 	if err := json.Unmarshal([]byte(jsonPart), &txResp); err != nil {
// 		return nil, fmt.Errorf("[Kafka: Burn] failed to parse tx response: %v\njson: %s", err, jsonPart)
// 	}

// 	// === raw_log 파싱 (이벤트 안의 REC 데이터 추출) ===
// 	var parsedLogs []map[string]interface{}
// 	if err := json.Unmarshal([]byte(txResp.RawLog), &parsedLogs); err != nil {
// 		return nil, fmt.Errorf("[Kafka: Burn] failed to parse raw_log: %v\nraw_log: %s", err, txResp.RawLog)
// 	}

// 	var recs []*rtypes.RECRecord
// 	var metas []*rtypes.RECMeta

// 	for _, logEntry := range parsedLogs {
// 		if events, ok := logEntry["events"].([]interface{}); ok {
// 			for _, ev := range events {
// 				event := ev.(map[string]interface{})
// 				switch event["type"] {
// 				case "rec_record_returned":
// 					if attrs, ok := event["attributes"].([]interface{}); ok {
// 						for _, attr := range attrs {
// 							kv := attr.(map[string]interface{})
// 							if kv["key"] == "data" {
// 								raw := kv["value"].(string)
// 								var rec rtypes.RECRecord
// 								if err := json.Unmarshal([]byte(raw), &rec); err == nil {
// 									recs = append(recs, &rec)
// 								} else {
// 									fmt.Println("[Kafka: Burn] RECRecord 파싱 실패:", err, "raw=", raw)
// 								}
// 							}
// 						}
// 					}
// 				case "rec_meta_returned":
// 					if attrs, ok := event["attributes"].([]interface{}); ok {
// 						for _, attr := range attrs {
// 							kv := attr.(map[string]interface{})
// 							if kv["key"] == "data" {
// 								raw := kv["value"].(string)
// 								var meta rtypes.RECMeta
// 								if err := json.Unmarshal([]byte(raw), &meta); err == nil {
// 									metas = append(metas, &meta)
// 								} else {
// 									fmt.Println("[Kafka: Burn] RECMeta 파싱 실패:", err, "raw=", raw)
// 								}
// 							}
// 						}
// 					}
// 				}
// 			}
// 		}
// 	}

// 	// === 최종 BurnResultMessage 구성 ===
// 	return &BurnResultMessage{
// 		Address:    targetAddr,
// 		Status:     "success",
// 		TxHash:     txResp.TxHash,
// 		RecRecords: recs,
// 		RecMetas:   metas,
// 	}, nil
// }

func BurnStableCoinCLI(clientCtx client.Context, targetAddr, amount string) (*BurnResultMessage, error) {
	cliLock.Lock()
	defer cliLock.Unlock()

	if !strings.Contains(amount, "aeil") {
		amount = fmt.Sprintf("%saeil", amount)
	}

	msg := &rtypes.MsgBurnStableCoin{
		Creator:    clientCtx.GetFromAddress().String(),
		TargetAddr: targetAddr,
		Amount:     amount,
	}

	txBuilder := clientCtx.TxConfig.NewTxBuilder()
	if err := txBuilder.SetMsgs(msg); err != nil {
		return nil, fmt.Errorf("[BurnStableCoinCLI] SetMsgs 실패: %w", err)
	}

	fromAddr := clientCtx.GetFromAddress()
	if !cacheInitialized {
		accNum, seq, err := clientCtx.AccountRetriever.GetAccountNumberSequence(clientCtx, fromAddr)
		if err != nil {
			return nil, fmt.Errorf("[BurnStableCoinCLI] AccountRetriever 실패: %w", err)
		}
		accNumCache = accNum
		seqCache = seq
		cacheInitialized = true
	} else {
		seqCache++
	}

	txf := tx.Factory{}.
		WithChainID(clientCtx.ChainID).
		WithTxConfig(clientCtx.TxConfig).
		WithAccountRetriever(clientCtx.AccountRetriever).
		WithKeybase(clientCtx.Keyring).
		WithAccountNumber(accNumCache).
		WithSequence(seqCache).
		WithGasAdjustment(1.2).
		WithGasPrices("0.025stake").
		WithMemo("burn-stable-coin")

	_, gasWanted, err := tx.CalculateGas(clientCtx, txf, msg)
	if err != nil {
		seqCache = forceSyncAccountSequence(clientCtx)
		return nil, fmt.Errorf("[BurnStableCoinCLI] 가스 계산 실패: %w", err)
	}

	gasLimit := uint64(float64(gasWanted)*1.2) + 10000
	txBuilder.SetGasLimit(gasLimit)

	gasPrices, _ := sdk.ParseDecCoins("0.025stake")
	decGas := sdk.NewDec(int64(gasLimit))
	decFees := gasPrices.MulDec(decGas)
	fees, _ := decFees.TruncateDecimal()
	if fees.IsZero() {
		fees = sdk.NewCoins(sdk.NewInt64Coin("stake", 1))
	}
	txBuilder.SetFeeAmount(fees)

	if err := tx.Sign(txf, clientCtx.FromName, txBuilder, true); err != nil {
		seqCache = forceSyncAccountSequence(clientCtx)
		waitForNextBlock(clientCtx)
		return nil, fmt.Errorf("[BurnStableCoinCLI] 서명 실패: %w", err)
	}

	txBytes, err := clientCtx.TxConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		seqCache = forceSyncAccountSequence(clientCtx)
		return nil, fmt.Errorf("[BurnStableCoinCLI] 인코딩 실패: %w", err)
	}

	res, err := clientCtx.BroadcastTxCommit(txBytes)
	if err != nil {
		seqCache = forceSyncAccountSequence(clientCtx)
		waitForNextBlock(clientCtx)
		return nil, fmt.Errorf("[BurnStableCoinCLI] 브로드캐스트 실패: %w", err)
	}

	if res.Code != 0 {
		seqCache = forceSyncAccountSequence(clientCtx)
		waitForNextBlock(clientCtx)

		return &BurnResultMessage{
			Address:     targetAddr,
			Status:      "failed",
			TxHash:      res.TxHash,
			ErrorReason: fmt.Sprintf("DeliverTx 실패: code=%d log=%s", res.Code, res.RawLog),
		}, fmt.Errorf("[BurnStableCoinCLI] DeliverTx 실패: code=%d log=%s", res.Code, res.RawLog)
	}

	fmt.Printf("\033[32m[BurnStableCoinCLI] 성공 (seq=%d)\033[0m\n", seqCache)

	var resp rtypes.MsgBurnStableCoinResponse
	if res.Data != "" {

		// ✅ 1️⃣ HEX → bytes 변환
		dataBytes, err := hex.DecodeString(res.Data)
		if err != nil {
			fmt.Printf("[BurnStableCoinCLI] HEX 디코딩 실패: %v\n", err)
		} else {
			// ✅ 2️⃣ TxMsgData로 먼저 풀기
			var msgData sdk.TxMsgData
			if err := proto.Unmarshal(dataBytes, &msgData); err != nil {
				fmt.Printf("[BurnStableCoinCLI] TxMsgData Unmarshal 실패: %v\n", err)
			} else {
				if len(msgData.Data) > 0 {
					var result rtypes.MsgBurnStableCoinResponse
					if err := proto.Unmarshal(msgData.Data[0].Data, &result); err != nil {
						fmt.Printf("[BurnStableCoinCLI] MsgBurnStableCoinResponse Unmarshal 실패: %v\n", err)
					} else {
						resp = result
						fmt.Printf("[BurnStableCoinCLI] RECRecord=%d, RECMeta=%d\n",
							len(resp.RecRecords), len(resp.RecMetas))
					}
				}
			}
		}
	}

	// === 최종 BurnResultMessage 구성 ===
	return &BurnResultMessage{
		Address:    targetAddr,
		Status:     "success",
		TxHash:     res.TxHash,
		RecRecords: resp.RecRecords,
		RecMetas:   resp.RecMetas,
	}, nil
}
