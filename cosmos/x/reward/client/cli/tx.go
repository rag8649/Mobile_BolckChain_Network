package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/x/reward/types"
	"github.com/spf13/cobra"
)

func CmdRewardSolarPower() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reward-solar-power [address] [amount]",
		Short: "Send solar power reward to an address",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			address := args[0]
			amount := args[1]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgRewardSolarPower(
				clientCtx.GetFromAddress().String(), // 보낸 사람
				address,
				amount,
			)

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdBurnStableCoin() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "burn-stable-coin [address] [amount]",
		Short: "Burn stable coin from a specific address and receive stake in return",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetAddr := args[0]
			amount := args[1]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgBurnStableCoin(
				clientCtx.GetFromAddress().String(), // 트랜잭션 실행자(서명자)
				targetAddr,                          // 소각 대상 주소
				amount,
			)

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdDepositCollateral() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deposit-collateral",
		Short: "Deposit a RECRecord or RECMeta as collateral into the reward module",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			// 플래그 확인
			recRecordFile, _ := cmd.Flags().GetString("rec-record")
			recMetaInput, _ := cmd.Flags().GetString("rec-meta")

			var msg *types.MsgDepositCollateral

			// RECRecord 입력
			if recRecordFile != "" {
				var rec types.RECRecord
				if strings.HasPrefix(strings.TrimSpace(recRecordFile), "{") {
					// JSON 문자열 직접 전달
					if err := json.Unmarshal([]byte(recRecordFile), &rec); err != nil {
						return fmt.Errorf("failed to unmarshal rec-record JSON string: %w", err)
					}
				} else {
					// 파일 경로로 처리
					data, err := os.ReadFile(recRecordFile)
					if err != nil {
						return fmt.Errorf("failed to read rec-record file: %w", err)
					}
					if err := json.Unmarshal(data, &rec); err != nil {
						return fmt.Errorf("failed to unmarshal rec-record file: %w", err)
					}
				}

				msg = types.NewMsgDepositCollateralWithRecord(
					clientCtx.GetFromAddress().String(),
					rec,
				)
			}

			// RECMeta 입력
			if recMetaInput != "" {
				var meta types.RECMeta
				if strings.HasPrefix(strings.TrimSpace(recMetaInput), "{") {
					// JSON 문자열 직접 전달
					if err := json.Unmarshal([]byte(recMetaInput), &meta); err != nil {
						return fmt.Errorf("failed to unmarshal rec-meta JSON string: %w", err)
					}
				} else {
					// 파일 경로로 처리
					data, err := os.ReadFile(recMetaInput)
					if err != nil {
						return fmt.Errorf("failed to read rec-meta file: %w", err)
					}
					if err := json.Unmarshal(data, &meta); err != nil {
						return fmt.Errorf("failed to unmarshal rec-meta file: %w", err)
					}
				}

				msg = types.NewMsgDepositCollateralWithMeta(
					clientCtx.GetFromAddress().String(),
					meta,
				)
			}

			if msg == nil {
				return fmt.Errorf("either --rec-record or --rec-meta must be provided")
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	// 플래그 정의
	cmd.Flags().String("rec-record", "", "Path to JSON file containing a RECRecord")
	cmd.Flags().String("rec-meta", "", "Path to JSON file containing a RECMeta")

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdRemoveCollateral() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove-collateral [amount]",
		Short: "Remove collateral from the reward module (reduce total stake)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			amount := args[0]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgRemoveCollateral(
				clientCtx.GetFromAddress().String(),
				amount,
			)

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdBurnModuleStable() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "burn-module-stable [amount]",
		Short: "소각: 모듈 계좌의 stable 코인을 지정한 수량만큼 소각",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			amount := args[0]

			msg := &types.MsgBurnModuleStable{
				Creator: clientCtx.GetFromAddress().String(),
				Amount:  amount,
			}

			if err := msg.ValidateBasic(); err != nil {
				return fmt.Errorf("메시지 유효성 검증 실패: %w", err)
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// AddEnergyCmd : 발전량 기록 및 REC 계산
func CmdAddEnergy() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-energy [to] [amountWh] [txHash]",
		Short: "기록: 발전자 주소에 amountWh(Wh)를 추가하고 REC 환산",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			to := args[0]
			amount := args[1]
			txHash := args[2]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgAddEnergy(
				clientCtx.GetFromAddress().String(), // creator
				to,
				amount,
				txHash,
			)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CreateRECRecordCmd : REC 발급
func CmdCreateRECRecord() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-rec-record [count]",
		Short: "발급: count 개수의 RECRecord 생성",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			count, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid count: %w", err)
			}

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgCreateRECRecord(
				clientCtx.GetFromAddress().String(), // creator
				count,
			)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdAppendTxHash() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "append-tx-hash [node_creator] [rec_id]",
		Short: "TxNode 값을 LinkedList에 기록 (항상 --from alice)",
		Args:  cobra.ExactArgs(2), // ✅ rec_id까지 2개 인자 받음
		RunE: func(cmd *cobra.Command, args []string) error {
			nodeCreator := args[0]
			recID := args[1]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			// ✅ hashes는 빈 배열로 전달 → Keeper 내부에서 KVStore 조회
			msg := types.NewMsgAppendTxHash(
				clientCtx.GetFromAddress().String(), // signer (항상 alice)
				[]string{},                          // hashes는 내부 조회
				nodeCreator,                         // 노드 생성자
				recID,                               // ✅ REC ID 추가
				"",                                  // next 값 (초기에는 빈 값)
			)

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdDistributeRewardPercent() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "distribute-reward-percent [address] [percent]",
		Short: "Distribute a percentage of the reward module's balance to an address",
		Long: `Distribute a percentage of the reward module account balance 
to a specified address without minting new coins.
The [percent] argument should be a decimal value between 0 and 1.
Example: distribute-reward-percent cosmos1abcd... 0.25`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			address := args[0]
			percent := args[1]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgDistributeRewardPercent(
				clientCtx.GetFromAddress().String(), // 보낸 사람(creator)
				address,                             // 받는 사람
				percent,                             // 비율
			)

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Reward transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	cmd.AddCommand(CmdRewardSolarPower())
	// 🔥 새로 추가한 담보 예치 명령 등록
	cmd.AddCommand(CmdDepositCollateral())
	// 🔥 소각 명령도 등록
	cmd.AddCommand(CmdBurnStableCoin())
	cmd.AddCommand(CmdRemoveCollateral())
	cmd.AddCommand(CmdBurnModuleStable())
	cmd.AddCommand(CmdAddEnergy())
	cmd.AddCommand(CmdCreateRECRecord())
	cmd.AddCommand(CmdAppendTxHash())
	cmd.AddCommand(CmdDistributeRewardPercent())
	return cmd
}
