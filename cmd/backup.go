package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/tderick/backup-companion-go/internal/backup"
	"github.com/tderick/backup-companion-go/internal/config"
)

// backupCmd represents the backup command
var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Load config using the root-level --config (cfgPath)
		cfg, err := config.LoadConfig(cfgPath)
		if err != nil {
			log.Fatalf("failed to load config: %v", err)
		}

		backup.Execute(cmd.Context(), cfg)
	},
}

func init() {
	rootCmd.AddCommand(backupCmd)

}
