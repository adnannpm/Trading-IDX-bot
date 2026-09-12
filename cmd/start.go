package cmd

import (
	"agent-bot/internal/api"
	"agent-bot/internal/bot"
	"agent-bot/internal/database"
	"agent-bot/internal/service"
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "A brief description of your command",
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("Starting Bot...")
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

		log.Println("Bot connected successfully. Press CTRL+C to exit.")

		payloadAgent := service.HearbeatPayload{
			AgentName: "IDX Scanner Bot",
			Version: "1.0.0",
			Interval: "10 Min",
			Status: "Active",
		}

		log.Println("Menghubungkan Agent ke Nusa Admin")
		if err := service.SendStatus("/online", payloadAgent); err != nil {
			log.Fatalf("Failed to send status to the server: %v", err)
		}

		log.Println("Agent connected successfully.")
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go api.StartServer(ctx, ":8080")
		go bot.StartStatusPoller(ctx, service.PollInterval)

		ticker := time.NewTicker(service.PingInterval)
		go func() {
			for {
				select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						_ = service.SendStatus("/heartbeat", payloadAgent)
				}
			}
		}()

		sc := make(chan os.Signal, 1)
		signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
		<-sc

		log.Println("Stop bot, send status offline")
		ticker.Stop()
		cancel()

		payloadAgent.Status = "Offline"
		_ = service.SendStatus("/offline", payloadAgent)

		log.Println("Bot shutting down...")
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
