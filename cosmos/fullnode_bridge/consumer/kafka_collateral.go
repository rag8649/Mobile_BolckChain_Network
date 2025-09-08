package consumer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cosmos/cosmos-sdk/fullnode_bridge/types"

	"github.com/IBM/sarama"
	"github.com/cosmos/cosmos-sdk/fullnode_bridge/config"
	"github.com/cosmos/cosmos-sdk/fullnode_bridge/tx"
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
		var collMsg types.CollateralMessage
		if err := json.Unmarshal(msg.Value, &collMsg); err != nil {
			fmt.Println("[Kafka: Collateral] 메시지 파싱 실패:", err)

			// 실패 응답 전송
			collMsg.REC = "err"
			encoded, _ := json.Marshal(collMsg)
			h.producer.SendMessage(&sarama.ProducerMessage{
				Topic: h.resultTopic,
				Value: sarama.ByteEncoder(encoded),
			})
			continue
		}

		// 담보 예치 실행
		result, err := tx.DepositCollateral(collMsg.REC)
		if err != nil {
			fmt.Printf("[Kafka: Collateral] 담보 예치 실패: %v\n", err)

			// 실패 응답 전송
			collMsg.REC = "err"
			encoded, _ := json.Marshal(collMsg)
			h.producer.SendMessage(&sarama.ProducerMessage{
				Topic: h.resultTopic,
				Value: sarama.ByteEncoder(encoded),
			})
			continue
		}

		fmt.Printf("[Kafka: Collateral] 담보 예치 성공: %v\n", result)

		// 성공 응답 전송
		encoded, err := json.Marshal(collMsg)
		if err != nil {
			fmt.Println("[Kafka: Collateral] JSON 직렬화 실패:", err)
			continue
		}

		_, _, err = h.producer.SendMessage(&sarama.ProducerMessage{
			Topic: h.resultTopic,
			Value: sarama.ByteEncoder(encoded),
		})
		if err != nil {
			fmt.Println("[Kafka: Collateral] 결과 메시지 전송 실패:", err)
		} else {
			fmt.Println("[Kafka: Collateral] 결과 메시지 전송 완료:", string(encoded))
		}
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
