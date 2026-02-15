package cmd

import (
	"context"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var sopDefinitionCmd = &cobra.Command{
	Use:   "sop-definition",
	Short: "Query SOP definitions",
}

var sopDefinitionGetCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Get an SOP definition by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := uuid.MustParse(args[0])
		result, err := deps.OperationQuery.GetSOPDefinition(context.Background(), id)
		if err != nil {
			return err
		}
		return printJSON(result)
	},
}

var sopDefinitionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List active SOP definitions",
	RunE: func(cmd *cobra.Command, args []string) error {
		result, err := deps.OperationQuery.ListSOPDefinitions(context.Background())
		if err != nil {
			return err
		}
		return printJSON(result)
	},
}

func init() {
	sopDefinitionCmd.AddCommand(sopDefinitionGetCmd)
	sopDefinitionCmd.AddCommand(sopDefinitionListCmd)
}
