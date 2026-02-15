package cmd

import (
	"context"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var laneCmd = &cobra.Command{
	Use:   "lane",
	Short: "Query lanes",
}

var laneGetCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Get a lane by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := uuid.MustParse(args[0])
		result, err := deps.NetworkQuery.GetLane(context.Background(), id)
		if err != nil {
			return err
		}
		return printJSON(result)
	},
}

func init() {
	laneCmd.AddCommand(laneGetCmd)
}
