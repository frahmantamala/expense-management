package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/frahmantamala/expense-management/internal"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	rootCmd.AddCommand(httpServerCmd)
	rootCmd.AddCommand(migrateCmd)
}
