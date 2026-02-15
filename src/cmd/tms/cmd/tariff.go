package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/sourcing/domain/pricing"
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

var tariffRegisterDirectCmd = &cobra.Command{
	Use:   "register-direct",
	Short: "Register a tariff from structured data (JSON file)",
	Long:  "Register a tariff directly from a JSON file containing structured line item data",
	RunE: func(cmd *cobra.Command, args []string) error {
		contractID, _ := cmd.Flags().GetString("contract-id")
		tariffName, _ := cmd.Flags().GetString("name")
		validFrom, _ := cmd.Flags().GetString("valid-from")
		validTo, _ := cmd.Flags().GetString("valid-to")
		dataFile, _ := cmd.Flags().GetString("data-file")

		from, err := time.Parse(time.DateOnly, validFrom)
		if err != nil {
			return fmt.Errorf("invalid valid-from date: %w", err)
		}
		to, err := time.Parse(time.DateOnly, validTo)
		if err != nil {
			return fmt.Errorf("invalid valid-to date: %w", err)
		}

		// JSONファイルからLineItemsを読み込み
		var lineItems []tariffapp.LineItemInput
		if dataFile != "" {
			data, err := os.ReadFile(dataFile)
			if err != nil {
				return fmt.Errorf("failed to read data file: %w", err)
			}
			var rawItems []struct {
				ChargeCode   string            `json:"charge_code"`
				Category     string            `json:"category"`
				ScopeType    string            `json:"scope_type"`
				ScopeAttrs   map[string]string `json:"scope_attrs"`
				PricingType  string            `json:"pricing_type"`
				PricingAttrs map[string]any    `json:"pricing_attrs"`
			}
			if err := json.Unmarshal(data, &rawItems); err != nil {
				return fmt.Errorf("failed to parse data file: %w", err)
			}
			for _, raw := range rawItems {
				lineItems = append(lineItems, tariffapp.LineItemInput{
					ChargeCode:   raw.ChargeCode,
					Category:     raw.Category,
					ScopeType:    pricing.ServiceScopeType(raw.ScopeType),
					ScopeAttrs:   raw.ScopeAttrs,
					PricingType:  pricing.PricingStrategyType(raw.PricingType),
					PricingAttrs: raw.PricingAttrs,
				})
			}
		}

		input := tariffapp.RegisterTariffDirectInput{
			ContractID:    uuid.MustParse(contractID),
			TariffName:    tariffName,
			EffectiveFrom: from,
			EffectiveTo:   to,
			LineItems:     lineItems,
		}

		output, err := deps.RegisterTariffDirectUC.Execute(context.Background(), input)
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

	tariffRegisterDirectCmd.Flags().String("contract-id", "", "Contract UUID")
	tariffRegisterDirectCmd.Flags().String("name", "", "Tariff name")
	tariffRegisterDirectCmd.Flags().String("valid-from", "", "Effective from date (YYYY-MM-DD)")
	tariffRegisterDirectCmd.Flags().String("valid-to", "", "Effective to date (YYYY-MM-DD)")
	tariffRegisterDirectCmd.Flags().String("data-file", "", "JSON file with line items")
	tariffRegisterDirectCmd.MarkFlagRequired("contract-id")
	tariffRegisterDirectCmd.MarkFlagRequired("name")
	tariffRegisterDirectCmd.MarkFlagRequired("valid-from")
	tariffRegisterDirectCmd.MarkFlagRequired("valid-to")

	tariffCmd.AddCommand(tariffGetCmd)
	tariffCmd.AddCommand(tariffListCmd)
	tariffCmd.AddCommand(tariffRemoveCmd)
	tariffCmd.AddCommand(tariffRegisterDirectCmd)
}
