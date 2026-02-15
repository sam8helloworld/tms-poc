package cmd

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sam8helloworld/tms-poc/pkg/postgres"
	"github.com/spf13/cobra"
)

var (
	dbHost     string
	dbPort     int
	dbUser     string
	dbPassword string
	dbName     string

	pool *pgxpool.Pool
	deps *Dependencies
)

var rootCmd = &cobra.Command{
	Use:   "tms",
	Short: "TMS CLI - International Logistics SCM Platform",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		cfg := postgres.Config{
			Host:     dbHost,
			Port:     dbPort,
			User:     dbUser,
			Password: dbPassword,
			Database: dbName,
			SSLMode:  "disable",
		}

		var err error
		pool, err = postgres.NewPool(context.Background(), cfg)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}

		deps = NewDependencies(pool)
		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if pool != nil {
			pool.Close()
		}
	},
}

func init() {
	defaults := postgres.DefaultConfig()

	rootCmd.PersistentFlags().StringVar(&dbHost, "db-host", defaults.Host, "Database host")
	rootCmd.PersistentFlags().IntVar(&dbPort, "db-port", defaults.Port, "Database port")
	rootCmd.PersistentFlags().StringVar(&dbUser, "db-user", defaults.User, "Database user")
	rootCmd.PersistentFlags().StringVar(&dbPassword, "db-password", defaults.Password, "Database password")
	rootCmd.PersistentFlags().StringVar(&dbName, "db-name", defaults.Database, "Database name")

	rootCmd.AddCommand(contractCmd)
	rootCmd.AddCommand(tariffCmd)
	rootCmd.AddCommand(vendorCmd)
	rootCmd.AddCommand(rateCmd)
	rootCmd.AddCommand(trackingCmd)
	rootCmd.AddCommand(documentCmd)
	rootCmd.AddCommand(shipmentCmd)
	rootCmd.AddCommand(locationCmd)
	rootCmd.AddCommand(laneCmd)
	rootCmd.AddCommand(standardRouteCmd)
	rootCmd.AddCommand(sopDefinitionCmd)
	rootCmd.AddCommand(sopInstanceCmd)
	rootCmd.AddCommand(eventCmd)
}

func Execute() error {
	return rootCmd.Execute()
}
