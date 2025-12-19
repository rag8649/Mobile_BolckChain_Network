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

type burnHandler struct {
	producer    sarama.SyncProducer
	resultTopic string
}

func (h *burnHandler) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (h *burnHandler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }
func (h *burnHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		// ✅ 메시지 단위로 큐에 트랜잭션 실행 함수를 등록
		TxQueue <- func() {
			var burnMsg types.BurnMessage

			// === Kafka 메시지 파싱 ===
			if err := json.Unmarshal(msg.Value, &burnMsg); err != nil {
				fmt.Println("[Kafka: Burn] 메시지 파싱 실패:", err)
				burnMsg.Coin = "err"

				encoded, _ := json.Marshal(burnMsg)
				h.producer.SendMessage(&sarama.ProducerMessage{
					Topic: h.resultTopic,
					Value: sarama.ByteEncoder(encoded),
				})
				return
			}

			// === 트랜잭션 실행 ===
			result, err := tx.BurnStableCoinCLI(clientCtx, burnMsg.Address, burnMsg.Coin)
			if err != nil {
				fmt.Printf("[Kafka: Burn] 소각 실패: %v\n", err)
				burnMsg.Coin = "err"

				encoded, _ := json.Marshal(burnMsg)
				h.producer.SendMessage(&sarama.ProducerMessage{
					Topic: h.resultTopic,
					Value: sarama.ByteEncoder(encoded),
				})
				return
			}

			// === 결과 메시지 구성 ===
			burnResult := tx.BurnResultMessage{
				Address:    burnMsg.Address,
				Status:     result.Status,
				TxHash:     result.TxHash,
				RecRecords: result.RecRecords,
				RecMetas:   result.RecMetas,
			}

			fmt.Printf("[Kafka: Burn] 소각 성공: %+v\n", burnResult)

			// === Kafka 전송 ===
			encoded, err := json.Marshal(burnResult)
			if err != nil {
				fmt.Println("[Kafka: Burn] JSON 직렬화 실패:", err)
				return
			}

			if _, _, err = h.producer.SendMessage(&sarama.ProducerMessage{
				Topic: h.resultTopic,
				Value: sarama.ByteEncoder(encoded),
			}); err != nil {
				fmt.Println("[Kafka: Burn] 결과 메시지 전송 실패:", err)
			} else {
				fmt.Println("[Kafka: Burn] 결과 메시지 전송 완료:", string(encoded))
			}

			// 오프셋 커밋
			session.MarkMessage(msg, "")
		}
	}

	return nil // ✅ for 루프 바깥으로 이동 (전체 메시지 다 처리)
}

func StartBurnConsumer() {
	brokers := config.KafkaBrokers
	topic := config.TopicBurnRequest
	groupID := config.TopicBurnGroup
	resultTopic := config.TopicBurnResult

	saramaConfig := sarama.NewConfig()
	saramaConfig.Version = sarama.V2_1_0_0
	saramaConfig.Consumer.Return.Errors = true
	saramaConfig.Producer.Return.Successes = true
	saramaConfig.Consumer.Offsets.Initial = sarama.OffsetNewest

	producer, err := sarama.NewSyncProducer(brokers, saramaConfig)
	if err != nil {
		panic(fmt.Sprintf("[Kafka: Account] Kafka producer 생성 실패: %v", err))
	}

	// ConsumerGroup 생성
	consumerGroup, err := sarama.NewConsumerGroup(brokers, groupID, saramaConfig)
	if err != nil {
		panic(fmt.Sprintf("[Kafka: Burn] Kafka ConsumerGroup 생성 실패: %v", err))
	}

	// handler 생성 (필요하면 producer 주입)
	handler := &burnHandler{
		producer:    producer, // 필요 시 초기화
		resultTopic: resultTopic,
	}

	go func() {
		for {
			err := consumerGroup.Consume(context.Background(), []string{topic}, handler)
			if err != nil {
				fmt.Printf("[Kafka: Burn] Consume 오류: %v\n", err)
			}
		}
	}()

	fmt.Println("[Kafka: Burn] on")
}
