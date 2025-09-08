package cli

import (
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
		Use:   "deposit-collateral [amount]",
		Short: "Deposit collateral tokens into the reward module",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			amount := args[0]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgDepositCollateral(
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
	return cmd
}
