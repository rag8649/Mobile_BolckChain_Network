Full Node Setup Guide

이 문서는 learning-chain-1 풀노드 실행 및 REC 담보 예치 초기 설정 방법을 설명한다.

1. 설치 (Setup)

cosmos 디렉토리에서 setup 스크립트를 실행한다.

cd ../cosmos
./setup.sh


빌드가 완료되면 build/simd 바이너리가 생성된다.

2. Full Node 설정
2.1 config.toml 설정

파일 경로:

private/.simapp/config/config.toml


모든 laddr의 IP를 0.0.0.0으로 변경한다.

예:

laddr = "tcp://0.0.0.0:26657"


(다른 laddr 항목도 동일하게 변경)

2.2 app.toml 설정 (API 활성화)

파일 경로:

private/.simapp/config/app.toml


아래와 같이 설정한다.

[api]
enable = true
swagger = true
address = "tcp://0.0.0.0:1317"

3. Full Node 실행
./build/simd start \
  --home ./private/.simapp


정상 실행 시 Tendermint 및 API 서버가 함께 기동된다.

4. 초기 REC 담보 예치 (Genesis 이후 필수)

REC 담보를 예치하여 보상 시스템을 초기화한다.

./build/simd tx reward deposit-collateral \
  --rec-meta '{
    "facility_id": "FAC12345",
    "facility_name": "Solar Farm B",
    "location": "Seoul",
    "technology_type": "solar",
    "capacity_mw": "10",
    "registration_date": "2025-11-02",
    "certified_id": "CERT19969B",
    "issue_date": "2025-01-02",
    "generation_start_date": "2025-01-01",
    "generation_end_date": "2025-12-31",
    "measured_volume_mwh": "5000",
    "retired_date": "",
    "retirement_purpose": "",
    "status": "active",
    "timestamp": "2025-09-20T01:31:50Z"
  }' \
  --from alice \
  --chain-id learning-chain-1 \
  --home ./private/.simapp \
  --keyring-backend test \
  --gas auto \
  --gas-adjustment 1.2 \
  --yes

5. 조회 API (REST)

API 서버는 기본적으로 1317 포트에서 실행된다.

5.1 예치된 REC 목록 조회
curl http://localhost:1317/cosmos/reward/v1beta1/rec_list

5.2 총 코인 발행량 조회
curl http://localhost:1317/cosmos/reward/v1beta1/supply

5.3 발급된 REC 블록(트랜잭션 노드) 조회
curl http://localhost:1317/cosmos/reward/v1beta1/tx_node_list