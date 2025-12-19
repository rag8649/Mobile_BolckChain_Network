package types

import rewardtypes "github.com/cosmos/cosmos-sdk/x/reward/types"

type SolarData struct {
	DeviceID    string   `json:"device_id"`
	Timestamp   string   `json:"timestamp"`
	TotalEnergy float64  `json:"total_energy"`
	Location    Location `json:"location"`
}

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type LightTxMessage struct {
	Original  *SolarData `json:"original,omitempty"`
	REC       *RECMeta   `json:"rec,omitempty"`
	Hash      string     `json:"hash"`
	Signature string     `json:"signature"`
	Pubkey    string     `json:"pubkey"`
}

type RECMeta struct {
	FacilityID       string `json:"facility_id"`
	FacilityName     string `json:"facility_name"`
	Location         string `json:"location"`
	TechnologyType   string `json:"technology_type"`   // 발전원
	CapacityMW       string `json:"capacity_mw"`       // 설비용량
	RegistrationDate string `json:"registration_date"` // i-REC 등록 승인일

	CertifiedId         string `json:"certified_id"`
	IssueDate           string `json:"issue_date"`
	GenerationStartDate string `json:"generation_start_date"`
	GenerationEndDate   string `json:"generation_end_date"`
	MeasuredVolumeMWh   string `json:"measured_volume_MWh"`
	RetiredDate         string `json:"retired_date"`
	RetirementPurpose   string `json:"retirement_purpose"`
	Status              string `json:"status"`
	Timestamp           string `json:"timestamp"`
}

type DeviceToAddressMessage struct {
	DeviceID string `json:"device_id"`
	Address  string `json:"address"`
	SenderID string `json:"sender_id"`
}

type AuthMessage struct {
	NodeID  string `json:"node_id"`
	Address string `json:"user_address"`
}

// 응답 메시지 구조체 정의
type BalanceResult struct {
	NodeId  string `json:"node_id"`
	Address string `json:"address"`
	Balance string `json:"balance"`
}
type TxHashResult struct {
	Address string `json:"address"`
	Hash    string `json:"hash"`
}

type CollateralMessage struct {
	REC string `json:"rec"`
}

type BurnMessage struct {
	Address string `json:"address"`
	Coin    string `json:"coin"`
}

type BurnResultMessage struct {
	Address     string                  `json:"address"`      // 소각 요청 계정
	Coin        string                  `json:"coin"`         // 소각된 stable 양
	TxHash      string                  `json:"tx_hash"`      // 소각 트랜잭션 해시
	RECRecords  []rewardtypes.RECRecord `json:"rec_records"`  // 반환된 RECRecord 목록
	RECMetas    []RECMeta               `json:"rec_metas"`    // 반환된 RECMeta 목록
	Status      string                  `json:"status"`       // success / error
	ErrorReason string                  `json:"error_reason"` // 실패 시 에러 사유
}
