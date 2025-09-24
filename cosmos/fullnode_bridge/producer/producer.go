package producer

import (
	"log"

	"github.com/cosmos/cosmos-sdk/fullnode_bridge/config"

	"github.com/IBM/sarama"
)

var KafkaProducerDevice sarama.SyncProducer // 디바이스 정보 전송 프로듀서
var KafkaContributors sarama.SyncProducer   // 발전량 데이터 전송
var KafkaProducerVMember sarama.SyncProducer
var KafkaProducerLatLng sarama.SyncProducer  // 위도경도 전송용 프로듀서
var KafkaProducerBalance sarama.SyncProducer // 잔고 전송 프로듀서
var KafkaProducerTxHash sarama.SyncProducer

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

func InitProducer() {
	KafkaProducerDevice = NewKafkaSyncProducer(config.KafkaBrokers)
	KafkaProducerLatLng = NewKafkaSyncProducer(config.KafkaBrokers)
	KafkaProducerVMember = NewKafkaSyncProducer(config.KafkaBrokers)
	KafkaProducerBalance = NewKafkaSyncProducer(config.KafkaBrokers)
	KafkaProducerTxHash = NewKafkaSyncProducer(config.KafkaBrokers)
	KafkaContributors = NewKafkaSyncProducer(config.KafkaBrokers)
}
