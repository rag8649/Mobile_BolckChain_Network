package config

var (
	FullnodeID = "node_A"
	// Kafka 브로커 IP 및 포트
	KafkaBrokers = []string{"192.168.0.173:26650"}

	// Kafka Consumer
	TopicLightTx             = "light-vote-topic"          // 태양광 발전량 메세지 토픽
	TopicAccountCreate       = "create-address-topic"      // 회원가입 메세지 토픽
	TopicVoteMember          = "user-count-topic"          // 전체 유권자 수 갱신 토픽
	TopicDeviceToAddress     = "device-address-topic"      // 오라클 -> 풀노드: 주소값 반환 토픽
	TopicLocationResult      = "result-location-topic"     // 오라클 -> 풀노드: 보상 가중치 반환 토픽
	TopicBalanceRequest      = "request-balance-topic"     // 라이트노드 -> 풀노드: 잔고 확인 메세지
	TopicCollateralRequest   = "request-collaterals-topic" // 라이트 노드 -> 풀노드: REC 담보 예치
	TopicBurnRequest         = "request-burn-topic"        // 라이트노드 -> 풀노드: 소각 요청
	TopicResultVMemberReward = "result-vote-member-reward"
	TopicRECPrice            = "rec-price-topic"
	TopicBlockCreator        = "block-creator" // 블록 생성사 선출 결과

	//Kafka Producer
	TopicDeviceToAddressRequest = "device-address-request-topic" // 풀노드 -> 오라클: 주소값 요청 토픽
	TopicContributors           = "send-contributors"            // 풀노드 -> 기여자 리스트 전송
	TopicRequestMemberCount     = "request-user-count-topic"
	TopicLocationProducer       = "request-location-topic"
	TopicAccountResult          = "result-account-topic" // 회원가입 후 잔액 메세지 토픽
	TopicBalanceResult          = "balance-topic"        // 잔고 전송 토픽
	TopicRequestVMemberReward   = "request-vote-member-topic"
	TopicTxHash                 = "tx-hash-topic"           // 트랜잭션 해시 전송 토픽
	TopicBurnResult             = "result-burn-topic"       // 소각 결과 반환 토픽
	TopicCollateralResult       = "result-collateral-topic" // 풀노드 -> 라이트노드

	//Group
	TopicLightTxGroup         = "light-tx-group"
	TopicDeviceToAddressGroup = "address-device-group"
	TopicAccountGroup         = "account-create-group"
	TopicLocationGroup        = "location-group"
	TopicBalanceGroup         = "balance-group"
	TopicCollateralGroup      = "colleteral-group"
	TopicBurnGroup            = "burn-group"
	TopicRECGroup             = "rec-group"

	CurrentRECPrice = 70000.0 // base
)
