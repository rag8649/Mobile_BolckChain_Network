package consumer

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/cosmos/cosmos-sdk/fullnode_bridge/tx"
	"github.com/cosmos/cosmos-sdk/fullnode_bridge/types"

	"github.com/cosmos/cosmos-sdk/fullnode_bridge/config"

	"github.com/IBM/sarama"

	"crypto/sha256"

	"github.com/btcsuite/btcutil/bech32"
	"golang.org/x/crypto/ripemd160"
)

type lightTxHandler struct{}

type SignatureEntry struct {
	TxMsg     types.LightTxMessage
	Address   string
	Timestamp time.Time
}

var (
	VoteMap   = make(map[string][]SignatureEntry) // hash -> 서명자 목록
	DeviceID  = make(map[string]string)           // hash -> device_id
	VoteMutex sync.Mutex
)

var (
	VoteMemberCount int // 데이터베이스 멤버 수 기록 변수
)

var SentLatLng = make(map[string]bool) // 중복 전송 방지용
var RewardWeight = make(map[string]float64)
var KafkaProducerLatLng sarama.SyncProducer // 위도경도 전송용 프로듀서

type Location struct { // 오라클에 전달하는 위치 값
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type LocationOutputMessage struct { // 오라클로부터 받는 결과값
	Hash     string  `json:"hash"`
	Output   float64 `json:"output"`
	SenderID string  `json:"sender_id"`
}

// 오라클로부터 받는 참가자 보상 결과값
type MemberRewardOutputMessage struct {
	SenderID string             `json:"sender_id"` // 메시지 송신자 ID
	Rewards  map[string]float64 `json:"rewards"`   // 참가자 주소 → 보상 금액
}

type VoteMemberMsg struct {
	Count int `json:"count"`
}

var KafkaProducerDevice sarama.SyncProducer // 디바이스 정보 전송 프로듀서
var KafkaProducerVMember sarama.SyncProducer

func InitProducer() {
	KafkaProducerDevice = NewKafkaSyncProducer(config.KafkaBrokers)
	KafkaProducerLatLng = NewKafkaSyncProducer(config.KafkaBrokers)
	KafkaProducerVMember = NewKafkaSyncProducer(config.KafkaBrokers)
}

func NewKafkaSyncProducer(brokers []string) sarama.SyncProducer { // 프로듀서 초기화
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5
	config.Producer.Return.Successes = true

	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		log.Fatalf("Kafka 프로듀서 생성 실패: %v", err)
	}
	return producer
}

func (h *lightTxHandler) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (h *lightTxHandler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }

func (h *lightTxHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error { // 태양광 데이터 수신 처리
	for msg := range claim.Messages() {
		fmt.Println("[Kafka: Solar data][Raw Message]:", string(msg.Value)) // 👉 수신된 원본 메시지 출력

		var txMsg types.LightTxMessage
		if err := json.Unmarshal(msg.Value, &txMsg); err != nil {
			fmt.Println("[Kafka: Solar data] 메시지 파싱 실패:", err)
			continue
		}

		for len(txMsg.Pubkey)%4 != 0 {
			txMsg.Pubkey += "="
		}
		pubkeyBytes, err := base64.StdEncoding.DecodeString(txMsg.Pubkey)
		if err != nil {
			fmt.Println("[Kafka: Solar data] 퍼블릭키 디코딩 실패:", err)
			continue
		}

		if len(pubkeyBytes) != 33 {
			fmt.Println("[Kafka: Solar data] 잘못된 퍼블릭키 길이:", len(pubkeyBytes))
			continue
		}

		address, err := PubKeyToAddress(pubkeyBytes)
		if err != nil {
			fmt.Println("[Kafka: Solar data] 주소 생성 실패:", err)
			continue
		}

		if !SentLatLng[txMsg.Hash] {
			var location Location
			if txMsg.Original != nil {
				location = Location{
					Latitude:  txMsg.Original.Location.Latitude,
					Longitude: txMsg.Original.Location.Longitude,
				}
			}

			// 위도/경도가 모두 0이 아니어야 전송
			if location.Latitude != 0 && location.Longitude != 0 {
				sendLocationToKafka(txMsg.Hash, location, config.FullnodeID)
				SentLatLng[txMsg.Hash] = true
			} else {
				fmt.Println("⚠️ 위도/경도 정보 없음 또는 0, Kafka 전송 생략:", txMsg.Hash)
			}
		}
		VoteMutex.Lock()
		VoteMap[txMsg.Hash] = append(VoteMap[txMsg.Hash], SignatureEntry{
			TxMsg:     txMsg,
			Address:   address,
			Timestamp: time.Now(),
		})

		// DeviceID 저장 로직 유지
		if txMsg.Original != nil && txMsg.Original.DeviceID != "" {
			DeviceID[txMsg.Hash] = txMsg.Original.DeviceID
		} else if txMsg.REC != nil && txMsg.REC.FacilityID != "" {
			DeviceID[txMsg.Hash] = txMsg.REC.FacilityID
		} else {
			fmt.Println("[Kafka: Solar data] DeviceID/FacilityID 없음:", txMsg.Hash)
		}

		// 👉 새로운 해시에 대해서만 goroutine 실행
		if len(VoteMap[txMsg.Hash]) == 1 {
			go startVoteTimer(txMsg.Hash)
		}
		VoteMutex.Unlock()

		session.MarkMessage(msg, "")
	}
	return nil
}

func PubKeyToAddress(pubKeyBytes []byte) (string, error) { // 주소 변환 함수
	// 1. SHA-256
	sha := sha256.Sum256(pubKeyBytes)

	// 2. RIPEMD-160
	ripemd := ripemd160.New()
	_, err := ripemd.Write(sha[:])
	if err != nil {
		return "", err
	}
	pubKeyHash := ripemd.Sum(nil) // 20바이트

	// 3. Bech32 인코딩
	converted, err := bech32.ConvertBits(pubKeyHash, 8, 5, true)
	if err != nil {
		return "", err
	}
	address, err := bech32.Encode("cosmos", converted)
	if err != nil {
		return "", err
	}

	return address, nil
}

// 위치 정보 -> 오라클 전송
func sendLocationToKafka(hash string, loc Location, senderID string) {
	payload := map[string]interface{}{
		"hash":      hash,
		"location":  loc,
		"sender_id": senderID,
	}
	msgBytes, _ := json.Marshal(payload)

	_, _, err := KafkaProducerLatLng.SendMessage(&sarama.ProducerMessage{
		Topic: config.TopicLocationProducer,
		Value: sarama.ByteEncoder(msgBytes),
	})
	if err != nil {
		fmt.Println("[Kafka: Solar data] Location Kafka 전송 실패:", err)
	} else {
		fmt.Println("[Kafka: Solar data] Location Kafka 전송 성공:", string(msgBytes))
	}
}

// 해시별로 10초 대기 후 평가
func startVoteTimer(hash string) {
	time.Sleep(10 * time.Second)

	VoteMutex.Lock()
	entries, ok := VoteMap[hash]
	if !ok || len(entries) == 0 {
		VoteMutex.Unlock()
		return
	}

	// 고유 주소 집합 만들기
	unique := map[string]bool{}
	for _, e := range entries {
		unique[e.Address] = true
	}
	var uniqueList []string
	for k := range unique {
		uniqueList = append(uniqueList, k)
	}

	// 조건 충족하면 트랜잭션 전송
	if len(unique) >= 1 {
		txMsg := entries[0].TxMsg
		fmt.Println("[Kafka: Solar data] 서명 조건 충족, 트랜잭션 전송 시작")

		txHash, err := tx.BroadcastLightTx(txMsg)
		if err != nil {
			fmt.Println("[Kafka: Solar data] 트랜잭션 전송 실패:", err)
		} else {
			fmt.Printf("[Kafka: Solar data] 트랜잭션 전송 성공: %s\n", txHash)
			fmt.Printf("[Kafka: Solar data] → 서명자 주소: %v\n", uniqueList)

			SendValidatorMembers(uniqueList) // 서명자 보상 함수

			// 보상 로직
			deviceId := DeviceID[hash]
			if err := requestDeviceAddress(KafkaProducerDevice, deviceId); err != nil {
				fmt.Println("주소 요청 실패:", err)
			} else {
				var userAddress string
				for i := 0; i < 50; i++ {
					if val, ok := deviceAddressMap.Load(deviceId); ok {
						userAddress = val.(string)
						break
					}
					time.Sleep(100 * time.Millisecond)
				}

				if txMsg.Original != nil {
					tx.SendRewardTx(userAddress, txMsg.Original.TotalEnergy+txMsg.Original.TotalEnergy*RewardWeight[hash])
				} else if txMsg.REC != nil {
					mwh, err := strconv.ParseFloat(txMsg.REC.MeasuredVolumeMWh, 64)
					if err == nil {
						tx.SendRewardTx(userAddress, mwh*1000000) // MWh → Wh
					}
				}
			}
		}
	}

	// cleanup
	delete(VoteMap, hash)
	delete(SentLatLng, hash)
	delete(RewardWeight, hash)
	VoteMutex.Unlock()
}

func requestDeviceAddress(producer sarama.SyncProducer, deviceId string) error { // 주소 요청 함수
	msg := types.DeviceToAddressMessage{
		DeviceID: deviceId,
		SenderID: config.FullnodeID,
	}
	bytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	kafkaMsg := &sarama.ProducerMessage{
		Topic: config.TopicDeviceToAddressRequest,
		Value: sarama.ByteEncoder(bytes),
	}
	_, _, err = producer.SendMessage(kafkaMsg)
	return err
}

var deviceAddressMap = sync.Map{} // deviceId → address

func StartDeviceAddressConsumer() {
	brokers := config.KafkaBrokers
	topic := config.TopicDeviceToAddress
	partition := int32(0)

	cfg := sarama.NewConfig()
	cfg.Version = sarama.V2_1_0_0

	consumer, err := sarama.NewConsumer(brokers, cfg)
	if err != nil {
		panic(fmt.Sprintf("[Kafka: DeviceAddress] Consumer 생성 실패: %v", err))
	}

	partitionConsumer, err := consumer.ConsumePartition(topic, partition, sarama.OffsetNewest)
	if err != nil {
		panic(fmt.Sprintf("[Kafka: DeviceAddress] 파티션 구독 실패: %v", err))
	}

	fmt.Println("[Kafka: DeviceAddress] Consumer 수신 대기 중...")

	go func() {
		for msg := range partitionConsumer.Messages() {
			fmt.Printf("[Kafka: DeviceAddress] 메시지 수신 (offset=%d, partition=%d): %s\n",
				msg.Offset, msg.Partition, string(msg.Value))

			var response types.DeviceToAddressMessage
			if err := json.Unmarshal(msg.Value, &response); err != nil {
				fmt.Printf("[Kafka: DeviceAddress] JSON 파싱 실패: %v\n", err)
				continue
			}

			// 내 노드가 보낸 메시지만 처리
			if response.SenderID != config.FullnodeID {
				continue // 다른 노드의 응답 → 무시
			}

			if response.DeviceID == "" {
				fmt.Printf("⚠️ [Kafka: DeviceAddress] device_id 없음. 무시됨: %v\n", response)
				continue
			}
			if response.Address == "" {
				fmt.Printf("⚠️ [Kafka: DeviceAddress] address 비어 있음. device_id=%s\n", response.DeviceID)
			}

			// 중복 확인
			if val, ok := deviceAddressMap.Load(response.DeviceID); ok {
				fmt.Printf("[Kafka: DeviceAddress] 기존 값 덮어씀: %s → %s (기존=%s)\n",
					response.DeviceID, response.Address, val.(string))
			} else {
				fmt.Printf("[Kafka: DeviceAddress] 저장됨: %s → %s\n", response.DeviceID, response.Address)
			}

			deviceAddressMap.Store(response.DeviceID, response.Address)
		}
	}()
}

func SendValidatorMembers(uniqueList []string) error {
	// 보낼 메시지 구성
	vMemberMsg := map[string]interface{}{
		"fullnode_id": config.FullnodeID,
		"validators":  uniqueList,
		"timestamp":   time.Now().Format(time.RFC3339),
	}

	// JSON 직렬화
	msgBytes, err := json.Marshal(vMemberMsg)
	if err != nil {
		return fmt.Errorf("VMember 메시지 직렬화 실패: %w", err)
	}

	// Kafka 메시지 생성
	kafkaMsg := &sarama.ProducerMessage{
		Topic: config.TopicRequestVMemberReward, // 원하는 토픽명으로 변경 가능
		Value: sarama.ByteEncoder(msgBytes),
	}

	// 전송
	_, _, err = KafkaProducerVMember.SendMessage(kafkaMsg)
	if err != nil {
		return fmt.Errorf("VMember 메시지 전송 실패: %w", err)
	}

	fmt.Printf("[Kafka: Solar data] VMember 메시지 전송 성공: %+v\n", vMemberMsg)
	return nil
}

func StartVMemberConsumer() {
	brokers := config.KafkaBrokers
	topic := config.TopicResultVMemberReward
	partition := int32(0) // 토픽 파티션 고정 "result-location-topic"

	cfg := sarama.NewConfig()
	cfg.Version = sarama.V2_1_0_0

	consumer, err := sarama.NewConsumer(brokers, cfg)
	if err != nil {
		panic(fmt.Sprintf("[Kafka: Member Reward] 단일 Consumer 생성 실패: %v", err))
	}

	partitionConsumer, err := consumer.ConsumePartition(topic, partition, sarama.OffsetNewest)
	if err != nil {
		panic(fmt.Sprintf("[Kafka: Member Reward] 파티션 구독 실패: %v", err))
	}

	go func() {
		fmt.Println("[Kafka: Member Reward] 응답 수신 대기 중...")
		for msg := range partitionConsumer.Messages() {
			fmt.Println("[Kafka: Member Reward] 수신된 메시지:", string(msg.Value))

			var outputMsg MemberRewardOutputMessage
			if err := json.Unmarshal(msg.Value, &outputMsg); err != nil {
				fmt.Println("[Kafka: Member Reward] 메시지 파싱 실패:", err)
				continue
			}

			// ⚠️ 필터링: 내 노드가 보낸 메시지인지 확인
			if outputMsg.SenderID != config.FullnodeID {
				fmt.Printf("[Kafka: Member Reward] id: %s\n", outputMsg.SenderID)
				continue // 내 응답 아님, 무시
			}

			// ✅ Rewards 맵 순회하면서 트랜잭션 전송
			for addr, reward := range outputMsg.Rewards {
				fmt.Printf("[Kafka: Member Reward] 서명자 보상 지급 시작 → 주소: %s, 보상: %f\n", addr, reward)
				if _, err := tx.SendRewardTx(addr, reward); err != nil {
					fmt.Printf("[Kafka: Member Reward] 보상 트랜잭션 실패 (addr=%s): %v\n", addr, err)
				}
			}
		}
	}()

}
func StartLocationOutputConsumer() {
	brokers := config.KafkaBrokers
	topic := config.TopicLocationResult
	partition := int32(0) // 토픽 파티션 고정 "result-location-topic"

	cfg := sarama.NewConfig()
	cfg.Version = sarama.V2_1_0_0

	consumer, err := sarama.NewConsumer(brokers, cfg)
	if err != nil {
		panic(fmt.Sprintf("[Kafka: Location] 단일 Consumer 생성 실패: %v", err))
	}

	partitionConsumer, err := consumer.ConsumePartition(topic, partition, sarama.OffsetNewest)
	if err != nil {
		panic(fmt.Sprintf("[Kafka: Location] 파티션 구독 실패: %v", err))
	}

	// ✅ 메시지 수신 루프
	go func() {
		fmt.Println("[Kafka: Location] 응답 수신 대기 중...")
		for msg := range partitionConsumer.Messages() {
			fmt.Println("[Kafka: Location] 수신된 메시지:", string(msg.Value))

			var outputMsg LocationOutputMessage
			if err := json.Unmarshal(msg.Value, &outputMsg); err != nil {
				fmt.Println("[Kafka: Location] 메시지 파싱 실패:", err)
				continue
			}

			// ⚠️ 필터링: 내 노드가 보낸 메시지인지 확인 (선택적으로 추가)
			if outputMsg.SenderID != config.FullnodeID {
				fmt.Printf("[Kafka: Location] id: %s\n", outputMsg.SenderID)
				continue // 내 응답 아님, 무시
			}

			RewardWeight[outputMsg.Hash] = outputMsg.Output
			// ✅ 처리 로직
			fmt.Printf("[Kafka: Location] 해시: %s, 보상 가중치: %f\n", outputMsg.Hash, RewardWeight[outputMsg.Hash])

		}
	}()
}

func StartSolarKafkaConsumer() {
	brokers := config.KafkaBrokers
	topic := config.TopicLightTx
	groupID := config.TopicLightTxGroup // 모든 서버에서 동일하게 설정해야 함

	saramaConfig := sarama.NewConfig()
	saramaConfig.Version = sarama.V2_1_0_0
	saramaConfig.Consumer.Return.Errors = true
	saramaConfig.Consumer.Offsets.Initial = sarama.OffsetNewest

	consumerGroup, err := sarama.NewConsumerGroup(brokers, groupID, saramaConfig)
	InitProducer()
	if err != nil {
		panic(fmt.Sprintf("[Kafka: Solar data] ConsumerGroup 생성 실패: %v", err))
	}

	go func() {
		for {
			err := consumerGroup.Consume(context.Background(), []string{topic}, &lightTxHandler{})
			if err != nil {
				fmt.Printf("[Kafka: Solar data] Consume 중 오류 발생: %v\n", err)
			}
		}
	}()

	fmt.Println("[Kafka: Solar data] Kafka Consumer Group 수신 대기 중...")
}

func StartConsumer() {
	go StartSolarKafkaConsumer()     // 태양광 발전량 토픽
	go StartLocationOutputConsumer() // 위치 정보 토픽
	go StartVoteMemberConsumer()     // 회원 수 토픽
	go StartDeviceAddressConsumer()  // 디바이스 id, 주소 매핑 토픽

	go StartAccountConsumer() // 회원가입 요청 토픽
	go StartBalanceConsumer() // 잔고 확인 토픽
	go StartVMemberConsumer() // 서명자 보상 토픽
}
