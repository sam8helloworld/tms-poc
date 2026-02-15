package cmd

import (
	"context"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var locationCmd = &cobra.Command{
	Use:   "location",
	Short: "Query locations",
}

var locationGetCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Get a location by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := uuid.MustParse(args[0])
		result, err := deps.NetworkQuery.GetLocation(context.Background(), id)
		if err != nil {
			return err
		}
		return printJSON(result)
	},
}

var locationGetByUNLocodeCmd = &cobra.Command{
	Use:   "get-by-unlocode [code]",
	Short: "Get a location by UN/LOCODE",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		result, err := deps.NetworkQuery.GetLocationByUNLocode(context.Background(), args[0])
		if err != nil {
			return err
		}
		return printJSON(result)
	},
}

func init() {
	locationCmd.AddCommand(locationGetCmd)
	locationCmd.AddCommand(locationGetByUNLocodeCmd)
}
