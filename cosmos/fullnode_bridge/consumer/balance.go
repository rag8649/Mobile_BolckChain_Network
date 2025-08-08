package consumer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cosmos/cosmos-sdk/fullnode_bridge/tx"
	"github.com/cosmos/cosmos-sdk/fullnode_bridge/types"

	"github.com/cosmos/cosmos-sdk/fullnode_bridge/config"

	"github.com/IBM/sarama"
)

// 잔고 확인 알고리즘
type balanceHandler struct {
	producer    sarama.SyncProducer
	resultTopic string
}

func (h *balanceHandler) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (h *balanceHandler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }

func (h *balanceHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		var authMsg types.AuthMessage
		if err := json.Unmarshal(msg.Value, &authMsg); err != nil {
			fmt.Println("[Kafka: Balance] 메시지 파싱 실패:", err)
			continue
		}

		// 잔고 조회
		balanceJSON, err := tx.QueryBalance(authMsg.Address)
		if err != nil {
			fmt.Println("[Kafka: Balance] 잔고 조회 실패:", err)
			continue
		}
		fmt.Println("[Kafka: Balance] 잔고 확인 결과:", balanceJSON)

		// 🔍 JSON에서 balance만 추출
		var balanceResult struct {
			Balances []struct {
				Denom  string `json:"denom"`
				Amount string `json:"amount"`
			} `json:"balances"`
		}
		if err := json.Unmarshal([]byte(balanceJSON), &balanceResult); err != nil {
			fmt.Println("[Kafka: Balance] 잔고 JSON 파싱 실패:", err)
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
			fmt.Println("[Kafka: Balance] 결과 메시지 인코딩 실패:", err)
			continue
		}

		// Kafka로 결과 전송
		producerMsg := &sarama.ProducerMessage{
			Topic: h.resultTopic,
			Value: sarama.ByteEncoder(encoded),
		}

		_, _, err = h.producer.SendMessage(producerMsg)
		if err != nil {
			fmt.Println("[Kafka: Balance] 결과 메시지 전송 실패:", err)
		} else {
			fmt.Println("[Kafka: Balance] 결과 메시지 전송 완료:", string(encoded))
		}

		session.MarkMessage(msg, "")
	}
	return nil
}

func StartBalanceConsumer() {
	brokers := config.KafkaBrokers
	topic := config.TopicBalanceRequest
	resultTopic := config.TopicBalanceResult
	groupID := config.TopicBalanceGroup

	saramaConfig := sarama.NewConfig()
	saramaConfig.Version = sarama.V2_1_0_0
	saramaConfig.Consumer.Return.Errors = true
	saramaConfig.Producer.Return.Successes = true
	saramaConfig.Consumer.Offsets.Initial = sarama.OffsetNewest

	// Producer 생성
	producer, err := sarama.NewSyncProducer(brokers, saramaConfig)
	if err != nil {
		panic(fmt.Sprintf("[Kafka: Balance] Kafka producer 생성 실패: %v", err))
	}

	// ConsumerGroup 생성
	consumerGroup, err := sarama.NewConsumerGroup(brokers, groupID, saramaConfig)
	if err != nil {
		panic(fmt.Sprintf("[Kafka: Balance] Kafka ConsumerGroup 생성 실패: %v", err))
	}

	handler := &balanceHandler{
		producer:    producer,
		resultTopic: resultTopic,
	}

	go func() {
		for {
			err := consumerGroup.Consume(context.Background(), []string{topic}, handler)
			if err != nil {
				fmt.Printf("[Kafka: Balance] Consume 오류: %v\n", err)
			}
		}
	}()

	fmt.Println("[Kafka: Balance] Kafka Consumer Group 수신 대기 중...")
}
