package cmd

import (
	"context"

	"github.com/google/uuid"
	trackingapp "github.com/sam8helloworld/tms-poc/internal/tracking/application/tracking"
	trackingdomain "github.com/sam8helloworld/tms-poc/internal/tracking/domain/tracking"
	"github.com/sam8helloworld/tms-poc/internal/shared"
	"github.com/spf13/cobra"
)

var trackingCmd = &cobra.Command{
	Use:   "tracking",
	Short: "Manage tracking units",
}

var trackingRegisterCmd = &cobra.Command{
	Use:   "register",
	Short: "Register a new tracking unit for a shipment",
	RunE: func(cmd *cobra.Command, args []string) error {
		shipmentID, _ := cmd.Flags().GetString("shipment-id")
		trackingNumber, _ := cmd.Flags().GetString("tracking-number")
		numberType, _ := cmd.Flags().GetString("number-type")
		carrierID, _ := cmd.Flags().GetString("carrier-id")
		originID, _ := cmd.Flags().GetString("origin-id")
		destID, _ := cmd.Flags().GetString("dest-id")
		mode, _ := cmd.Flags().GetString("mode")

		input := trackingapp.RegisterShipmentTrackingInput{
			ShipmentID:         uuid.MustParse(shipmentID),
			TrackingNumber:     trackingNumber,
			TrackingNumberType: trackingdomain.TrackingNumberType(numberType),
			CarrierID:          uuid.MustParse(carrierID),
			Segments: []trackingapp.SegmentInput{
				{
					ActualOriginLocationID: uuid.MustParse(originID),
					ActualDestLocationID:   uuid.MustParse(destID),
					Mode:                   shared.TransportMode(mode),
					PrimarySource:          trackingdomain.TrackingSourceType("MANUAL_INPUT"),
				},
			},
		}

		output, err := deps.RegisterTrackingUC.Execute(context.Background(), input)
		if err != nil {
			return err
		}
		return printJSON(output)
	},
}

var trackingSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync tracking events from external providers",
	RunE: func(cmd *cobra.Command, args []string) error {
		trackingUnitID, _ := cmd.Flags().GetString("tracking-unit-id")

		input := trackingapp.SyncTrackingInput{
			TrackingUnitID: uuid.MustParse(trackingUnitID),
		}

		output, err := deps.SyncTrackingUC.Execute(context.Background(), input)
		if err != nil {
			return err
		}
		return printJSON(output)
	},
}

var trackingGetCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Get a tracking unit by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := uuid.MustParse(args[0])
		result, err := deps.TrackingQuery.GetTrackingUnit(context.Background(), id)
		if err != nil {
			return err
		}
		return printJSON(result)
	},
}

var trackingGetByNumberCmd = &cobra.Command{
	Use:   "get-by-number [number]",
	Short: "Get a tracking unit by tracking number",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		result, err := deps.TrackingQuery.GetTrackingUnitByNumber(context.Background(), args[0])
		if err != nil {
			return err
		}
		return printJSON(result)
	},
}

func init() {
	trackingRegisterCmd.Flags().String("shipment-id", "", "Shipment UUID")
	trackingRegisterCmd.Flags().String("tracking-number", "", "Tracking number")
	trackingRegisterCmd.Flags().String("number-type", "CONTAINER", "Tracking number type")
	trackingRegisterCmd.Flags().String("carrier-id", "", "Carrier UUID")
	trackingRegisterCmd.Flags().String("origin-id", "", "Origin location UUID")
	trackingRegisterCmd.Flags().String("dest-id", "", "Destination location UUID")
	trackingRegisterCmd.Flags().String("mode", "OCEAN", "Transport mode")
	trackingRegisterCmd.MarkFlagRequired("shipment-id")
	trackingRegisterCmd.MarkFlagRequired("tracking-number")
	trackingRegisterCmd.MarkFlagRequired("carrier-id")
	trackingRegisterCmd.MarkFlagRequired("origin-id")
	trackingRegisterCmd.MarkFlagRequired("dest-id")

	trackingSyncCmd.Flags().String("tracking-unit-id", "", "Tracking unit UUID")
	trackingSyncCmd.MarkFlagRequired("tracking-unit-id")

	trackingCmd.AddCommand(trackingRegisterCmd)
	trackingCmd.AddCommand(trackingSyncCmd)
	trackingCmd.AddCommand(trackingGetCmd)
	trackingCmd.AddCommand(trackingGetByNumberCmd)
}
