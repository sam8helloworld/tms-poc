package cmd

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sam8helloworld/tms-poc/internal/sourcing/application/bid"
	"github.com/spf13/cobra"
)

var contractCmd = &cobra.Command{
	Use:   "contract",
	Short: "Manage service contracts",
}

var contractCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new DRAFT bid contract",
	RunE: func(cmd *cobra.Command, args []string) error {
		providerID, _ := cmd.Flags().GetString("provider-id")
		shipperID, _ := cmd.Flags().GetString("shipper-id")
		bidRequestID, _ := cmd.Flags().GetString("bid-request-id")
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

		input := bid.CreateBidContractInput{
			BidRequestID: uuid.MustParse(bidRequestID),
			ProviderID:   uuid.MustParse(providerID),
			ShipperID:    uuid.MustParse(shipperID),
			ValidFrom:    from,
			ValidTo:      to,
		}

		output, err := deps.CreateBidContractUC.Execute(context.Background(), input)
		if err != nil {
			return err
		}
		return printJSON(output)
	},
}

var contractDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete (cancel) a DRAFT contract",
	RunE: func(cmd *cobra.Command, args []string) error {
		contractID, _ := cmd.Flags().GetString("contract-id")

		input := bid.DeleteBidContractInput{
			ContractID: uuid.MustParse(contractID),
		}

		output, err := deps.DeleteBidContractUC.Execute(context.Background(), input)
		if err != nil {
			return err
		}
		return printJSON(output)
	},
}

var contractUpdatePeriodCmd = &cobra.Command{
	Use:   "update-period",
	Short: "Update the valid period of a DRAFT contract",
	RunE: func(cmd *cobra.Command, args []string) error {
		contractID, _ := cmd.Flags().GetString("contract-id")
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

		input := bid.UpdateContractPeriodInput{
			ContractID: uuid.MustParse(contractID),
			ValidFrom:  from,
			ValidTo:    to,
		}

		output, err := deps.UpdateContractPeriodUC.Execute(context.Background(), input)
		if err != nil {
			return err
		}
		return printJSON(output)
	},
}

var contractAwardCmd = &cobra.Command{
	Use:   "award",
	Short: "Award a DRAFT contract (DRAFT → CONTRACTED)",
	RunE: func(cmd *cobra.Command, args []string) error {
		contractID, _ := cmd.Flags().GetString("contract-id")

		input := bid.AwardBidContractInput{
			ContractID: uuid.MustParse(contractID),
		}

		output, err := deps.AwardBidContractUC.Execute(context.Background(), input)
		if err != nil {
			return err
		}
		return printJSON(output)
	},
}

var contractGetCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Get a contract by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := uuid.MustParse(args[0])
		result, err := deps.SourcingQuery.GetContract(context.Background(), id)
		if err != nil {
			return err
		}
		return printJSON(result)
	},
}

var contractListCmd = &cobra.Command{
	Use:   "list",
	Short: "List contracts by shipper ID",
	RunE: func(cmd *cobra.Command, args []string) error {
		shipperID, _ := cmd.Flags().GetString("shipper-id")
		result, err := deps.SourcingQuery.ListContractsByShipper(context.Background(), uuid.MustParse(shipperID))
		if err != nil {
			return err
		}
		return printJSON(result)
	},
}

func init() {
	contractCreateCmd.Flags().String("provider-id", "", "Provider UUID")
	contractCreateCmd.Flags().String("shipper-id", "", "Shipper UUID")
	contractCreateCmd.Flags().String("bid-request-id", "", "Bid request UUID")
	contractCreateCmd.Flags().String("valid-from", "", "Valid from date (YYYY-MM-DD)")
	contractCreateCmd.Flags().String("valid-to", "", "Valid to date (YYYY-MM-DD)")
	contractCreateCmd.MarkFlagRequired("provider-id")
	contractCreateCmd.MarkFlagRequired("shipper-id")
	contractCreateCmd.MarkFlagRequired("bid-request-id")
	contractCreateCmd.MarkFlagRequired("valid-from")
	contractCreateCmd.MarkFlagRequired("valid-to")

	contractDeleteCmd.Flags().String("contract-id", "", "Contract UUID")
	contractDeleteCmd.MarkFlagRequired("contract-id")

	contractUpdatePeriodCmd.Flags().String("contract-id", "", "Contract UUID")
	contractUpdatePeriodCmd.Flags().String("valid-from", "", "New valid from date (YYYY-MM-DD)")
	contractUpdatePeriodCmd.Flags().String("valid-to", "", "New valid to date (YYYY-MM-DD)")
	contractUpdatePeriodCmd.MarkFlagRequired("contract-id")
	contractUpdatePeriodCmd.MarkFlagRequired("valid-from")
	contractUpdatePeriodCmd.MarkFlagRequired("valid-to")

	contractListCmd.Flags().String("shipper-id", "", "Shipper UUID")
	contractListCmd.MarkFlagRequired("shipper-id")

	contractAwardCmd.Flags().String("contract-id", "", "Contract UUID")
	contractAwardCmd.MarkFlagRequired("contract-id")

	contractCmd.AddCommand(contractCreateCmd)
	contractCmd.AddCommand(contractDeleteCmd)
	contractCmd.AddCommand(contractUpdatePeriodCmd)
	contractCmd.AddCommand(contractAwardCmd)
	contractCmd.AddCommand(contractGetCmd)
	contractCmd.AddCommand(contractListCmd)
}
