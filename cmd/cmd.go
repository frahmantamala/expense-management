package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/frahmantamala/expense-management/internal"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	clearData bool
)

var rootCmd = &cobra.Command{
	Use:   "expense-management",
	Short: "Expense Management",
	Long:  `For managing expenses and financial transactions.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func loadConfig(path string) (*internal.Config, error) {
	// Check if we're running in Docker environment
	if os.Getenv("APP_ENV") == "production" || os.Getenv("DOCKER_ENV") == "true" {
		// Load configuration from environment variables (Docker deployment)
		cfg := internal.LoadConfigFromEnv()
		if err := cfg.Validate(); err != nil {
			return nil, fmt.Errorf("error validating config from environment: %w", err)
		}
		return cfg, nil
	}

	// Load configuration from file (development)
	v := viper.New()
	v.AddConfigPath(path)
	v.SetConfigName("config")
	v.SetConfigType("yml")
	v.SetEnvPrefix("ENV")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config: %w", err)
	}

	var cfg internal.Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	return &cfg, nil
}

func init() {
	seedCmd.Flags().BoolVar(&clearData, "clear", false, "Clear existing data before seeding")

	rootCmd.AddCommand(httpServerCmd)
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(seedCmd)
}
