package cmd

import (
	"context"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var eventCmd = &cobra.Command{
	Use:   "event",
	Short: "Query domain events",
}

var eventListCmd = &cobra.Command{
	Use:   "list",
	Short: "List domain events by aggregate",
	RunE: func(cmd *cobra.Command, args []string) error {
		aggregateID, _ := cmd.Flags().GetString("aggregate-id")
		aggregateType, _ := cmd.Flags().GetString("aggregate-type")

		result, err := deps.EventQuery.ListEventsByAggregate(
			context.Background(),
			uuid.MustParse(aggregateID),
			aggregateType,
		)
		if err != nil {
			return err
		}
		return printJSON(result)
	},
}

func init() {
	eventListCmd.Flags().String("aggregate-id", "", "Aggregate UUID")
	eventListCmd.Flags().String("aggregate-type", "", "Aggregate type (e.g. TrackingUnit, ServiceContract)")
	eventListCmd.MarkFlagRequired("aggregate-id")
	eventListCmd.MarkFlagRequired("aggregate-type")

	eventCmd.AddCommand(eventListCmd)
}
