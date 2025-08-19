package consumer

import (
	"encoding/json"
	"log"

	"github.com/IBM/sarama"

	"github.com/cosmos/cosmos-sdk/fullnode_bridge/config"
	"github.com/cosmos/cosmos-sdk/fullnode_bridge/types"
)

func StartRewardOutputConsumer() error {
	log.Println("[Kafka: RewardOut] StartRewardOutputConsumer (no dedupe, no worker)")

	cfg := sarama.NewConfig()
	cfg.Version = sarama.V2_1_0_0

	consumer, err := sarama.NewConsumer(config.KafkaBrokers, cfg)
	if err != nil {
		return err
	}

	const partition = int32(0)
	pc, err := consumer.ConsumePartition(config.TopicResultVMemberReward, partition, sarama.OffsetNewest)
	if err != nil {
		_ = consumer.Close()
		return err
	}

	go func() {
		defer func() {
			_ = pc.Close()
			_ = consumer.Close()
		}()

		for msg := range pc.Messages() {
			if msg == nil || len(msg.Value) == 0 {
				continue
			}

			// 1) 메시지 파싱
			var resp types.MemberRewardOutputMessage
			if err := json.Unmarshal(msg.Value, &resp); err != nil {
				log.Printf("[RewardOut] 파싱 실패: %v", err)
				continue
			}

			// 2) 내 풀노드 요청인지 확인 (아니면 무시)
			if resp.FullnodeID != config.FullnodeID {
				continue
			}

			// 3) 여기서 추가 동작 없이 로그만 남김 (요청사항 반영)
			log.Printf("[RewardOut] 보상 응답 수신: 대상=%d명 req=%s ts=%s",
				len(resp.Rewards), resp.RequestID, resp.Timestamp)

			// 필요 시, 아래 주석을 해제해 즉시 처리 로직을 직접 넣을 수 있습니다.
			// for addr, amt := range resp.Rewards {
			//     // TODO: 즉시 체인 트랜잭션 전송/기록/기타 처리
			// }
		}
	}()

	return nil
}
