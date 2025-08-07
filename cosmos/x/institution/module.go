package institution

import (
	"encoding/json"

	client "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	"github.com/cosmos/cosmos-sdk/x/institution/keeper"
	"github.com/cosmos/cosmos-sdk/x/institution/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"
	abci "github.com/tendermint/tendermint/abci/types"
)

type AppModuleBasic struct{}

func (AppModuleBasic) Name() string {
	return types.ModuleName
}

func NewAppModule(cdc codec.Codec, k keeper.Keeper, sk stakingkeeper.Keeper, bankKeeper bankkeeper.Keeper) AppModule {
	return AppModule{
		cdc:           cdc,
		keeper:        k,
		stakingKeeper: sk,
		bankKeeper:    bankKeeper,
	}
}

type AppModule struct {
	cdc           codec.Codec
	keeper        keeper.Keeper
	stakingKeeper stakingkeeper.Keeper
	bankKeeper    bankkeeper.Keeper
}

func (am AppModule) Name() string { return types.ModuleName }

func (am AppModule) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {}

func (am AppModule) RegisterInterfaces(reg codectypes.InterfaceRegistry) {}

func (am AppModule) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(types.DefaultGenesisState())
}

func (am AppModule) ValidateGenesis(cdc codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	var data types.GenesisState
	if err := cdc.UnmarshalJSON(bz, &data); err != nil {
		return err
	}
	return data.Validate()
}

func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, bz json.RawMessage) []abci.ValidatorUpdate {
	var genesisState types.GenesisState
	cdc.MustUnmarshalJSON(bz, &genesisState)

	// bankKeeper 추가
	InitGenesis(ctx, am.keeper, am.stakingKeeper, am.bankKeeper, genesisState)

	return []abci.ValidatorUpdate{}
}

func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	genesis := am.keeper.ExportGenesis(ctx)
	return cdc.MustMarshalJSON(&genesis)
}

func (am AppModule) BeginBlock(ctx sdk.Context, req abci.RequestBeginBlock) {
	// No-op
}

// EndBlock 실행: 특별한 로직 없으면 빈 구현
func (am AppModule) EndBlock(ctx sdk.Context, req abci.RequestEndBlock) []abci.ValidatorUpdate {
	return []abci.ValidatorUpdate{}
}

// ConsensusVersion: 보통 1 리턴
func (am AppModule) ConsensusVersion() uint64 {
	return 1
}

func (am AppModule) RegisterServices(cfg module.Configurator) {
	// 아직 MsgServer, QueryServer가 없다면 비워둠
}

func (am AppModule) GetQueryCmd() *cobra.Command {
	return &cobra.Command{
		Use:   types.ModuleName,
		Short: "Query commands for the institution module",
	}
}

func (am AppModule) GetTxCmd() *cobra.Command {
	return &cobra.Command{
		Use:   types.ModuleName,
		Short: "Transaction commands for the institution module",
	}
}

func (am AppModule) QuerierRoute() string {
	return types.ModuleName
}

func (am AppModule) LegacyQuerierHandler(amino *codec.LegacyAmino) sdk.Querier {
	return nil
}

func (am AppModule) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
	// 현재 Query gRPC 서비스가 없으면 빈 구현
}

func (am AppModule) RegisterInvariants(ir sdk.InvariantRegistry) {}

func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {}

func (AppModuleBasic) RegisterInterfaces(reg codectypes.InterfaceRegistry) {}

func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(types.DefaultGenesisState())
}

func (am AppModule) RegisterRESTRoutes(clientCtx client.Context, rtr *mux.Router) {
	// 현재 REST 라우터 없음 → 빈 구현
}

func (am AppModule) Route() sdk.Route {
	// gRPC 기반만 사용할 경우, 빈 Route 반환
	return sdk.Route{}
}

func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	var data types.GenesisState
	if err := cdc.UnmarshalJSON(bz, &data); err != nil {
		return err
	}
	return data.Validate()
}

func (AppModuleBasic) GetQueryCmd() *cobra.Command {
	return &cobra.Command{
		Use:   types.ModuleName,
		Short: "Querying commands for the institution module",
	}
}

func (AppModuleBasic) GetTxCmd() *cobra.Command {
	return &cobra.Command{
		Use:   types.ModuleName,
		Short: "Transaction commands for the institution module",
	}
}

func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
	// 아직 쿼리 서비스 정의가 없다면 빈 메서드로 둡니다.
}

func (AppModuleBasic) RegisterRESTRoutes(clientCtx client.Context, rtr *mux.Router) {
	// REST API가 없다면 빈 구현으로 둡니다.
}
