package tx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/cosmos/cosmos-sdk/fullnode_bridge/config"
	"github.com/cosmos/cosmos-sdk/fullnode_bridge/producer"
	"github.com/cosmos/cosmos-sdk/fullnode_bridge/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	lighttxtypes "github.com/cosmos/cosmos-sdk/x/light_tx/types"
	rewardtypes "github.com/cosmos/cosmos-sdk/x/reward/types"
	grpc "google.golang.org/grpc"
)

func BroadcastLightTxWithReward(clientCtx client.Context, grpcConn *grpc.ClientConn, msg types.LightTxMessage, addr string, power float64) (string, error) {
	// --- clientCtx 필수 요소 가드 ---
	if clientCtx.TxConfig == nil {
		return "", fmt.Errorf("clientCtx.TxConfig is nil")
	}
	if clientCtx.Keyring == nil {
		return "", fmt.Errorf("clientCtx.Keyring is nil")
	}
	if clientCtx.AccountRetriever == nil {
		clientCtx = clientCtx.WithAccountRetriever(authtypes.AccountRetriever{})
	}
	if clientCtx.FromName == "" || clientCtx.FromAddress.Empty() {
		return "", fmt.Errorf("clientCtx.FromName/FromAddress not set")
	}

	if power <= 0 {
		return "", fmt.Errorf("보상할 발전량이 없습니다")
	}

	signer := clientCtx.GetFromAddress().String() // ← alice 주소

	// 1) LightTx 메시지
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
		return "", fmt.Errorf("no valid LightTx data")
	}

	// 2) Reward 메시지(수령자는 addr, 서명자는 alice)
	if power <= 0 {
		return "", fmt.Errorf("보상할 발전량이 없습니다")
	}
	rewardMsg := &rewardtypes.MsgRewardSolarPower{
		Creator: signer,
		Address: addr, // 보상받을 사용자 주소
		Amount:  strconv.FormatInt(int64(math.Round(power)), 10),
	}

	// 3) TxBuilder에 Msg 두 개 추가
	txBuilder := clientCtx.TxConfig.NewTxBuilder()
	if err := txBuilder.SetMsgs(lightMsg, rewardMsg); err != nil {
		return "", fmt.Errorf("failed to set msgs: %w", err)
	}

	// 4) Factory (0.45: PrepareFactory / CalculateFees 없음)
	fromName := clientCtx.FromName
	fromAddr := clientCtx.GetFromAddress()

	txf := tx.Factory{}.
		WithChainID(clientCtx.ChainID).
		WithTxConfig(clientCtx.TxConfig).
		WithAccountRetriever(clientCtx.AccountRetriever).
		WithGasAdjustment(1.2).      // 시뮬 보정치
		WithGasPrices("0.025stake"). // 기준 가스프라이스
		WithMemo("light-tx").
		WithKeybase(clientCtx.Keyring)

	// 4-1) 계정번호/시퀀스 조회 후 팩토리에 주입
	accNum, seq, err := clientCtx.AccountRetriever.GetAccountNumberSequence(clientCtx, fromAddr)
	if err != nil {
		return "", fmt.Errorf("failed to get account/sequence: %w", err)
	}
	txf = txf.WithAccountNumber(accNum).WithSequence(seq)

	// 5) 가스 시뮬레이션 → 여유 가스 산정
	_, gasWanted, err := tx.CalculateGas(clientCtx, txf, lightMsg, rewardMsg)
	if err != nil {
		return "", fmt.Errorf("failed to simulate gas: %w", err)
	}

	adjustedGas := uint64(float64(gasWanted)*1.3) + 10000 // 여유치
	txBuilder.SetGasLimit(adjustedGas)

	// 6) 수수료 수동 계산 (DecCoins → Coins)
	gasPrices, err := sdk.ParseDecCoins("0.025stake")
	if err != nil {
		return "", fmt.Errorf("invalid gas prices: %w", err)
	}
	decGas := sdk.NewDec(int64(adjustedGas))
	decFees := gasPrices.MulDec(decGas)  // DecCoins * Dec
	fees, _ := decFees.TruncateDecimal() // Coins 로 반올림 버림
	if fees.IsZero() {                   // 최소 1 udenom 같은 가드(옵션)
		fees = sdk.NewCoins(sdk.NewInt64Coin("stake", 1))
	}
	txBuilder.SetFeeAmount(fees)

	// 7) 서명
	if err := tx.Sign(txf, fromName, txBuilder, true); err != nil {
		return "", fmt.Errorf("failed to sign tx: %w", err)
	}

	// 8) 브로드캐스트
	txBytes, err := clientCtx.TxConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		return "", fmt.Errorf("failed to encode tx: %w", err)
	}
	res, err := clientCtx.BroadcastTxCommit(txBytes)
	if err != nil {
		return "", fmt.Errorf("broadcast failed: %w", err)
	}
	// code 체크 (0이 아니면 실패)
	if res.Code != 0 {
		return res.TxHash, fmt.Errorf("deliverTx failed: code=%d codespace=%s raw_log=%s", res.Code, res.Codespace, res.RawLog)
	}

	fmt.Println("[Kafka: SolarData] 트랜잭션 전송 성공")
	return res.TxHash, nil
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

func QueryBalance(address string) (string, error) {
	// simd CLI를 통한 잔고 조회
	cmd := exec.Command("build/simd", "query", "bank", "balances", address,
		"--node", "tcp://localhost:26657", // RPC 노드 주소 (필요 시 수정 가능)
		"--home", "private/.simapp",
		"--output", "json")

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("잔고 조회 실패: %v\n출력: %s", err, string(out))
	}

	var resp struct {
		Balances []struct {
			Denom  string `json:"denom"`
			Amount string `json:"amount"`
		} `json:"balances"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return "", fmt.Errorf("잔고 JSON 파싱 실패: %v\n출력: %s", err, string(out))
	}

	if len(resp.Balances) == 0 {
		return "", fmt.Errorf("잔고 없음")
	}

	denom := resp.Balances[0].Denom
	amount := resp.Balances[0].Amount

	balance := fmt.Sprintf("%s%s", amount, denom)
	sendBalanceTopic(address, balance)

	return balance, nil
}

func sendBalanceTopic(address, balance string) error {
	msg := types.BalanceResult{
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

// var txLock sync.Mutex

// func SendRewardTxSafely(addr string, amount float64, creator bool) error {
// 	txLock.Lock()
// 	defer txLock.Unlock()
// 	_, err := SendRewardTx(addr, amount, creator)
// 	return err
// }

var reSeqMismatch = regexp.MustCompile(`expected\s+(\d+),\s*got\s+(\d+)`)

func runTx(args []string) (stdout, stderr []byte, err error) {
	cmd := exec.Command("build/simd", args...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout, cmd.Stderr = &outBuf, &errBuf
	err = cmd.Run()
	return outBuf.Bytes(), errBuf.Bytes(), err
}

func SendRewardTx(toAddr string, reward float64) (string, error) {

	amount := strconv.FormatInt(int64(math.Round(reward)), 10)

	base := []string{
		"tx", "reward", "reward-solar-power", toAddr, amount,
		"--from", "alice",
		"--chain-id", "learning-chain-1",
		"--home", "private/.simapp",
		"--keyring-backend", "test",
		"--broadcast-mode", "sync",
		"--output", "json",
		"--node", "tcp://localhost:26657",
		"--gas", "auto", "--gas-adjustment", "1.2",
		"--fees", "100stake",
		"--yes",
	}

	var stdout, stderr []byte
	var err error

	for attempt := 1; attempt <= 3; attempt++ {
		stdout, stderr, err = runTx(base)
		fmt.Printf("[Kafka: reward][try #%d] stdout: %s\n", attempt, stdout)

		if err == nil {
			break
		}

		// 시퀀스 mismatch면 expected로 다음 시도
		if m := reSeqMismatch.FindStringSubmatch(string(stderr)); len(m) == 3 {
			exp, _ := strconv.ParseUint(m[1], 10, 64)
			fmt.Printf("[Kafka: reward] seq mismatch → expected=%d로 재시도\n", exp)
			// base에 --sequence만 추가하여 덮어쓰기
			base = append(base, "--sequence", fmt.Sprintf("%d", exp))
			// 짧은 백오프
			time.Sleep(200 * time.Millisecond)
			continue
		}
		// 다른 오류면 바로 종료
		break
	}

	if err != nil {
		return "", fmt.Errorf("[Kafka: reward] 전송 실패: %v\nstderr: %s\nstdout: %s", err, stderr, stdout)
	}

	go func(addr string) {
		time.Sleep(10 * time.Second)
		if balance, err := QueryBalance(addr); err != nil {
			fmt.Printf("[Kafka: reward] 잔고 조회 실패: %v\n", err)
		} else {
			fmt.Printf("[Kafka: reward] %s 잔고: %s\n", addr, balance)
		}
	}(toAddr)

	return "", nil
}

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

// AddEnergyCLI : 발전량 기록 및 REC 계산
func AddEnergyCLI(toAddr string, whAmt sdk.Int, txHash string) (int64, []*rewardtypes.Contributor, string, error) {
	cmd := exec.Command("./build/simd", "tx", "reward", "add-energy",
		toAddr, whAmt.String(), txHash,
		"--from", "alice",
		"--chain-id", "learning-chain-1",
		"--keyring-backend", "test",
		"--home", "private/.simapp",
		"--gas", "auto",
		"--gas-adjustment", "1.2",
		"--yes",
		"--broadcast-mode", "block",
		"-o", "json", // JSON 출력 옵션
	)

	out, err := cmd.CombinedOutput()
	outputStr := string(out)

	if err != nil {
		return 0, nil, outputStr, fmt.Errorf("AddEnergy CLI 실행 실패: %w", err)
	}

	// --- gas estimate 텍스트 제거 → 마지막 줄만 JSON으로 파싱 ---
	lines := strings.Split(strings.TrimSpace(outputStr), "\n")
	jsonStr := lines[len(lines)-1]

	// --- JSON 파싱 ---
	var resp struct {
		Logs []struct {
			Events []struct {
				Type       string `json:"type"`
				Attributes []struct {
					Key   string `json:"key"`
					Value string `json:"value"`
				} `json:"attributes"`
			} `json:"events"`
		} `json:"logs"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
		return 0, nil, outputStr, fmt.Errorf("AddEnergy JSON 파싱 실패: %w", err)
	}

	var recs int64
	var contributors []*rewardtypes.Contributor

	for _, log := range resp.Logs {
		for _, ev := range log.Events {
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
	}

	return recs, contributors, outputStr, nil
}

func CreateRECRecordCLI(count int64) (string, error) {
	cmd := exec.Command("./build/simd", "tx", "reward", "create-rec-record",
		strconv.FormatInt(count, 10),
		"--from", "alice",
		"--chain-id", "learning-chain-1",
		"--keyring-backend", "test",
		"--home", "private/.simapp",
		"--gas", "auto",
		"--gas-adjustment", "1.2",
		"--yes",
		"--broadcast-mode", "block",
		"-o", "json", // JSON 출력
	)

	out, err := cmd.CombinedOutput()
	return string(out), err
}

// AppendTxHashCLI : LinkedList 노드 추가 (nodeCreator, recID 전달)
func AppendTxHashCLI(nodeCreator string, recID string) (string, error) {
	cmd := exec.Command("./build/simd", "tx", "reward", "append-tx-hash",
		nodeCreator,       // MsgAppendTxHash.node_creator
		recID,             // MsgAppendTxHash.rec_id
		"--from", "alice", // signer 고정
		"--chain-id", "learning-chain-1",
		"--keyring-backend", "test",
		"--home", "private/.simapp",
		"--gas", "auto",
		"--gas-adjustment", "1.2",
		"--yes",
		"--broadcast-mode", "sync",
		"-o", "json",
	)

	out, err := cmd.CombinedOutput()
	return string(out), err
}

func DistributeRewardPercentCLI(address string, percent float64) (string, error) {
	cmd := exec.Command(
		"./build/simd", "tx", "reward", "distribute-reward-percent",
		address, strconv.FormatFloat(percent, 'f', -1, 64),
		"--from", "alice",
		"--chain-id", "learning-chain-1",
		"--keyring-backend", "test",
		"--home", "private/.simapp",
		"--gas", "auto",
		"--gas-adjustment", "1.2",
		"--yes",
		"--broadcast-mode", "block",
		"-o", "json", // JSON 출력
	)

	out, err := cmd.CombinedOutput()
	return string(out), err
}
