package tx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os/exec"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/cosmos/cosmos-sdk/fullnode_bridge/config"
	"github.com/cosmos/cosmos-sdk/fullnode_bridge/producer"
	"github.com/cosmos/cosmos-sdk/fullnode_bridge/types"
)

func BroadcastLightTx(msg types.LightTxMessage, addr string) (string, error) {
	var args []string

	if msg.Original != nil {
		// SolarData 전송
		args = []string{
			"tx", "lighttx", "send-light-tx",
			msg.Original.DeviceID,
			msg.Original.Timestamp,
			fmt.Sprintf("%.2f", msg.Original.TotalEnergy),
			strconv.FormatFloat(msg.Original.Location.Latitude, 'f', -1, 64),
			strconv.FormatFloat(msg.Original.Location.Longitude, 'f', -1, 64),
			msg.Hash,
			msg.Signature,
			msg.Pubkey,
		}
	} else if msg.REC != nil {
		// RECMeta 전송
		args = []string{
			"tx", "lighttx", "send-light-tx",
			msg.REC.FacilityID,
			msg.REC.FacilityName,
			msg.REC.Location,
			msg.REC.TechnologyType,
			msg.REC.CapacityMW,
			msg.REC.RegistrationDate,
			msg.REC.CertifiedId,
			msg.REC.IssueDate,
			msg.REC.GenerationStartDate,
			msg.REC.GenerationEndDate,
			msg.REC.MeasuredVolumeMWh,
			msg.REC.RetiredDate,
			msg.REC.RetirementPurpose,
			msg.REC.Status,
			msg.REC.Timestamp,
			msg.Hash,
			msg.Signature,
			msg.Pubkey,
		}
	} else {
		return "", fmt.Errorf("no valid data to send (both Original and REC are nil)")
	}

	// 공통 플래그 추가
	args = append(args,
		"--from", "alice",
		"--home", "private/.simapp",
		"--chain-id", "learning-chain-1",
		"--keyring-backend", "test",
		"--broadcast-mode", "block",
		"--node", "http://localhost:26657",
		"--yes",
		"--output", "json",
	)

	cmd := exec.Command("build/simd", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("simd error: %v\noutput: %s", err, string(output))
	}

	var resp struct {
		TxHash string `json:"txhash"`
	}
	if err := json.Unmarshal(output, &resp); err != nil {
		return "", fmt.Errorf("failed to parse tx response: %v\noutput: %s", err, string(output))
	}
	// JSON 파싱

	if resp.TxHash != "" {
		_ = SendTxHash(addr, resp.TxHash)
	}

	return resp.TxHash, nil
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

var txLock sync.Mutex

func SendRewardTxSafely(addr string, amount float64, creator bool) error {
	txLock.Lock()
	defer txLock.Unlock()
	_, err := SendRewardTx(addr, amount, creator)
	return err
}

var reSeqMismatch = regexp.MustCompile(`expected\s+(\d+),\s*got\s+(\d+)`)

func runTx(args []string) (stdout, stderr []byte, err error) {
	cmd := exec.Command("build/simd", args...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout, cmd.Stderr = &outBuf, &errBuf
	err = cmd.Run()
	return outBuf.Bytes(), errBuf.Bytes(), err
}

func SendRewardTx(toAddr string, power float64, creator bool) (string, error) {
	if power <= 0 {
		return "", fmt.Errorf("[Kafka: reward] 보상할 발전량이 없습니다")
	}
	amount := strconv.FormatInt(int64(math.Round(power)), 10)

	base := []string{
		"tx", "reward", "reward-solar-power", toAddr, amount,
		"--from", "alice",
		"--chain-id", "learning-chain-1",
		"--home", "private/.simapp",
		"--keyring-backend", "test",
		"--broadcast-mode", "sync", // 느려져도 안정 원하면 "block"
		"--output", "json",
		"--node", "tcp://localhost:26657", // 조회/브로드캐스트 동일 노드 고정!
		"--gas", "auto", "--gas-adjustment", "1.2",
		"--fees", "100stake",
		"--yes",
	}

	var stdout, stderr []byte
	var err error

	// 최대 3회 시도: 1차(기본) + 미스매치 시 expected로 재시도(최대 2회)
	for attempt := 1; attempt <= 3; attempt++ {
		stdout, stderr, err = runTx(base)
		fmt.Printf("[Kafka: reward][try #%d] stdout: %s\n", attempt, stdout)
		// if len(stderr) > 0 {
		// 	fmt.Printf("[Kafka: reward][try #%d] stderr: %s\n", attempt, stderr)
		// }

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
