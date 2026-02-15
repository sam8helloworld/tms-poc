package cmd

import (
	"context"

	"github.com/google/uuid"
	tariffapp "github.com/sam8helloworld/tms-poc/internal/sourcing/application/tariff"
	"github.com/spf13/cobra"
)

var tariffCmd = &cobra.Command{
	Use:   "tariff",
	Short: "Manage tariffs",
}

var tariffGetCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Get a tariff by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := uuid.MustParse(args[0])
		result, err := deps.SourcingQuery.GetTariff(context.Background(), id)
		if err != nil {
			return err
		}
		return printJSON(result)
	},
}

var tariffListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tariffs by contract ID",
	RunE: func(cmd *cobra.Command, args []string) error {
		contractID, _ := cmd.Flags().GetString("contract-id")
		result, err := deps.SourcingQuery.ListTariffsByContract(context.Background(), uuid.MustParse(contractID))
		if err != nil {
			return err
		}
		return printJSON(result)
	},
}

var tariffRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a tariff from a DRAFT contract",
	RunE: func(cmd *cobra.Command, args []string) error {
		contractID, _ := cmd.Flags().GetString("contract-id")
		tariffID, _ := cmd.Flags().GetString("tariff-id")

		input := tariffapp.RemoveTariffInput{
			ContractID: uuid.MustParse(contractID),
			TariffID:   uuid.MustParse(tariffID),
		}

		output, err := deps.RemoveTariffUC.Execute(context.Background(), input)
		if err != nil {
			return err
		}
		return printJSON(output)
	},
}

func init() {
	tariffListCmd.Flags().String("contract-id", "", "Contract UUID")
	tariffListCmd.MarkFlagRequired("contract-id")

	tariffRemoveCmd.Flags().String("contract-id", "", "Contract UUID")
	tariffRemoveCmd.Flags().String("tariff-id", "", "Tariff UUID")
	tariffRemoveCmd.MarkFlagRequired("contract-id")
	tariffRemoveCmd.MarkFlagRequired("tariff-id")

	tariffCmd.AddCommand(tariffGetCmd)
	tariffCmd.AddCommand(tariffListCmd)
	tariffCmd.AddCommand(tariffRemoveCmd)
}
