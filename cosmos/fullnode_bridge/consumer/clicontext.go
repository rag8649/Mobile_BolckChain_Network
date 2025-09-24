package consumer

import (
	"context"
	"fmt"
	"os"
	"time"

	rpchttp "github.com/tendermint/tendermint/rpc/client/http"
	"google.golang.org/grpc"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/std"
	sdk "github.com/cosmos/cosmos-sdk/types"

	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	lighttxtypes "github.com/cosmos/cosmos-sdk/x/light_tx/types"
	rewardtypes "github.com/cosmos/cosmos-sdk/x/reward/types"
)

// 1) client.Context는 RPC만(브로드캐스트 용) 붙인다.
func NewClientCtx045() (client.Context, *grpc.ClientConn, codec.ProtoCodecMarshaler) {
	// 인터페이스 레지스트리 & 표준 타입 등록
	reg := codectypes.NewInterfaceRegistry()
	std.RegisterInterfaces(reg)
	lighttxtypes.RegisterInterfaces(reg)
	rewardtypes.RegisterInterfaces(reg)
	authtypes.RegisterInterfaces(reg) // ✅ BaseAccount, ModuleAccount 등록

	cdc := codec.NewProtoCodec(reg)
	amino := codec.NewLegacyAmino()
	std.RegisterLegacyAminoCodec(amino)

	txCfg := authtx.NewTxConfig(cdc, authtx.DefaultSignModes)

	// 키링 & From
	kr, err := keyring.New("learning-chain-1", keyring.BackendTest, "private/.simapp", os.Stdin)
	if err != nil {
		panic(err)
	}
	info, err := kr.Key("alice")
	if err != nil {
		panic(err)
	}
	fromAddr := info.GetAddress()

	// Tendermint RPC (브로드캐스트 용)
	rpc, err := rpchttp.New("tcp://localhost:26657", "/websocket")
	if err != nil {
		panic(err)
	}

	// gRPC는 별도로 리턴 (쿼리/시뮬레이션 용)
	grpcConn, err := grpc.Dial("localhost:9090", grpc.WithInsecure())
	if err != nil {
		panic(err)
	}

	ctx := client.Context{}.
		WithChainID("learning-chain-1").
		WithCodec(cdc).
		WithInterfaceRegistry(reg).
		WithTxConfig(txCfg).
		WithLegacyAmino(amino).
		WithKeyring(kr).
		WithFromName("alice").
		WithFromAddress(fromAddr).
		WithNodeURI("tcp://localhost:26657").
		WithClient(rpc).
		WithAccountRetriever(authtypes.AccountRetriever{}).
		WithBroadcastMode(flags.BroadcastSync)

	return ctx, grpcConn, cdc
}

// 2) gRPC로 account number / sequence 조회 (client.Context를 거치지 않고 직접)
func FetchAccNumSeq(grpcConn *grpc.ClientConn, cdc codec.ProtoCodecMarshaler, addr sdk.AccAddress) (uint64, uint64, error) {
	q := authtypes.NewQueryClient(grpcConn)
	res, err := q.Account(context.Background(), &authtypes.QueryAccountRequest{Address: addr.String()})
	if err != nil {
		return 0, 0, err
	}

	var acc authtypes.AccountI
	if err := cdc.InterfaceRegistry().UnpackAny(res.Account, &acc); err != nil {
		return 0, 0, fmt.Errorf("unpack account: %w", err)
	}
	return acc.GetAccountNumber(), acc.GetSequence(), nil
}

// 3) gRPC로 시뮬레이션 호출
func Simulate(grpcConn *grpc.ClientConn, txCfg client.TxConfig, builder client.TxBuilder) (*txtypes.SimulateResponse, error) {
	svc := txtypes.NewServiceClient(grpcConn)
	txBytes, err := txCfg.TxEncoder()(builder.GetTx())
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return svc.Simulate(ctx, &txtypes.SimulateRequest{TxBytes: txBytes})
}
