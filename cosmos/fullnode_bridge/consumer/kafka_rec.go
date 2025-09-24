package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"strconv"

	"github.com/IBM/sarama"
	"github.com/cosmos/cosmos-sdk/fullnode_bridge/config"
)

// 회원가입 알고리즘

type recHandler struct {
	// producer    sarama.S/
}

type recMessage struct {
	Price string `json:"price"`
}

type CollateralResponse struct {
	TotalAmount string `json:"total_amount"`
}

type SupplyResponse struct {
	Supply struct {
		Minted string `json:"minted"`
	} `json:"supply"`
}

func (h *recHandler) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (h *recHandler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }

func (h *recHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		var recMsg recMessage

		if err := json.Unmarshal(msg.Value, &recMsg); err != nil {
			fmt.Println("[Kafka: REC] 메시지 파싱 실패:", err)
			continue
		}

		priceStr := recMsg.Price // string 타입 예: "1000.50"

		priceFloat, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			log.Printf("[Kafka: REC] REC 가격 변환 실패: %v", err)
			continue
		}

		config.CurrentRECPrice = priceFloat
		log.Printf("[Kafka: REC] 가격 업데이트 완료: %.2f", config.CurrentRECPrice)

		CheckLiquidation()

		session.MarkMessage(msg, "")
	}
	return nil
}

// 담보/공급량 조회 후 청산 필요 여부 확인
func CheckLiquidation() {
	// 1. 담보 조회
	collateralResp, err := http.Get("http://localhost:1317/cosmos/reward/v1beta1/collateral")
	if err != nil {
		log.Printf("Collateral API 조회 실패: %v", err)
		return
	}
	defer collateralResp.Body.Close()

	collateralBody, _ := ioutil.ReadAll(collateralResp.Body)
	var collateral CollateralResponse
	if err := json.Unmarshal(collateralBody, &collateral); err != nil {
		log.Printf("Collateral JSON 파싱 실패: %v", err)
		return
	}

	totalCollateral, _ := strconv.ParseFloat(collateral.TotalAmount, 64)

	// 2. 공급량 조회
	supplyResp, err := http.Get("http://localhost:1317/cosmos/reward/v1beta1/supply")
	if err != nil {
		log.Printf("Supply API 조회 실패: %v", err)
		return
	}
	defer supplyResp.Body.Close()

	supplyBody, _ := ioutil.ReadAll(supplyResp.Body)
	var supply SupplyResponse
	if err := json.Unmarshal(supplyBody, &supply); err != nil {
		log.Printf("Supply JSON 파싱 실패: %v", err)
		return
	}

	totalSupply, _ := strconv.ParseFloat(supply.Supply.Minted, 64)

	// 3. 담보 가치 계산
	collateralValue := totalCollateral * config.CurrentRECPrice
	totalSupply = 100 * totalSupply
	// 4. 비교
	log.Printf("담보량=%.2f, 가격=%.2f → 담보가치=%.2f | 공급량=%.2f",
		totalCollateral, config.CurrentRECPrice, collateralValue, totalSupply)

	const MinCollateralRatio = 1.0 // 담보비율

	if collateralValue/MinCollateralRatio < totalSupply {
		log.Printf("UnderCollateral!!")
		go func() {
			cmd := exec.Command(
				"./build/simd", "tx", "reward", "burn-module-stable", "0",
				"--from", "alice",
				"--chain-id", "learning-chain-1",
				"--home", "./private/.simapp",
				"--keyring-backend", "test",
				"--yes",
				"--broadcast-mode", "block",
			)

			out, err := cmd.CombinedOutput()
			if err != nil {
				log.Printf("[Kafka: REC] burn-module-stable 실행 실패: %v\n출력: %s", err, string(out))
				return
			}
			log.Printf("[Kafka: REC] burn-module-stable 실행 완료\n%s", string(out))
		}()
	}
}

func StartRECConsumer() {
	brokers := config.KafkaBrokers
	topic := config.TopicRECPrice
	groupID := config.TopicRECGroup

	saramaConfig := sarama.NewConfig()
	saramaConfig.Version = sarama.V2_1_0_0
	saramaConfig.Consumer.Return.Errors = true
	saramaConfig.Producer.Return.Successes = true
	saramaConfig.Consumer.Offsets.Initial = sarama.OffsetNewest

	consumerGroup, err := sarama.NewConsumerGroup(brokers, groupID, saramaConfig)
	if err != nil {
		panic(fmt.Sprintf("[Kafka: REC] Kafka ConsumerGroup 생성 실패: %v", err))
	}

	handler := &recHandler{}

	go func() {
		for {
			err := consumerGroup.Consume(context.Background(), []string{topic}, handler)
			if err != nil {
				fmt.Printf("[Kafka: REC] Consume 오류: %v\n", err)
			}
		}
	}()

	fmt.Println("[Kafka: REC] Kafka Consumer Group 수신 대기 중...")
}
