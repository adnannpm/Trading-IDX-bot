/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"agent-bot/internal/bot"
	"agent-bot/internal/database"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "A brief description of your command",
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("Database connection is being established...")
		if err := database.Connect(); err != nil {
			log.Fatalf("Failed to connect to the database: %v", err)
		}

		log.Println("Database connection established successfully.")

		if err := bot.ConnectBot(); err != nil {
			log.Fatalf("Failed to connect to the bot: %v", err)
		}
		defer func() {
			if bot.Session != nil {
				_ = bot.Session.Close()
			}
		}()

		log.Println("Bot connected successfully. Tekan CTRL+C untuk berhenti.")

		sc := make(chan os.Signal, 1)
		signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
		<-sc
		log.Println("Bot dimatikan.")
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
