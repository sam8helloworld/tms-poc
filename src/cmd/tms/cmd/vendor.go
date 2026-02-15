package cmd

import (
	"context"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var vendorCmd = &cobra.Command{
	Use:   "vendor",
	Short: "Query vendors",
}

var vendorGetCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Get a vendor by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := uuid.MustParse(args[0])
		result, err := deps.SourcingQuery.GetVendor(context.Background(), id)
		if err != nil {
			return err
		}
		return printJSON(result)
	},
}

func init() {
	vendorCmd.AddCommand(vendorGetCmd)
}
