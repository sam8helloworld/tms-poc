package cmd

import (
	"context"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var shipmentCmd = &cobra.Command{
	Use:   "shipment",
	Short: "Query shipments",
}

var shipmentGetCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Get a shipment by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := uuid.MustParse(args[0])
		result, err := deps.ShipmentQuery.GetShipment(context.Background(), id)
		if err != nil {
			return err
		}
		return printJSON(result)
	},
}

var shipmentGetByNoCmd = &cobra.Command{
	Use:   "get-by-no [no]",
	Short: "Get a shipment by shipment number",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		result, err := deps.ShipmentQuery.GetShipmentByNo(context.Background(), args[0])
		if err != nil {
			return err
		}
		return printJSON(result)
	},
}

func init() {
	shipmentCmd.AddCommand(shipmentGetCmd)
	shipmentCmd.AddCommand(shipmentGetByNoCmd)
}
