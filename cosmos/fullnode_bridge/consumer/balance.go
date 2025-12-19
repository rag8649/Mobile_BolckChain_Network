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
		balanceJSON, err := tx.QueryBalance(authMsg.NodeID, authMsg.Address)
		if err != nil {
			fmt.Println("[Kafka: Balance] 잔고 조회 실패:", err)
			continue
		}
		fmt.Println("[Kafka: Balance] 잔고 확인 결과:", balanceJSON)
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

	fmt.Println("[Kafka: Balance] on")
}
