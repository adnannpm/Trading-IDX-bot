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

var disableLaravel bool

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Menjalankan Agent Discord Bot dan service terkait",
	Run: func(cmd *cobra.Command, args []string) {
		if disableLaravel {
			service.LaravelEnabled = false
			log.Println("[Config] Integrasi Laravel dinonaktifkan (--no-laravel).")
		}

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
			Version:   "1.0.0",
			Interval:  "10 Min",
			Status:    "Active",
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Menjalankan API Server lokal untuk menerima webhook/request dari Laravel
		go api.StartServer(ctx, ":8080")

		// Menjalankan poller status user
		go bot.StartStatusPoller(ctx, service.PollInterval)

		// Menjalankan scanner saham ARA / Gainers berkala ke channel Discord
		go bot.StartStockScanner(ctx, bot.Session, 15*time.Minute)

		// Menjalankan worker heartbeat & auto-reconnect ke Laravel secara non-blocking
		go service.StartHeartbeatWorker(ctx, payloadAgent)

		sc := make(chan os.Signal, 1)
		signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
		<-sc

		log.Println("Menghentikan bot...")
		cancel()

		// Kirim status offline ke Laravel jika terhubung
		service.ShutdownHeartbeat(payloadAgent)

		log.Println("Bot shutting down...")
	},
}

func init() {
	startCmd.Flags().BoolVar(&disableLaravel, "no-laravel", false, "Jalankan bot tanpa menghubungkan ke Nusa Admin (Laravel)")
	rootCmd.AddCommand(startCmd)
}
