package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/IBM/sarama"
	"github.com/cosmos/cosmos-sdk/fullnode_bridge/config"
	prototypes "github.com/cosmos/cosmos-sdk/x/reward/types"
)

// 회원가입 알고리즘

type collateralHandler struct {
	producer    sarama.SyncProducer
	resultTopic string
}

func (h *collateralHandler) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (h *collateralHandler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }

func (h *collateralHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		fmt.Printf("[Kafka: Collateral] 수신 메시지: %s\n", string(msg.Value))

		// 1. Kafka 메시지를 MsgDepositCollateral로 변환
		var collateralMsg prototypes.MsgDepositCollateral
		if err := json.Unmarshal(msg.Value, &collateralMsg); err != nil {
			fmt.Printf("[Kafka: Collateral] JSON 파싱 실패: %v\n", err)
			continue
		}

		// 2. 트랜잭션 실행
		res, err := BroadcastDepositCollateral(&collateralMsg) // 🔥 tx 모듈에서 체인에 브로드캐스트
		if err != nil {
			fmt.Printf("[Kafka: Collateral] 트랜잭션 전송 실패: %v\n", err)
			continue
		}

		// 3. 결과를 resultTopic으로 송신
		resultBytes, _ := json.Marshal(res)
		_, _, err = h.producer.SendMessage(&sarama.ProducerMessage{
			Topic: h.resultTopic,
			Value: sarama.ByteEncoder(resultBytes),
		})
		if err != nil {
			fmt.Printf("[Kafka: Collateral] 결과 전송 실패: %v\n", err)
		}

		// 오프셋 커밋
		session.MarkMessage(msg, "")
	}
	return nil
}

func StartCollateralConsumer() {
	brokers := config.KafkaBrokers
	topic := config.TopicCollateralRequest
	groupID := config.TopicCollateralGroup
	resultTopic := config.TopicCollateralResult

	saramaConfig := sarama.NewConfig()
	saramaConfig.Version = sarama.V2_1_0_0
	saramaConfig.Consumer.Return.Errors = true
	saramaConfig.Producer.Return.Successes = true
	saramaConfig.Consumer.Offsets.Initial = sarama.OffsetNewest

	producer, err := sarama.NewSyncProducer(brokers, saramaConfig)
	if err != nil {
		panic(fmt.Sprintf("[Kafka: Collateral] Kafka producer 생성 실패: %v", err))
	}
	// ConsumerGroup 생성
	consumerGroup, err := sarama.NewConsumerGroup(brokers, groupID, saramaConfig)
	if err != nil {
		panic(fmt.Sprintf("[Kafka: Collateral] Kafka ConsumerGroup 생성 실패: %v", err))
	}

	// handler 생성 (필요하면 producer 주입)
	handler := &collateralHandler{
		producer:    producer, // 필요 시 초기화
		resultTopic: resultTopic,
	}

	go func() {
		for {
			err := consumerGroup.Consume(context.Background(), []string{topic}, handler)
			if err != nil {
				fmt.Printf("[Kafka: Collateral] Consume 오류: %v\n", err)
			}
		}
	}()

	fmt.Println("[Kafka: Collateral] Kafka Consumer Group 수신 대기 중...")
}

// BroadcastDepositCollateral : deposit-collateral 메시지를 브로드캐스트
func BroadcastDepositCollateral(msg *prototypes.MsgDepositCollateral) (string, error) {
	args := []string{
		"tx", "reward", "deposit-collateral",
		string(mustJSON(msg)),
		"--from", "alice",
		"--home", "private/.simapp",
		"--chain-id", "learning-chain-1",
		"--keyring-backend", "test",
		"--broadcast-mode", "block",
		"--node", "http://localhost:26657",
		"--yes",
		"--output", "json",
	}

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

// 헬퍼: JSON 직렬화 (panic-safe)
func mustJSON(v interface{}) []byte {
	bz, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return bz
}
