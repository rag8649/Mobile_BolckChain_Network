package consumer

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	// "strconv"
	"sync"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/fullnode_bridge/producer"
	"github.com/cosmos/cosmos-sdk/fullnode_bridge/tx"
	"github.com/cosmos/cosmos-sdk/fullnode_bridge/types"

	"github.com/cosmos/cosmos-sdk/fullnode_bridge/config"

	"github.com/IBM/sarama"

	"crypto/sha256"

	"github.com/btcsuite/btcutil/bech32"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"golang.org/x/crypto/ripemd160"
	grpc "google.golang.org/grpc"
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
	clientCtx client.Context // ⚡ 여기서는 타입만 선언
	GRPCConn  *grpc.ClientConn
)

var (
	RECCount        int64
	VoteMemberCount int // 데이터베이스 멤버 수 기록 변수
)

var SentLatLng = make(map[string]bool) // 중복 전송 방지용
var RewardWeight = make(map[string]float64)

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

type BlockCreatorMsg struct {
	Creator      string  `json:"creator"`
	Contribution float64 `json:"contribution"`
	FullnodeID   string  `json:"fullnode_id"`
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
				// sendLocationToKafka(txMsg.Hash, location, config.FullnodeID)
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
			if err := requestDeviceAddress(producer.KafkaProducerDevice, DeviceID[txMsg.Hash]); err != nil {
				fmt.Println("주소 요청 실패:", err)
			}

			go startVoteTimer(producer.KafkaProducerDevice, txMsg.Hash)
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

func startVoteTimer(producer sarama.SyncProducer, hash string) {
	// 투표 수집 대기
	time.Sleep(5 * time.Second)

	// 필요한 데이터만 뽑고 Lock 해제
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

	// TxMsg 복사
	txMsg := entries[0].TxMsg

	// cleanup
	delete(VoteMap, hash)
	delete(SentLatLng, hash)
	delete(RewardWeight, hash)
	VoteMutex.Unlock()

	// -------------------------
	// ⚡ Lock 해제 후 처리 시작
	// -------------------------

	if len(uniqueList) == 0 {
		return
	}

	fmt.Println("[Kafka: Solar data] 서명 조건 충족, 트랜잭션 전송 시작")
	fmt.Printf("[Kafka: Solar data] → 서명자 주소: %v\n", uniqueList)

	// 검증자 보상
	SendValidatorMembers(uniqueList)

	// userAddress 찾기
	addr, ok := deviceAddressMap.Load(DeviceID[txMsg.Hash])
	if !ok {
		fmt.Printf("[Kafka: Solar data] 주소 비어있음 (hash=%s)\n", txMsg.Hash)
		return
	}
	userAddress := addr.(string)

	var power float64
	if txMsg.Original != nil {
		power = txMsg.Original.TotalEnergy
	} else if txMsg.REC != nil {
		mwh, _ := strconv.ParseFloat(txMsg.REC.MeasuredVolumeMWh, 64)
		power = mwh * 1_000_000 // MWh → Wh
	}

	txHash, err := tx.BroadcastLightTxWithReward(clientCtx, GRPCConn, txMsg, userAddress, power)
	if err != nil {
		fmt.Println("[Kafka: Solar data] 트랜잭션 전송 실패:", err)
	} else {
		whAmt := sdk.NewInt(int64(math.Round(power)))

		// AddEnergy 실행
		recs, contributors, output, err := tx.AddEnergyCLI(userAddress, whAmt, txHash)
		if err != nil {
			fmt.Println("[Kafka: AddEnergy] 실행 실패:", err)
			fmt.Println("출력:", output)
			return
		}
		// fmt.Println("[Kafka: AddEnergy] 실행 결과:", output)

		// REC 발급 조건 확인
		if recs > 0 {
			RECCount = recs

			// Kafka로 기여자 리스트 전송
			data := map[string]interface{}{
				"fullnode_id":  config.FullnodeID,
				"contributors": contributors,
			}
			bytes, _ := json.Marshal(data)

			kafkaMsg := &sarama.ProducerMessage{
				Topic: config.TopicContributors,
				Value: sarama.ByteEncoder(bytes),
			}
			_, _, err = producer.SendMessage(kafkaMsg)
			if err != nil {
				fmt.Println("[Kafka: Solar data] 기여자 리스트 전송 실패:", err)
			} else {
				fmt.Println("[Kafka: Solar data] 기여자 리스트 전송 성공 (RECCount:", RECCount, ")")
			}
		}
	}
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
	_, _, err = producer.KafkaProducerVMember.SendMessage(kafkaMsg)
	if err != nil {
		return fmt.Errorf("VMember 메시지 전송 실패: %w", err)
	}

	fmt.Printf("[Kafka: Solar data] VMember 메시지 전송 성공: %+v\n", vMemberMsg)
	return nil
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
	producer.InitProducer()
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

	clientCtx, GRPCConn, _ = NewClientCtx045()
	fmt.Println("[Kafka: Solar data] Kafka Consumer Group 수신 대기 중...")
}

// 🔹 Cosmos SDK tx 결과 구조체
type TxResponse struct {
	Height string `json:"height"`
	TxHash string `json:"txhash"`
	Code   int    `json:"code"`
	Logs   []struct {
		Events []struct {
			Type       string `json:"type"`
			Attributes []struct {
				Key   string `json:"key"`
				Value string `json:"value"`
			} `json:"attributes"`
		} `json:"events"`
	} `json:"logs"`
}

func StartBlockCreatorConsumer() {
	brokers := config.KafkaBrokers
	topic := config.TopicBlockCreator

	cfg := sarama.NewConfig()
	cfg.Version = sarama.V2_1_0_0

	consumer, err := sarama.NewConsumer(brokers, cfg)
	if err != nil {
		panic(fmt.Sprintf("[Kafka: BlockCreator] Consumer 생성 실패: %v", err))
	}

	partitionConsumer, err := consumer.ConsumePartition(topic, 0, sarama.OffsetNewest)
	if err != nil {
		panic(fmt.Sprintf("[Kafka: BlockCreator] 파티션 구독 실패: %v", err))
	}

	fmt.Println("[Kafka: BlockCreator] Consumer 수신 대기 중...")

	go func() {
		for msg := range partitionConsumer.Messages() {
			handleBlockCreatorMessage(msg.Value)
		}
	}()
}

func handleBlockCreatorMessage(msg []byte) {
	var data BlockCreatorMsg
	if err := json.Unmarshal(msg, &data); err != nil {
		fmt.Printf("[Kafka: BlockCreator] JSON 파싱 실패: %v\n", err)
		return
	}

	// 🔹 Fullnode 검증
	if data.FullnodeID != config.FullnodeID {
		fmt.Printf("[Kafka: BlockCreator] Fullnode ID 불일치 → 무시 (got=%s, expected=%s)\n",
			data.FullnodeID, config.FullnodeID)
		return
	}

	fmt.Printf("[Kafka: BlockCreator] 블록 생성자: %s\n", data.Creator)

	// === 최종 REC 발급 ===
	if RECCount <= 0 {
		fmt.Println("[Kafka: BlockCreator] 발급할 REC 없음 (RECCount=0)")
		return
	}

	fmt.Printf("[Kafka: BlockCreator] REC 발급 시작 (count=%d)\n", RECCount)
	output, err := tx.CreateRECRecordCLI(RECCount)
	if err != nil {
		fmt.Println("[Kafka: REC 발급] 실패:", err)
		RECCount = 0
		return
	}

	fmt.Println("[Kafka: REC 발급] 성공:")

	// 🔹 JSON 부분만 추출
	idx := strings.Index(output, "{")
	if idx == -1 {
		fmt.Println("[Kafka: REC 발급] 출력에서 JSON 시작점을 찾을 수 없음")
		RECCount = 0
		return
	}
	jsonPart := output[idx:]

	// 🔹 TxResponse 파싱
	var resp TxResponse
	if err := json.Unmarshal([]byte(jsonPart), &resp); err != nil {
		fmt.Println("[Kafka: REC 발급] JSON 파싱 실패:", err)
		RECCount = 0
		return
	}

	// 🔹 rec_id 추출 (logs에서 그대로 사용)
	recID := extractRecID(resp)
	if recID == "" {
		fmt.Println("[Kafka: REC 발급] rec_id를 찾을 수 없음")
		RECCount = 0
		return
	}

	// === AppendTxHash 트랜잭션 실행 ===
	output, err = tx.AppendTxHashCLI(data.Creator, recID)
	if err != nil {
		fmt.Println("[Kafka: AppendTxHash] 실패:", err)
		fmt.Println("출력:", output)
	} else {
		fmt.Println("[Kafka: AppendTxHash] 성공:", output)
	}

	// 발급 후 RECCount 리셋
	RECCount = 0
}

// 🔹 rec_id 추출 로직 (logs 기준 → base64 디코딩 불필요)
func extractRecID(resp TxResponse) string {
	for _, e := range resp.Logs {
		for _, ev := range e.Events {

			if ev.Type != "create_rec_record" {
				continue
			}

			for _, attr := range ev.Attributes {

				if attr.Key == "rec_id" {
					return attr.Value
				}
			}
		}
	}
	fmt.Println("[extractRecID] rec_id를 찾지 못함")
	return ""
}

func StartConsumer() {

	config.InitConfig()

	go StartSolarKafkaConsumer()    // 태양광 발전량 토픽
	go StartVoteMemberConsumer()    // 회원 수 토픽
	go StartDeviceAddressConsumer() // 디바이스 id, 주소 매핑 토픽

	go StartCollateralConsumer()   // 담보 예치 요청 토픽
	go StartBurnConsumer()         // 소각 요청 토픽
	go StartBalanceConsumer()      // 잔고 확인 토픽
	go StartBlockCreatorConsumer() // 블록 생성
	// go CheckLiquidation()          // REC 가격 조회
}
