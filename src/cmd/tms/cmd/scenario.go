package cmd

import (
	"context"
	"fmt"

	"github.com/sam8helloworld/tms-poc/cmd/tms/cmd/scenarios"
	"github.com/spf13/cobra"
)

// scenarioRegistry: 利用可能なシナリオの一覧
var scenarioRegistry = []scenarios.Scenario{
	&scenarios.SourcingBidScenario{},
	&scenarios.RateBafUpdateScenario{},
	&scenarios.RateSimulationScenario{},
	&scenarios.ShipmentDocumentScenario{},
	&scenarios.CsvTariffParseScenario{},
}

var scenarioCmd = &cobra.Command{
	Use:   "scenario",
	Short: "Run POC scenarios",
}

var scenarioListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available scenarios",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Available scenarios:")
		fmt.Println()
		for _, s := range scenarioRegistry {
			fmt.Printf("  %-20s %s\n", s.Name(), s.Description())
		}
		return nil
	},
}

var scenarioRunCmd = &cobra.Command{
	Use:   "run [scenario-name]",
	Short: "Run a scenario by name",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		scenarioDeps := &scenarios.ScenarioDeps{
			CreateBidContractUC:     deps.CreateBidContractUC,
			AwardBidContractUC:      deps.AwardBidContractUC,
			RegisterTariffDirectUC:  deps.RegisterTariffDirectUC,
			AmendTariffDirectUC:     deps.AmendTariffDirectUC,
			CreateRateUC:            deps.CreateRateUC,
			ActivateRateUC:          deps.ActivateRateUC,
			ApplyContractToRateUC:   deps.ApplyContractToRateUC,
			UpdateRateEntryTariffUC: deps.UpdateRateEntryTariffUC,
			SimulateRateCostUC:      deps.SimulateRateCostUC,
			CreateShipmentUC:        deps.CreateShipmentUC,
			RecordMilestoneUC:       deps.RecordMilestoneUC,
			UploadDocumentUC:        deps.UploadDocumentUC,
			ConfirmDocumentUC:       deps.ConfirmDocumentUC,
			NetworkQuery:            deps.NetworkQuery,
			SourcingQuery:           deps.SourcingQuery,
			RateQuery:               deps.RateQuery,
			ShipmentQuery:           deps.ShipmentQuery,
			DocumentQuery:           deps.DocumentQuery,
		}

		for _, s := range scenarioRegistry {
			if s.Name() == name {
				return s.Run(context.Background(), scenarioDeps, pool)
			}
		}

		return fmt.Errorf("unknown scenario: %s (use 'tms scenario list' to see available scenarios)", name)
	},
}

func init() {
	scenarioCmd.AddCommand(scenarioListCmd)
	scenarioCmd.AddCommand(scenarioRunCmd)
}
