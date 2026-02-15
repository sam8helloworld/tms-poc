package cmd

import (
	"context"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var sopInstanceCmd = &cobra.Command{
	Use:   "sop-instance",
	Short: "Query SOP instances",
}

var sopInstanceGetCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Get an SOP instance by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := uuid.MustParse(args[0])
		result, err := deps.OperationQuery.GetSOPInstance(context.Background(), id)
		if err != nil {
			return err
		}
		return printJSON(result)
	},
}

var sopInstanceGetByShipmentCmd = &cobra.Command{
	Use:   "get-by-shipment [shipment-id]",
	Short: "Get an SOP instance by shipment ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := uuid.MustParse(args[0])
		result, err := deps.OperationQuery.GetSOPInstanceByShipment(context.Background(), id)
		if err != nil {
			return err
		}
		return printJSON(result)
	},
}

func init() {
	sopInstanceCmd.AddCommand(sopInstanceGetCmd)
	sopInstanceCmd.AddCommand(sopInstanceGetByShipmentCmd)
}
