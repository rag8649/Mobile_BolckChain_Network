package tx

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/cosmos/cosmos-sdk/fullnode_bridge/config"
	"github.com/cosmos/cosmos-sdk/fullnode_bridge/producer"
	"github.com/cosmos/cosmos-sdk/fullnode_bridge/types"
)

func BroadcastLightTx(msg types.LightTxMessage) (string, error) {
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
			msg.REC.IssueData,
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

	return resp.TxHash, nil
}

// SendStakeToAddress.go
func SendStakeToAddress(toAddr string) (string, error) {
	// 예시 CLI 호출
	cmd := exec.Command("build/simd", "tx", "bank", "send",
		"alice", toAddr, "1stake",
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

func SendRewardTx(toAddr string, power float64, creator bool) (string, error) {
	// 발전량이 0 이하이면 트랜잭션 안 보냄
	if power <= 0 {
		return "", fmt.Errorf("[Kafka: reward] 보상할 발전량이 없습니다")
	}

	// 소수점 버림
	amount := int64(power * 10)
	amountStr := strconv.FormatInt(amount, 10)

	// 트랜잭션 실행 명령
	cmd := exec.Command("build/simd", "tx", "reward", "reward-solar-power",
		toAddr, amountStr,
		"--from", "alice",
		"--chain-id", "learning-chain-1",
		"--home", "private/.simapp",
		"--gas", "auto",
		"--yes",
		"--keyring-backend", "test",
		"--broadcast-mode", "sync",
		"--output", "json")

	out, err := cmd.CombinedOutput()

	if err != nil {
		fmt.Println("[Kafka: reward] 트랜잭션 전송 실패:", err)
		return "", fmt.Errorf("[Kafka: reward] 출력 내용: %v\n출력: %s", err, string(out))
	}

	// 필요하다면 txhash 추출 후 블록 포함 여부 확인
	time.Sleep(10 * time.Second)

	var txResp struct {
		TxHash string `json:"txhash"`
	}
	if err := json.Unmarshal(out, &txResp); err != nil {
		fmt.Printf("JSON 파싱 실패: %s\n", err)
	}

	if creator { // 블록 생성자일 경우에만 해시값 전송
		SendTxHash(toAddr, txResp.TxHash)
	}

	// 잔고 조회
	balance, err := QueryBalance(toAddr)
	if err != nil {
		fmt.Println("[Kafka: reward] 잔고 조회 실패:", err)
	} else {
		fmt.Println("[Kafka: reward] 결과:", balance)
	}

	return "", nil
}

func SendTxHash(addr, hash string) error {
	msg := types.TxHashResult{
		Address: addr,
		Hash:    hash,
	}
	bytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	kafkaMsg := &sarama.ProducerMessage{
		Topic: config.TopicTxHash,
		Value: sarama.ByteEncoder(bytes),
	}
	_, _, err = producer.KafkaProducerTxHash.SendMessage(kafkaMsg)
	return err
}
