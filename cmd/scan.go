package cmd

import (
	"agent-bot/internal/bot"
	"log"

	"github.com/spf13/cobra"
)

var scanAraCmd = &cobra.Command{
	Use:   "scan-ara",
	Short: "Memindai Top 10 saham ARA / Gainers dan mengirim 1 pesan rangkuman ke channel top-ara",
	Run: func(cmd *cobra.Command, args []string) {
		runScan(true, false, false)
	},
}

var scanArbCmd = &cobra.Command{
	Use:   "scan-arb",
	Short: "Memindai Top 10 saham ARB / Losers dan mengirim 1 pesan rangkuman ke channel top-arb",
	Run: func(cmd *cobra.Command, args []string) {
		runScan(false, true, false)
	},
}

var scanMomentumCmd = &cobra.Command{
	Use:   "scan-momentum",
	Short: "Memindai saham Early Momentum & Unusual Volume Spike ke channel emiten-momentum",
	Run: func(cmd *cobra.Command, args []string) {
		runScan(false, false, true)
	},
}

var scanAllCmd = &cobra.Command{
	Use:   "scan",
	Short: "Memindai Top 10 saham ARA, ARB, dan Momentum sekaligus ke masing-masing channel",
	Run: func(cmd *cobra.Command, args []string) {
		runScan(true, true, true)
	},
}

func runScan(doAra, doArb, doMomentum bool) {
	log.Println("Menghubungkan bot ke Discord...")
	if err := bot.ConnectBot(); err != nil {
		log.Fatalf("Gagal terhubung ke Discord: %v", err)
	}
	defer func() {
		if bot.Session != nil {
			_ = bot.Session.Close()
		}
	}()

	if doAra {
		araChannelID := bot.GetTopAraChannelID()
		log.Printf("Mengirim rangkuman Top 10 saham ARA/Gainers ke channel: %s\n", araChannelID)
		sentAra, err := bot.BroadcastTop10ARA(bot.Session, araChannelID)
		if err != nil {
			log.Printf("❌ Gagal scan ARA: %v\n", err)
		} else {
			log.Printf("✅ Berhasil mengirim 1 pesan rangkuman Top %d ARA ke channel %s!\n", len(sentAra), araChannelID)
		}
	}

	if doArb {
		arbChannelID := bot.GetTopArbChannelID()
		log.Printf("Mengirim rangkuman Top 10 saham ARB/Losers ke channel: %s\n", arbChannelID)
		sentArb, err := bot.BroadcastTop10ARB(bot.Session, arbChannelID)
		if err != nil {
			log.Printf("❌ Gagal scan ARB: %v\n", err)
		} else {
			log.Printf("✅ Berhasil mengirim 1 pesan rangkuman Top %d ARB ke channel %s!\n", len(sentArb), arbChannelID)
		}
	}

	if doMomentum {
		momentumChannelID := bot.GetMomentumChannelID()
		log.Printf("Mengirim rangkuman saham Emiten Momentum ke channel: %s\n", momentumChannelID)
		sentMom, err := bot.BroadcastTopMomentum(bot.Session, momentumChannelID)
		if err != nil {
			log.Printf("❌ Gagal scan Momentum: %v\n", err)
		} else {
			log.Printf("✅ Berhasil mengirim 1 pesan rangkuman Top %d Momentum ke channel %s!\n", len(sentMom), momentumChannelID)
		}
	}
}

func init() {
	rootCmd.AddCommand(scanAraCmd)
	rootCmd.AddCommand(scanArbCmd)
	rootCmd.AddCommand(scanMomentumCmd)
	rootCmd.AddCommand(scanAllCmd)
}
