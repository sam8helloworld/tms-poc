package cmd

import (
	"context"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var standardRouteCmd = &cobra.Command{
	Use:   "standard-route",
	Short: "Query standard routes",
}

var standardRouteGetCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Get a standard route by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := uuid.MustParse(args[0])
		result, err := deps.NetworkQuery.GetStandardRoute(context.Background(), id)
		if err != nil {
			return err
		}
		return printJSON(result)
	},
}

var standardRouteListCmd = &cobra.Command{
	Use:   "list",
	Short: "List active standard routes by shipper ID",
	RunE: func(cmd *cobra.Command, args []string) error {
		shipperID, _ := cmd.Flags().GetString("shipper-id")
		result, err := deps.NetworkQuery.ListStandardRoutes(context.Background(), uuid.MustParse(shipperID))
		if err != nil {
			return err
		}
		return printJSON(result)
	},
}

func init() {
	standardRouteListCmd.Flags().String("shipper-id", "", "Shipper UUID")
	standardRouteListCmd.MarkFlagRequired("shipper-id")

	standardRouteCmd.AddCommand(standardRouteGetCmd)
	standardRouteCmd.AddCommand(standardRouteListCmd)
}
