package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Database management commands",
}

var dbResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Truncate all data tables (keeps schema)",
	RunE: func(cmd *cobra.Command, args []string) error {
		confirm, _ := cmd.Flags().GetBool("confirm")
		if !confirm {
			fmt.Println("This will delete ALL data from all tables.")
			fmt.Println("Use --confirm to proceed.")
			return nil
		}

		// domain_events は他テーブルから参照されないが先にTRUNCATEして安全に
		tables := []string{
			"domain_events",
			"document_reviews",
			"documents",
			"sop_tasks",
			"sop_instances",
			"sop_step_definitions",
			"sop_definitions",
			"tracking_events",
			"tracking_segments",
			"tracking_units",
			"service_operators",
			"shipment_milestones",
			"shipment_cost_summaries",
			"shipment_cost_line_items",
			"shipment_tracking_units",
			"shipment_items",
			"shipment_segments",
			"shipments",
			"rate_entries",
			"rates",
			"logistics_resources",
			"tariff_line_items",
			"tariffs",
			"service_contracts",
			"vendors",
			"standard_route_legs",
			"standard_routes",
			"lanes",
			"locations",
		}

		ctx := context.Background()
		for _, table := range tables {
			_, err := pool.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
			if err != nil {
				return fmt.Errorf("failed to truncate %s: %w", table, err)
			}
		}

		fmt.Println("All tables truncated successfully.")
		return nil
	},
}

func init() {
	dbResetCmd.Flags().Bool("confirm", false, "Confirm the reset operation")
	dbCmd.AddCommand(dbResetCmd)
}
