package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/fullnode_bridge/tx"
	"github.com/cosmos/cosmos-sdk/fullnode_bridge/types"

	"github.com/cosmos/cosmos-sdk/fullnode_bridge/config"

	"github.com/IBM/sarama"
)

// 회원가입 알고리즘

type accountHandler struct {
	producer    sarama.SyncProducer
	resultTopic string
}

// 응답 메시지 구조체 정의
type BalanceResult struct {
	NodeID  string          `json:"node_id"`
	Address string          `json:"address"`
	Result  json.RawMessage `json:"result"`
}

func (h *accountHandler) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (h *accountHandler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }

func (h *accountHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		var authMsg types.AuthMessage
		if err := json.Unmarshal(msg.Value, &authMsg); err != nil {
			fmt.Println("[Kafka: Account] 메시지 파싱 실패:", err)
			continue
		}

		fmt.Println("[Kafka: Account] 주소 활성화 요청:", authMsg.Address)

		// 1 stake 송금
		_, err := tx.SendStakeToAddress(authMsg.Address)
		if err != nil {
			fmt.Println("[Kafka: Account] 송금 실패:", err)
			continue
		}
		fmt.Println("[Kafka: Account] 송금 성공")

		// ⏱️ 블록 생성 대기 (최대 10초)
		time.Sleep(10 * time.Second)

		// 잔고 조회
		balanceJSON, err := tx.QueryBalance(authMsg.Address)
		if err != nil {
			fmt.Println("[Kafka: Account] 잔고 조회 실패:", err)
			continue
		}
		fmt.Println("[Kafka: Account] 잔고 확인 결과:", balanceJSON)

		// 🔍 JSON에서 balance만 추출
		var balanceResult struct {
			Balances []struct {
				Denom  string `json:"denom"`
				Amount string `json:"amount"`
			} `json:"balances"`
		}
		if err := json.Unmarshal([]byte(balanceJSON), &balanceResult); err != nil {
			fmt.Println("[Kafka: Account] 잔고 JSON 파싱 실패:", err)
			continue
		}

		// 필요한 잔액(예: stake)만 추출
		var stakeAmount string
		for _, b := range balanceResult.Balances {
			if b.Denom == "stake" {
				stakeAmount = b.Amount
				break
			}
		}

		if stakeAmount == "" {
			stakeAmount = "0"
		}

		// 🔁 결과 메시지 (잔액만 포함)
		response := struct {
			NodeID  string `json:"node_id"`
			Address string `json:"address"`
			Balance string `json:"balance"`
		}{
			NodeID:  authMsg.NodeID,
			Address: authMsg.Address,
			Balance: stakeAmount,
		}

		encoded, err := json.Marshal(response)
		if err != nil {
			fmt.Println("[Kafka: Account] 결과 메시지 인코딩 실패:", err)
			continue
		}

		// Kafka로 결과 전송
		producerMsg := &sarama.ProducerMessage{
			Topic: h.resultTopic,
			Value: sarama.ByteEncoder(encoded),
		}

		_, _, err = h.producer.SendMessage(producerMsg)
		if err != nil {
			fmt.Println("[Kafka: Account] 결과 메시지 전송 실패:", err)
		} else {
			fmt.Println("[Kafka: Account] 결과 메시지 전송 완료:", string(encoded))
		}

		session.MarkMessage(msg, "")
	}
	return nil
}

func StartAccountConsumer() {
	brokers := config.KafkaBrokers
	topic := config.TopicAccountCreate
	resultTopic := config.TopicAccountResult
	groupID := config.TopicAccountGroup

	saramaConfig := sarama.NewConfig()
	saramaConfig.Version = sarama.V2_1_0_0
	saramaConfig.Consumer.Return.Errors = true
	saramaConfig.Producer.Return.Successes = true
	saramaConfig.Consumer.Offsets.Initial = sarama.OffsetNewest

	// Producer 생성
	producer, err := sarama.NewSyncProducer(brokers, saramaConfig)
	if err != nil {
		panic(fmt.Sprintf("[Kafka: Account] Kafka producer 생성 실패: %v", err))
	}

	// ConsumerGroup 생성
	consumerGroup, err := sarama.NewConsumerGroup(brokers, groupID, saramaConfig)
	if err != nil {
		panic(fmt.Sprintf("[Kafka: Account] Kafka ConsumerGroup 생성 실패: %v", err))
	}

	handler := &accountHandler{
		producer:    producer,
		resultTopic: resultTopic,
	}

	go func() {
		for {
			err := consumerGroup.Consume(context.Background(), []string{topic}, handler)
			if err != nil {
				fmt.Printf("[Kafka: Account] Consume 오류: %v\n", err)
			}
		}
	}()

	fmt.Println("[Kafka: Account] Kafka Consumer Group 수신 대기 중...")
}
