package cmd

import (
	"agent-bot/internal/database"
	"agent-bot/internal/model"
	"log"

	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "A brief description of your command",
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("Database connection is being established...")
		if err := database.Connect(); err != nil {
			log.Fatalf("Failed to connect to the database: %v", err)
		}

		log.Println("Database connection established successfully.")
		if err := database.DB.AutoMigrate(&model.User{}); err != nil {
			log.Fatalf("Failed to migrate User model: %v", err)
		}
		log.Println("User model migrated successfully.")
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}
