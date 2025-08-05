package consumer

import (
	"encoding/json"
	"fmt"

	"github.com/cosmos/cosmos-sdk/fullnode_bridge/config"

	"github.com/IBM/sarama"
)

func StartVoteMemberConsumer() {
	fmt.Println("[Kafka: Users] StartVoteMemberConsumer 시작됨")

	brokers := config.KafkaBrokers
	topic := config.TopicVoteMember
	partition := int32(0)

	saramaConfig := sarama.NewConfig()
	saramaConfig.Version = sarama.V2_1_0_0

	// 1. 메시지 요청을 먼저 전송
	go func() {
		err := sendInitialRequest(brokers, config.TopicRequestMemberCount)
		if err != nil {
			fmt.Printf("[Kafka: Users] 초기 요청 전송 실패: %v\n", err)
		} else {
			fmt.Println("[Kafka: Users] 초기 VoteMemberCount 요청 전송 완료")
		}
	}()

	// 2. 컨슈머 초기화
	consumer, err := sarama.NewConsumer(brokers, saramaConfig)
	if err != nil {
		panic(fmt.Sprintf("[Kafka: Users] Consumer 생성 실패: %v", err))
	}

	partitionConsumer, err := consumer.ConsumePartition(topic, partition, sarama.OffsetNewest)
	if err != nil {
		panic(fmt.Sprintf("[Kafka: Users] 파티션 구독 실패: %v", err))
	}

	go func() {
		fmt.Println("[Kafka: Users] Kafka Partition Consumer 수신 대기 중...")
		for msg := range partitionConsumer.Messages() {
			fmt.Printf("[Kafka: Users] 수신 메시지: %s\n", string(msg.Value))

			var parsed VoteMemberMsg
			if err := json.Unmarshal(msg.Value, &parsed); err != nil {
				fmt.Printf("[Kafka: Users] JSON 파싱 오류: %v\n", err)
				continue
			}

			VoteMemberCount = parsed.Count
			fmt.Printf("[Kafka: Users] VoteMemberCount 갱신됨: %d\n", VoteMemberCount)
		}
	}()
}

func sendInitialRequest(brokers []string, topic string) error {
	producer, err := sarama.NewSyncProducer(brokers, nil)
	if err != nil {
		return fmt.Errorf("[Kafka: Users] Kafka 프로듀서 생성 실패: %w", err)
	}
	defer producer.Close()

	// 메시지 내용이 없어도 OK. 수신자(오라클)는 topic만 보면 됨
	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.StringEncoder(`{"request": "latest_vote_count"}`),
	}

	_, _, err = producer.SendMessage(msg)
	return err
}
