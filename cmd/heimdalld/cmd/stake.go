package heimdalld

import (

	// TODO HV2 - uncomment when we have FetchFromAPI uncommented in helper
	// chainmanagertypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"

	"github.com/spf13/cobra"
)

const chainManagerEndpoint = "/chainmanager/params"

// StakeCmd stakes for a validator
func StakeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stake",
		Short: "Stake polygon pos tokens for your account",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			// TODO HV2 - uncomment when we have staking
			/*
				helper.InitHeimdallConfig("")

				validatorStr := viper.GetString(stakingcli.FlagValidatorAddress)
				stakeAmountStr := viper.GetString(stakingcli.FlagAmount)
				feeAmountStr := viper.GetString(stakingcli.FlagFeeAmount)
				acceptDelegation := viper.GetBool(stakingcli.FlagAcceptDelegation)

				// validator str
				if validatorStr == "" {
					return errors.New("Validator address is required")
				}

				// stake amount
				stakeAmount, ok := big.NewInt(0).SetString(stakeAmountStr, 10)
				if !ok {
					return errors.New("Invalid stake amount")
				}

				// fee amount
				feeAmount, ok := big.NewInt(0).SetString(feeAmountStr, 10)
				if !ok {
					return errors.New("Invalid fee amount")
				}

				// contract caller
				contractCaller, err := helper.NewContractCaller()
				if err != nil {
					return err
				}

				params, err := GetChainManagerParams(cliCtx)
				if err != nil {
					return err
				}

				stakingManagerAddress := params.ChainParams.StakingManagerAddress.EthAddress()
				stakeManagerInstance, err := contractCaller.GetStakeManagerInstance(stakingManagerAddress)
				if err != nil {
					return err
				}

				return contractCaller.StakeFor(
					common.HexToAddress(validatorStr),
					stakeAmount,
					feeAmount,
					acceptDelegation,
					stakingManagerAddress,
					stakeManagerInstance,
				)
			*/
			return nil
		},
	}

	// TODO HV2 - uncomment when we have staking
	/*
		cmd.Flags().String(stakingcli.FlagValidatorAddress, "", "--validator=<validator address here>")
		cmd.Flags().String(stakingcli.FlagAmount, "10000000000000000000", "--staked-amount=<stake amount>, if left blank it will be assigned as 10 matic tokens")
		cmd.Flags().String(stakingcli.FlagFeeAmount, "5000000000000000000", "--fee-amount=<heimdall fee amount>, if left blank will be assigned as 5 matic tokens")
		cmd.Flags().Bool(stakingcli.FlagAcceptDelegation, true, "--accept-delegation=<accept delegation>, if left blank will be assigned as true")
	*/

	return cmd
}

// ApproveCmd approves tokens for a validator
func ApproveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "approve",
		Short: "Approve the tokens to stake",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			// TODO HV2 - uncomment when we have staking
			/*
				helper.InitHeimdallConfig("")

				stakeAmountStr := viper.GetString(stakingcli.FlagAmount)
				feeAmountStr := viper.GetString(stakingcli.FlagFeeAmount)

				// stake amount
				stakeAmount, ok := big.NewInt(0).SetString(stakeAmountStr, 10)
				if !ok {
					return errors.New("Invalid stake amount")
				}

				// fee amount
				feeAmount, ok := big.NewInt(0).SetString(feeAmountStr, 10)
				if !ok {
					return errors.New("Invalid fee amount")
				}

				contractCaller, err := helper.NewContractCaller()
				if err != nil {
					return err
				}

				params, err := GetChainManagerParams(cliCtx)
				if err != nil {
					return err
				}

				stakingManagerAddress := params.ChainParams.StakingManagerAddress.EthAddress()
				maticTokenAddress := params.ChainParams.PolTokenAddress.EthAddress()

				maticTokenInstance, err := contractCaller.GetPolygonPosTokenInstance(maticTokenAddress)
				if err != nil {
					return err
				}

				return contractCaller.ApproveTokens(stakeAmount.Add(stakeAmount, feeAmount), stakingManagerAddress, maticTokenAddress, maticTokenInstance)
			*/
			return nil
		},
	}

	// TODO HV2 - uncomment when we have staking
	/*
		cmd.Flags().String(stakingcli.FlagAmount, "10000000000000000000", "--staked-amount=<stake amount>, if left blank will be assigned as 10 matic tokens")
		cmd.Flags().String(stakingcli.FlagFeeAmount, "5000000000000000000", "--fee-amount=<heimdall fee amount>, if left blank will be assigned as 5 matic tokens")
	*/

	return cmd
}

// TODO HV2 - uncomment when we have FetchFromAPI uncommented in helper
/*
// GetChainManagerParams return configManager params
func GetChainManagerParams(cliCtx client.Context) (*chainmanagertypes.Params, error) {
	response, err := helper.FetchFromAPI(
		cliCtx,
		helper.GetHeimdallServerEndpoint(chainmanagerEndpoint),
	)

	if err != nil {
		return nil, err
	}

	var params chainmanagertypes.Params
	if err := json.Unmarshal(response.Result, &params); err != nil {
		return nil, err
	}

	return &params, nil
}
*/
