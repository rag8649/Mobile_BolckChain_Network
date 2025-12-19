package tx

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os/exec"
	"strconv"
	"time"

	"github.com/IBM/sarama"
	"github.com/cosmos/cosmos-sdk/fullnode_bridge/config"
	"github.com/cosmos/cosmos-sdk/fullnode_bridge/producer"
	"github.com/cosmos/cosmos-sdk/fullnode_bridge/types"

	"sync"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	lighttxtypes "github.com/cosmos/cosmos-sdk/x/light_tx/types"
	rewardtypes "github.com/cosmos/cosmos-sdk/x/reward/types"
	grpc "google.golang.org/grpc"
)

var (
	cliLock sync.Mutex // ✅ 전역에서 한 번 선언

	accNumCache      uint64
	seqCache         uint64
	cacheInitialized bool
)

// BroadcastLightTxWithReward : LightTx + Reward 트랜잭션 전송
func BroadcastLightTxWithReward(
	clientCtx client.Context,
	grpcConn *grpc.ClientConn,
	msg types.LightTxMessage,
	addr string,
	power float64,
) (string, error) {

	// 🔒 트랜잭션 전송 동시 접근 방지
	cliLock.Lock()
	defer cliLock.Unlock()

	// --- ✅ 기본 검증 ---
	if clientCtx.TxConfig == nil {
		return "", fmt.Errorf("[BLT] clientCtx.TxConfig is nil")
	}
	if clientCtx.Keyring == nil {
		return "", fmt.Errorf("[BLT] clientCtx.Keyring is nil")
	}
	if clientCtx.AccountRetriever == nil {
		clientCtx = clientCtx.WithAccountRetriever(authtypes.AccountRetriever{})
	}
	if clientCtx.FromName == "" || clientCtx.FromAddress.Empty() {
		return "", fmt.Errorf("[BLT] clientCtx.FromName/FromAddress not set")
	}
	if power <= 0 {
		return "", fmt.Errorf("[BLT] 보상할 발전량이 없습니다")
	}

	signer := clientCtx.GetFromAddress().String()
	fromAddr := clientCtx.GetFromAddress()
	fromName := clientCtx.FromName

	// ✅ 최초 1회만 시퀀스/계정번호 조회
	if !cacheInitialized {
		accNum, seq, err := clientCtx.AccountRetriever.GetAccountNumberSequence(clientCtx, fromAddr)
		if err != nil {
			return "", fmt.Errorf("[BLT] 초기 AccountRetriever 실패: %w", err)
		}
		accNumCache = accNum
		seqCache = seq
		cacheInitialized = true
	} else {
		seqCache++ // ✅ 이전 트랜잭션 성공 가정 후 시퀀스 수동 증가
	}

	// --- LightTx 메시지 구성 ---
	var lightMsg sdk.Msg
	if msg.Original != nil {
		lightMsg = &lighttxtypes.MsgSendLightTx{
			Creator: signer,
			Payload: &lighttxtypes.MsgSendLightTx_Original{
				Original: &lighttxtypes.SolarData{
					DeviceId:    msg.Original.DeviceID,
					Timestamp:   msg.Original.Timestamp,
					TotalEnergy: msg.Original.TotalEnergy,
					Location: &lighttxtypes.Location{
						Latitude:  msg.Original.Location.Latitude,
						Longitude: msg.Original.Location.Longitude,
					},
				},
			},
			Hash:      msg.Hash,
			Signature: msg.Signature,
			Pubkey:    msg.Pubkey,
		}
	} else if msg.REC != nil {
		lightMsg = &lighttxtypes.MsgSendLightTx{
			Creator: signer,
			Payload: &lighttxtypes.MsgSendLightTx_Rec{
				Rec: &lighttxtypes.RECMeta{
					FacilityId:          msg.REC.FacilityID,
					FacilityName:        msg.REC.FacilityName,
					Location:            msg.REC.Location,
					TechnologyType:      msg.REC.TechnologyType,
					CapacityMw:          msg.REC.CapacityMW,
					RegistrationDate:    msg.REC.RegistrationDate,
					CertifiedId:         msg.REC.CertifiedId,
					IssueDate:           msg.REC.IssueDate,
					GenerationStartDate: msg.REC.GenerationStartDate,
					GenerationEndDate:   msg.REC.GenerationEndDate,
					MeasuredVolume_MWh:  msg.REC.MeasuredVolumeMWh,
					RetiredDate:         msg.REC.RetiredDate,
					RetirementPurpose:   msg.REC.RetirementPurpose,
					Status:              msg.REC.Status,
					Timestamp:           msg.REC.Timestamp,
				},
			},
			Hash:      msg.Hash,
			Signature: msg.Signature,
			Pubkey:    msg.Pubkey,
		}
	} else {
		return "", fmt.Errorf("[BLT] no valid LightTx data")
	}

	// --- Reward 메시지 ---
	rewardMsg := &rewardtypes.MsgRewardSolarPower{
		Creator: signer,
		Address: addr,
		Amount:  strconv.FormatInt(int64(math.Round(power)), 10),
	}

	// --- TxBuilder 구성 ---
	txBuilder := clientCtx.TxConfig.NewTxBuilder()
	if err := txBuilder.SetMsgs(lightMsg, rewardMsg); err != nil {
		seqCache = forceSyncAccountSequence(clientCtx)

		return "", fmt.Errorf("[BLT] failed to set msgs: %w", err)
	}

	// --- Factory 생성 ---
	txf := tx.Factory{}.
		WithChainID(clientCtx.ChainID).
		WithTxConfig(clientCtx.TxConfig).
		WithAccountRetriever(clientCtx.AccountRetriever).
		WithGasAdjustment(1.2).
		WithGasPrices("0.025stake").
		WithMemo("light-tx").
		WithKeybase(clientCtx.Keyring).
		WithAccountNumber(accNumCache).
		WithSequence(seqCache)

	// --- 가스 시뮬레이션 ---
	_, gasWanted, err := tx.CalculateGas(clientCtx, txf, lightMsg, rewardMsg)
	if err != nil {
		seqCache = forceSyncAccountSequence(clientCtx)
		return "", fmt.Errorf("[BLT] failed to simulate gas: %w", err)
	}

	adjustedGas := uint64(float64(gasWanted)*1.3) + 10000
	txBuilder.SetGasLimit(adjustedGas)

	// --- 수수료 계산 ---
	gasPrices, _ := sdk.ParseDecCoins("0.025stake")
	decGas := sdk.NewDec(int64(adjustedGas))
	decFees := gasPrices.MulDec(decGas)
	fees, _ := decFees.TruncateDecimal()
	if fees.IsZero() {
		fees = sdk.NewCoins(sdk.NewInt64Coin("stake", 1))
	}
	txBuilder.SetFeeAmount(fees)

	// --- 서명 ---
	if err := tx.Sign(txf, fromName, txBuilder, true); err != nil {
		seqCache = forceSyncAccountSequence(clientCtx)
		return "", fmt.Errorf("[BLT] failed to sign tx: %w", err)
	}

	// --- 브로드캐스트 ---
	txBytes, err := clientCtx.TxConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		seqCache = forceSyncAccountSequence(clientCtx)
		return "", fmt.Errorf("[BLT] failed to encode tx: %w", err)
	}

	res, err := clientCtx.BroadcastTxCommit(txBytes)
	if err != nil {
		seqCache = forceSyncAccountSequence(clientCtx)
		return "", fmt.Errorf("[BLT] broadcast failed: %w", err)
	}

	// --- 실패 처리 ---
	if res.Code != 0 {
		seqCache = forceSyncAccountSequence(clientCtx)
		return res.TxHash, fmt.Errorf("[BLT] deliverTx failed: code=%d raw_log=%s", res.Code, res.RawLog)
	}

	waitForNextBlock(clientCtx)

	fmt.Printf("\033[32m[BLT] 성공 (accNum=%d seq=%d)\033[0m\n", accNumCache, seqCache)
	return res.TxHash, nil
}

func forceSyncAccountSequence(clientCtx client.Context) uint64 {
	fromAddr := clientCtx.GetFromAddress()
	accNum, seq, err := clientCtx.AccountRetriever.GetAccountNumberSequence(clientCtx, fromAddr)
	if err != nil {
		fmt.Println("Sequence 재동기화 실패:", err)
		return seqCache // 이전 값 유지
	}
	accNumCache = accNum
	seqCache = seq
	fmt.Println("Sequence 재동기화 완료:", seq)
	seq--
	return seq
}

func waitForNextBlock(clientCtx client.Context) {
	status, _ := clientCtx.Client.Status(context.Background())
	curHeight := status.SyncInfo.LatestBlockHeight

	for {
		time.Sleep(500 * time.Millisecond)
		newStatus, _ := clientCtx.Client.Status(context.Background())
		if newStatus.SyncInfo.LatestBlockHeight > curHeight {
			break
		}
	}
}

// UserCheck.go
func UserCheck(toAddr string) (string, error) {
	// 예시 CLI 호출
	cmd := exec.Command("build/simd", "tx", "bank", "send",
		"alice", toAddr, "0stake",
		"--gas", "auto",
		"--chain-id", "learning-chain-1",
		"--home", "private/.simapp",
		"--yes", "--keyring-backend", "test", "--broadcast-mode", "sync")

	out, err := cmd.CombinedOutput()
	return string(out), err
}

func QueryBalance(node_id, address string) (string, error) {
	// simd CLI를 통한 잔고 조회
	cmd := exec.Command("build/simd", "query", "bank", "balances", address,
		"--node", "tcp://localhost:26657", // RPC 노드 주소 (필요 시 수정 가능)
		"--home", "private/.simapp",
		"--output", "json")

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("[QueryBalance] 잔고 조회 실패: %v\n출력: %s", err, string(out))
	}

	var resp struct {
		Balances []struct {
			Denom  string `json:"denom"`
			Amount string `json:"amount"`
		} `json:"balances"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return "", fmt.Errorf("[QueryBalance] 잔고 JSON 파싱 실패: %v\n출력: %s", err, string(out))
	}

	if len(resp.Balances) == 0 {
		return "", fmt.Errorf("[QueryBalance] 잔고 없음")
	}

	denom := resp.Balances[0].Denom
	amount := resp.Balances[0].Amount

	balance := fmt.Sprintf("%s%s", amount, denom)
	sendBalanceTopic(node_id, address, balance)

	return balance, nil
}

func sendBalanceTopic(nodeId, address, balance string) error {
	msg := types.BalanceResult{
		NodeId:  nodeId,
		Address: address,
		Balance: balance,
	}
	bytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	kafkaMsg := &sarama.ProducerMessage{
		Topic: config.TopicBalanceResult,
		Value: sarama.ByteEncoder(bytes),
	}
	_, _, err = producer.KafkaProducerBalance.SendMessage(kafkaMsg)
	return err
}

// var reSeqMismatch = regexp.MustCompile(`expected\s+(\d+),\s*got\s+(\d+)`)

// func runTx(args []string) (stdout, stderr []byte, err error) {
// 	cmd := exec.Command("build/simd", args...)
// 	var outBuf, errBuf bytes.Buffer
// 	cmd.Stdout, cmd.Stderr = &outBuf, &errBuf
// 	err = cmd.Run()
// 	return outBuf.Bytes(), errBuf.Bytes(), err
// }

func SendTxHash(addr, hash string) error {
	msg := types.TxHashResult{
		Address: addr,
		Hash:    hash,
	}
	bytes, err := json.Marshal(msg)
	if err != nil {
		fmt.Printf("[Kafka: TxHash] JSON 직렬화 실패: %v\n", err)
		return err
	}

	fmt.Printf("[Kafka: TxHash] 전송 준비: addr=%s hash=%s\n", addr, hash)

	kafkaMsg := &sarama.ProducerMessage{
		Topic: config.TopicTxHash,
		Value: sarama.ByteEncoder(bytes),
	}
	partition, offset, err := producer.KafkaProducerTxHash.SendMessage(kafkaMsg)
	if err != nil {
		fmt.Printf("[Kafka: TxHash] Kafka 전송 실패: %v\n", err)
		return err
	}

	fmt.Printf("[Kafka: TxHash] Kafka 전송 성공 (partition=%d offset=%d)\n", partition, offset)
	return nil
}

func AddEnergyCLI(clientCtx client.Context, toAddr string, whAmt sdk.Int, txHash string) (int64, []*rewardtypes.Contributor, string, error) {
	cliLock.Lock()
	defer cliLock.Unlock()

	msg := &rewardtypes.MsgAddEnergy{
		Creator: clientCtx.GetFromAddress().String(),
		To:      toAddr,
		Amount:  whAmt.String(),
		TxHash:  txHash,
	}

	txBuilder := clientCtx.TxConfig.NewTxBuilder()
	if err := txBuilder.SetMsgs(msg); err != nil {
		return 0, nil, "", fmt.Errorf("[AddEnergyCLI] SetMsgs 실패: %w", err)
	}

	fromAddr := clientCtx.GetFromAddress()

	// ✅ seqCache 재사용
	if !cacheInitialized {
		accNum, seq, err := clientCtx.AccountRetriever.GetAccountNumberSequence(clientCtx, fromAddr)
		if err != nil {
			return 0, nil, "", fmt.Errorf("[AddEnergyCLI] AccountRetriever 실패: %w", err)
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
		WithMemo("add-energy")

	_, gasWanted, err := tx.CalculateGas(clientCtx, txf, msg)
	if err != nil {
		seqCache = forceSyncAccountSequence(clientCtx)

		return 0, nil, "", fmt.Errorf("[AddEnergyCLI] 가스 계산 실패: %w", err)
	}

	gasLimit := uint64(float64(gasWanted)*1.3) + 10000
	txBuilder.SetGasLimit(gasLimit)

	// 수수료 계산
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

		return 0, nil, "", fmt.Errorf("[AddEnergyCLI] 서명 실패: %w", err)
	}
	txBytes, err := clientCtx.TxConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		seqCache = forceSyncAccountSequence(clientCtx)

		return 0, nil, "", fmt.Errorf("[AddEnergyCLI] 인코딩 실패: %w", err)
	}

	res, err := clientCtx.BroadcastTxCommit(txBytes)
	if err != nil {
		seqCache = forceSyncAccountSequence(clientCtx)

		return 0, nil, "", fmt.Errorf("[AddEnergyCLI] 브로드캐스트 실패: %w", err)
	}

	if res.Code != 0 {
		seqCache = forceSyncAccountSequence(clientCtx)

		return 0, nil, res.RawLog, fmt.Errorf("[AddEnergyCLI] DeliverTx 실패: code=%d log=%s", res.Code, res.RawLog)
	}

	var recs int64
	var contributors []*rewardtypes.Contributor
	for _, ev := range res.Logs[0].Events {
		if ev.Type == "add_energy" {
			for _, attr := range ev.Attributes {
				switch attr.Key {
				case "recs":
					fmt.Sscan(attr.Value, &recs)
				case "contributors":
					_ = json.Unmarshal([]byte(attr.Value), &contributors)
				}
			}
		}
	}
	waitForNextBlock(clientCtx)

	fmt.Printf("\033[32m[AddEnergyCLI] 성공 (seq=%d)\033[0m\n", seqCache)
	return recs, contributors, res.RawLog, nil
}

func CreateRECRecordCLI(clientCtx client.Context, count int64) ([]string, error) {
	cliLock.Lock()
	defer cliLock.Unlock()

	msg := &rewardtypes.MsgCreateRECRecord{
		Creator: clientCtx.GetFromAddress().String(),
		Count:   count,
	}

	txBuilder := clientCtx.TxConfig.NewTxBuilder()
	if err := txBuilder.SetMsgs(msg); err != nil {
		return nil, fmt.Errorf("[CreateRECRecord] SetMsgs 실패: %w", err)
	}

	fromAddr := clientCtx.GetFromAddress()

	// ✅ AccountNumber / Sequence 캐시 재사용
	if !cacheInitialized {
		accNum, seq, err := clientCtx.AccountRetriever.GetAccountNumberSequence(clientCtx, fromAddr)
		if err != nil {
			return nil, fmt.Errorf("[CreateRECRecord] AccountRetriever 실패: %w", err)
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
		WithMemo("create-rec-record")

	_, gasWanted, err := tx.CalculateGas(clientCtx, txf, msg)
	if err != nil {
		seqCache = forceSyncAccountSequence(clientCtx)
		return nil, fmt.Errorf("[CreateRECRecord] 가스 계산 실패: %w", err)
	}

	gasLimit := uint64(float64(gasWanted)*1.3) + 10000
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
		return nil, fmt.Errorf("[CreateRECRecord] 서명 실패: %w", err)
	}

	txBytes, err := clientCtx.TxConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		seqCache = forceSyncAccountSequence(clientCtx)
		return nil, fmt.Errorf("[CreateRECRecord] 인코딩 실패: %w", err)
	}

	res, err := clientCtx.BroadcastTxCommit(txBytes)
	if err != nil {
		seqCache = forceSyncAccountSequence(clientCtx)
		return nil, fmt.Errorf("[CreateRECRecord] 브로드캐스트 실패: %w", err)
	}

	if res.Code != 0 {
		seqCache = forceSyncAccountSequence(clientCtx)
		return nil, fmt.Errorf("[CreateRECRecord] DeliverTx 실패: code=%d log=%s", res.Code, res.RawLog)
	}

	fmt.Printf("\033[32m[CreateRECRecord] 성공 (seq=%d)\033[0m\n", seqCache)

	// ✅ RawLog에서 rec_id 리스트 추출
	var recIDs []string
	if len(res.Logs) > 0 {
		for _, evt := range res.Logs[0].Events {
			if evt.Type == "message" || evt.Type == "create_rec_record" {
				for _, attr := range evt.Attributes {
					if attr.Key == "rec_id" {
						recIDs = append(recIDs, attr.Value)
					}
				}
			}
		}

	}

	// ✅ RawLog JSON 파싱 fallback
	if len(recIDs) == 0 {
		var raw []map[string]interface{}
		if err := json.Unmarshal([]byte(res.RawLog), &raw); err == nil {
			for _, log := range raw {
				if events, ok := log["events"].([]interface{}); ok {
					for _, e := range events {
						ev := e.(map[string]interface{})
						if ev["type"] == "message" {
							if attrs, ok := ev["attributes"].([]interface{}); ok {
								for _, a := range attrs {
									attr := a.(map[string]interface{})
									if attr["key"] == "rec_id" {
										recIDs = append(recIDs, attr["value"].(string))
									}
								}
							}
						}
					}
				}
			}
		}
	}

	waitForNextBlock(clientCtx)
	return recIDs, nil
}

// AppendTxHashCLI : LinkedList 노드 추가 (nodeCreator, recID 전달)
func CreateBlockCLI(clientCtx client.Context, nodeCreator, recID string) (string, error) {
	cliLock.Lock()
	defer cliLock.Unlock()

	msg := &rewardtypes.MsgAppendTxHash{
		NodeCreator: nodeCreator,
		RecId:       recID,
		Creator:     clientCtx.GetFromAddress().String(),
	}

	txBuilder := clientCtx.TxConfig.NewTxBuilder()
	if err := txBuilder.SetMsgs(msg); err != nil {
		return "", fmt.Errorf("[CreateBlockCLI] SetMsgs 실패: %w", err)
	}

	fromAddr := clientCtx.GetFromAddress()

	// ✅ AccountNumber / Sequence 캐시 재사용
	if !cacheInitialized {
		accNum, seq, err := clientCtx.AccountRetriever.GetAccountNumberSequence(clientCtx, fromAddr)
		if err != nil {
			return "", fmt.Errorf("[CreateBlockCLI] AccountRetriever 실패: %w", err)
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
		WithMemo("append-tx-hash")

	_, gasWanted, err := tx.CalculateGas(clientCtx, txf, msg)
	if err != nil {
		seqCache = forceSyncAccountSequence(clientCtx)

		return "", fmt.Errorf("[CreateBlockCLI] 가스 계산 실패: %w", err)
	}

	gasLimit := uint64(float64(gasWanted)*1.2) + 10000
	txBuilder.SetGasLimit(gasLimit)

	// 수수료 계산
	gasPrices, _ := sdk.ParseDecCoins("0.025stake")
	decGas := sdk.NewDec(int64(gasLimit))
	decFees := gasPrices.MulDec(decGas)
	fees, _ := decFees.TruncateDecimal()
	if fees.IsZero() {
		fees = sdk.NewCoins(sdk.NewInt64Coin("stake", 1))
	}
	txBuilder.SetFeeAmount(fees)

	// 서명
	if err := tx.Sign(txf, clientCtx.FromName, txBuilder, true); err != nil {
		seqCache = forceSyncAccountSequence(clientCtx)
		waitForNextBlock(clientCtx)

		return "", fmt.Errorf("[CreateBlockCLI] 서명 실패: %w", err)
	}
	txBytes, err := clientCtx.TxConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		seqCache = forceSyncAccountSequence(clientCtx)

		return "", fmt.Errorf("[CreateBlockCLI] 인코딩 실패: %w", err)
	}

	// 트랜잭션 전송
	res, err := clientCtx.BroadcastTxCommit(txBytes)
	if err != nil {
		seqCache = forceSyncAccountSequence(clientCtx)
		waitForNextBlock(clientCtx)

		return "", fmt.Errorf("[CreateBlockCLI] 브로드캐스트 실패: %w", err)
	}

	if res.Code != 0 {
		seqCache = forceSyncAccountSequence(clientCtx)
		waitForNextBlock(clientCtx)

		return res.RawLog, fmt.Errorf("[CreateBlockCLI] DeliverTx 실패: code=%d log=%s", res.Code, res.RawLog)
	}

	fmt.Printf("\033[32m[CreateBlockCLI] 성공 (seq=%d)\033[0m\n", seqCache)
	return res.TxHash, nil
}

func BlockCreatorRewardPercentCLI(clientCtx client.Context, address string, percent string) (string, error) {
	cliLock.Lock()
	defer cliLock.Unlock()

	msg := &rewardtypes.MsgDistributeRewardPercent{
		Creator: clientCtx.GetFromAddress().String(),
		Address: address,
		Percent: percent,
	}

	txBuilder := clientCtx.TxConfig.NewTxBuilder()
	if err := txBuilder.SetMsgs(msg); err != nil {
		return "", fmt.Errorf("[BlockCreatorReward] SetMsgs 실패: %w", err)
	}

	fromAddr := clientCtx.GetFromAddress()

	// ✅ AccountNumber / Sequence 캐시 재사용
	if !cacheInitialized {
		accNum, seq, err := clientCtx.AccountRetriever.GetAccountNumberSequence(clientCtx, fromAddr)
		if err != nil {
			return "", fmt.Errorf("[BlockCreatorReward] AccountRetriever 실패: %w", err)
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
		WithMemo("distribute-reward-percent")

	_, gasWanted, err := tx.CalculateGas(clientCtx, txf, msg)
	if err != nil {
		seqCache = forceSyncAccountSequence(clientCtx)

		return "", fmt.Errorf("[BlockCreatorReward] 가스 계산 실패: %w", err)
	}

	gasLimit := uint64(float64(gasWanted)*1.3) + 10000
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

		return "", fmt.Errorf("[BlockCreatorReward] 서명 실패: %w", err)
	}
	txBytes, err := clientCtx.TxConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		seqCache = forceSyncAccountSequence(clientCtx)

		return "", fmt.Errorf("[BlockCreatorReward] 인코딩 실패: %w", err)
	}

	res, err := clientCtx.BroadcastTxCommit(txBytes)
	if err != nil {
		seqCache = forceSyncAccountSequence(clientCtx)

		return "", fmt.Errorf("[BlockCreatorReward] 브로드캐스트 실패: %w", err)
	}

	if res.Code != 0 {
		seqCache = forceSyncAccountSequence(clientCtx)

		return res.RawLog, fmt.Errorf("[BlockCreatorReward] DeliverTx 실패: code=%d log=%s", res.Code, res.RawLog)
	}

	fmt.Printf("\033[32m[BlockCreatorReward] 성공 (seq=%d)\033[0m\n", seqCache)
	waitForNextBlock(clientCtx)
	return res.TxHash, nil
}
