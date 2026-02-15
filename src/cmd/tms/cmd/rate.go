package cmd

import (
	"context"
	"time"

	"github.com/google/uuid"
	rateapp "github.com/sam8helloworld/tms-poc/internal/rate/application/rate"
	"github.com/spf13/cobra"
)

var rateCmd = &cobra.Command{
	Use:   "rate",
	Short: "Manage rates",
}

var rateCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new DRAFT rate",
	RunE: func(cmd *cobra.Command, args []string) error {
		shipperID, _ := cmd.Flags().GetString("shipper-id")
		name, _ := cmd.Flags().GetString("name")
		validFrom, _ := cmd.Flags().GetString("valid-from")
		validTo, _ := cmd.Flags().GetString("valid-to")

		from, err := time.Parse(time.DateOnly, validFrom)
		if err != nil {
			return err
		}
		to, err := time.Parse(time.DateOnly, validTo)
		if err != nil {
			return err
		}

		input := rateapp.CreateRateInput{
			ShipperID: uuid.MustParse(shipperID),
			Name:      name,
			ValidFrom: from,
			ValidTo:   to,
		}

		output, err := deps.CreateRateUC.Execute(context.Background(), input)
		if err != nil {
			return err
		}
		return printJSON(output)
	},
}

var rateActivateCmd = &cobra.Command{
	Use:   "activate",
	Short: "Activate a DRAFT rate (DRAFT → ACTIVE)",
	RunE: func(cmd *cobra.Command, args []string) error {
		rateID, _ := cmd.Flags().GetString("rate-id")

		input := rateapp.ActivateRateInput{
			RateID: uuid.MustParse(rateID),
		}

		output, err := deps.ActivateRateUC.Execute(context.Background(), input)
		if err != nil {
			return err
		}
		return printJSON(output)
	},
}

var rateApplyContractCmd = &cobra.Command{
	Use:   "apply-contract",
	Short: "Apply a CONTRACTED contract's tariffs to a DRAFT rate",
	RunE: func(cmd *cobra.Command, args []string) error {
		rateID, _ := cmd.Flags().GetString("rate-id")
		contractID, _ := cmd.Flags().GetString("contract-id")

		input := rateapp.ApplyContractToRateInput{
			RateID:     uuid.MustParse(rateID),
			ContractID: uuid.MustParse(contractID),
		}

		output, err := deps.ApplyContractToRateUC.Execute(context.Background(), input)
		if err != nil {
			return err
		}
		return printJSON(output)
	},
}

var rateUpdateEntryTariffCmd = &cobra.Command{
	Use:   "update-entry-tariff",
	Short: "Replace a rate entry's tariff ID",
	RunE: func(cmd *cobra.Command, args []string) error {
		rateID, _ := cmd.Flags().GetString("rate-id")
		entryID, _ := cmd.Flags().GetString("entry-id")
		contractID, _ := cmd.Flags().GetString("contract-id")
		newTariffID, _ := cmd.Flags().GetString("new-tariff-id")

		input := rateapp.UpdateRateEntryTariffInput{
			RateID:      uuid.MustParse(rateID),
			EntryID:     uuid.MustParse(entryID),
			ContractID:  uuid.MustParse(contractID),
			NewTariffID: uuid.MustParse(newTariffID),
		}

		output, err := deps.UpdateRateEntryTariffUC.Execute(context.Background(), input)
		if err != nil {
			return err
		}
		return printJSON(output)
	},
}

var rateGetCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Get a rate by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := uuid.MustParse(args[0])
		result, err := deps.RateQuery.GetRate(context.Background(), id)
		if err != nil {
			return err
		}
		return printJSON(result)
	},
}

var rateListCmd = &cobra.Command{
	Use:   "list",
	Short: "List rates by shipper ID",
	RunE: func(cmd *cobra.Command, args []string) error {
		shipperID, _ := cmd.Flags().GetString("shipper-id")
		result, err := deps.RateQuery.ListRatesByShipper(context.Background(), uuid.MustParse(shipperID))
		if err != nil {
			return err
		}
		return printJSON(result)
	},
}

func init() {
	rateApplyContractCmd.Flags().String("rate-id", "", "Rate UUID")
	rateApplyContractCmd.Flags().String("contract-id", "", "Contract UUID")
	rateApplyContractCmd.MarkFlagRequired("rate-id")
	rateApplyContractCmd.MarkFlagRequired("contract-id")

	rateUpdateEntryTariffCmd.Flags().String("rate-id", "", "Rate UUID")
	rateUpdateEntryTariffCmd.Flags().String("entry-id", "", "Entry UUID")
	rateUpdateEntryTariffCmd.Flags().String("contract-id", "", "Contract UUID")
	rateUpdateEntryTariffCmd.Flags().String("new-tariff-id", "", "New Tariff UUID")
	rateUpdateEntryTariffCmd.MarkFlagRequired("rate-id")
	rateUpdateEntryTariffCmd.MarkFlagRequired("entry-id")
	rateUpdateEntryTariffCmd.MarkFlagRequired("contract-id")
	rateUpdateEntryTariffCmd.MarkFlagRequired("new-tariff-id")

	rateListCmd.Flags().String("shipper-id", "", "Shipper UUID")
	rateListCmd.MarkFlagRequired("shipper-id")

	rateCreateCmd.Flags().String("shipper-id", "", "Shipper UUID")
	rateCreateCmd.Flags().String("name", "", "Rate name")
	rateCreateCmd.Flags().String("valid-from", "", "Valid from date (YYYY-MM-DD)")
	rateCreateCmd.Flags().String("valid-to", "", "Valid to date (YYYY-MM-DD)")
	rateCreateCmd.MarkFlagRequired("shipper-id")
	rateCreateCmd.MarkFlagRequired("name")
	rateCreateCmd.MarkFlagRequired("valid-from")
	rateCreateCmd.MarkFlagRequired("valid-to")

	rateActivateCmd.Flags().String("rate-id", "", "Rate UUID")
	rateActivateCmd.MarkFlagRequired("rate-id")

	rateCmd.AddCommand(rateCreateCmd)
	rateCmd.AddCommand(rateActivateCmd)
	rateCmd.AddCommand(rateApplyContractCmd)
	rateCmd.AddCommand(rateUpdateEntryTariffCmd)
	rateCmd.AddCommand(rateGetCmd)
	rateCmd.AddCommand(rateListCmd)
}
