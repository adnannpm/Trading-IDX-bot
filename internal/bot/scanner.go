package bot

import (
	"agent-bot/internal/config"
	"agent-bot/internal/service"
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

var (
	DefaultTopAraChannelID = config.DefaultTopAraChannelID

	DefaultTopArbChannelID = config.DefaultTopArbChannelID

	sentAlerts     = make(map[string]time.Time)
	sentAlertsLock sync.Mutex

	lastBroadcastMsgIDs  = make(map[string]string)
	lastBroadcastMsgLock sync.Mutex
)

func GetLastBroadcastMessageID(channelID string) string {
	lastBroadcastMsgLock.Lock()
	defer lastBroadcastMsgLock.Unlock()
	return lastBroadcastMsgIDs[channelID]
}

func SetLastBroadcastMessageID(channelID, msgID string) {
	lastBroadcastMsgLock.Lock()
	defer lastBroadcastMsgLock.Unlock()
	lastBroadcastMsgIDs[channelID] = msgID
}

func SendAndReplaceBroadcast(s *discordgo.Session, channelID string, embed *discordgo.MessageEmbed) (*discordgo.Message, error) {
	lastBroadcastMsgLock.Lock()
	oldMsgID := lastBroadcastMsgIDs[channelID]
	lastBroadcastMsgLock.Unlock()

	newMsg, err := s.ChannelMessageSendEmbed(channelID, embed)
	if err != nil {
		return nil, err
	}

	lastBroadcastMsgLock.Lock()
	lastBroadcastMsgIDs[channelID] = newMsg.ID
	lastBroadcastMsgLock.Unlock()

	if oldMsgID != "" {
		if delErr := s.ChannelMessageDelete(channelID, oldMsgID); delErr != nil {
			log.Printf("[Scanner] Warning: gagal menghapus pesan lama %s di channel %s: %v\n", oldMsgID, channelID, delErr)
		}
	} else {
		cleanPreviousBotAlerts(s, channelID, newMsg.ID)
	}

	return newMsg, nil
}

func cleanPreviousBotAlerts(s *discordgo.Session, channelID, currentMsgID string) {
	if s == nil {
		return
	}

	var botID string
	if s.State != nil && s.State.User != nil {
		botID = s.State.User.ID
	}

	msgs, err := s.ChannelMessages(channelID, 10, currentMsgID, "", "")
	if err != nil {
		return
	}

	for _, m := range msgs {
		if m.ID == currentMsgID {
			continue
		}
		if botID != "" && m.Author != nil && m.Author.ID == botID && len(m.Embeds) > 0 {
			title := m.Embeds[0].Title
			if strings.Contains(title, "Top 10 Saham") || strings.Contains(title, "Emiten Momentum") || strings.Contains(title, "MOMENTUM") {
				_ = s.ChannelMessageDelete(channelID, m.ID)
			}
		}
	}
}

func GetTopAraChannelID() string {
	return config.Get().TopAraChannelID
}

func GetTopArbChannelID() string {
	return config.Get().TopArbChannelID
}

func GetMomentumChannelID() string {
	return config.Get().MomentumChannelID
}

func FormatPrice(price float64) string {
	return service.FormatNumberWithDots(int64(price))
}

func CreateStockEmbed(q service.StockQuote) *discordgo.MessageEmbed {
	var title string
	var statusLabel string
	color := 0x00D084

	if q.IsARA {
		title = fmt.Sprintf("🔥 [ARA DETECTED] %s - %s", q.DisplaySymbol, q.Name)
		statusLabel = fmt.Sprintf("🔥 **AUTO REJECTION ATAS (ARA ~%.0f%%)**", q.ARALimitPct)
		color = 0xFF4D4D
	} else {
		title = fmt.Sprintf("🚀 [TOP GAINER] %s - %s", q.DisplaySymbol, q.Name)
		statusLabel = "🚀 **TOP GAINER IDX**"
	}

	changeSign := "+"
	if q.Change < 0 {
		changeSign = ""
	}

	tradingViewURL := fmt.Sprintf("https://id.tradingview.com/symbols/IDX-%s/", q.DisplaySymbol)

	embed := &discordgo.MessageEmbed{
		Title:       title,
		URL:         tradingViewURL,
		Description: fmt.Sprintf("Saham **%s** mencatatkan penguatan sebesar **%s%.2f%%** hari ini di Bursa Efek Indonesia (IDX).", q.DisplaySymbol, changeSign, q.ChangePercent),
		Color:       color,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "💰 Harga Terkini",
				Value:  fmt.Sprintf("`Rp %s`", FormatPrice(q.Price)),
				Inline: true,
			},
			{
				Name:   "📈 Perubahan",
				Value:  fmt.Sprintf("`%s%s (%s%.2f%%)`", changeSign, FormatPrice(q.Change), changeSign, q.ChangePercent),
				Inline: true,
			},
			{
				Name:   "🏷️ Status",
				Value:  statusLabel,
				Inline: true,
			},
			{
				Name:   "💵 Rentang Hari Ini",
				Value:  fmt.Sprintf("`Rp %s - Rp %s`", FormatPrice(q.Low), FormatPrice(q.High)),
				Inline: true,
			},
			{
				Name:   "🕒 Harga Pembukaan",
				Value:  fmt.Sprintf("`Rp %s`", FormatPrice(q.Open)),
				Inline: true,
			},
			{
				Name:   "📦 Volume Transaksi",
				Value:  fmt.Sprintf("`%s`", service.FormatVolume(q.Volume)),
				Inline: true,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "IDX Market Scanner • Sumber: Yahoo Finance Feed • Nusa Terminal",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	return embed
}

func CreateTop10StocksEmbed(stocks []service.StockQuote) *discordgo.MessageEmbed {
	medals := []string{"🥇", "🥈", "🥉", "4️⃣", "5️⃣", "6️⃣", "7️⃣", "8️⃣", "9️⃣", "🔟"}

	var sb strings.Builder
	sb.WriteString("Berikut pantauan realtime **Top 10 Saham ARA & Gainers** di Bursa Efek Indonesia (IDX):\n\n")

	hasARA := false
	for i, q := range stocks {
		if i >= 10 {
			break
		}
		if q.IsARA {
			hasARA = true
		}

		medal := fmt.Sprintf("`#%d`", i+1)
		if i < len(medals) {
			medal = medals[i]
		}

		tradingViewURL := fmt.Sprintf("https://id.tradingview.com/symbols/IDX-%s/", q.DisplaySymbol)

		statusBadge := "🚀 *Top Gainer*"
		if q.IsARA {
			statusBadge = fmt.Sprintf("🔥 **ARA (~%.0f%%)**", q.ARALimitPct)
		}

		changeSign := "+"
		if q.Change < 0 {
			changeSign = ""
		}

		lots := q.Volume / 100
		var lotStr string
		if lots >= 1_000_000 {
			lotStr = fmt.Sprintf("%.2fM lot", float64(lots)/1_000_000)
		} else if lots >= 1_000 {
			lotStr = fmt.Sprintf("%.1fK lot", float64(lots)/1_000)
		} else {
			lotStr = fmt.Sprintf("%d lot", lots)
		}

		sb.WriteString(fmt.Sprintf("%s **[%s](%s)** — %s\n", medal, q.DisplaySymbol, tradingViewURL, q.Name))
		sb.WriteString(fmt.Sprintf("> 💰 `Rp %s` • 📈 `%s%.2f%%` (%s%s) • 📦 `%s` • %s\n\n",
			FormatPrice(q.Price),
			changeSign, q.ChangePercent,
			changeSign, FormatPrice(q.Change),
			lotStr,
			statusBadge,
		))
	}

	color := 0x00D084
	title := "🚀 Top 10 Saham ARA & Gainers Hari Ini (IDX)"
	if hasARA {
		title = "🔥 [ARA DETECTED] Top 10 Saham ARA & Gainers (IDX)"
		color = 0xFF5722
	}

	return &discordgo.MessageEmbed{
		Title:       title,
		Description: sb.String(),
		Color:       color,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "IDX Market Scanner • Sumber: Yahoo Finance Feed • Nusa Terminal",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

func BroadcastTop10ARA(s *discordgo.Session, channelID string) ([]service.StockQuote, error) {
	if s == nil {
		return nil, fmt.Errorf("sesi Discord bot belum siap")
	}

	if channelID == "" {
		channelID = GetTopAraChannelID()
	}

	gainers, err := service.FetchTopGainers(10)
	if err != nil {
		return nil, fmt.Errorf("gagal memindai saham gainers: %w", err)
	}

	if len(gainers) == 0 {
		return nil, fmt.Errorf("tidak ada data gainer yang ditemukan")
	}

	embed := CreateTop10StocksEmbed(gainers)
	_, err = SendAndReplaceBroadcast(s, channelID, embed)
	if err != nil {
		return nil, fmt.Errorf("gagal mengirim pesan top 10 embed ke channel %s: %w", channelID, err)
	}

	log.Printf("[Scanner] Berhasil mengirim rangkuman Top 10 Saham ARA/Gainers ke channel %s (1 pesan tunggal)\n", channelID)
	return gainers, nil
}

func CreateTop10ARBEmbed(stocks []service.StockQuote) *discordgo.MessageEmbed {
	medals := []string{"🥇", "🥈", "🥉", "4️⃣", "5️⃣", "6️⃣", "7️⃣", "8️⃣", "9️⃣", "🔟"}

	var sb strings.Builder
	sb.WriteString("Berikut pantauan realtime **Top 10 Saham ARB & Losers** terdalam di Bursa Efek Indonesia (IDX):\n\n")

	hasARB := false
	for i, q := range stocks {
		if i >= 10 {
			break
		}
		if q.IsARB {
			hasARB = true
		}

		medal := fmt.Sprintf("`#%d`", i+1)
		if i < len(medals) {
			medal = medals[i]
		}

		tradingViewURL := fmt.Sprintf("https://id.tradingview.com/symbols/IDX-%s/", q.DisplaySymbol)

		statusBadge := "📉 *Top Loser*"
		if q.IsARB {
			statusBadge = fmt.Sprintf("⚠️ **ARB (~%.0f%%)**", q.ARBLimitPct)
		}

		changeStr := FormatPrice(q.Change)
		if q.Change > 0 {
			changeStr = "+" + changeStr
		}

		lots := q.Volume / 100
		var lotStr string
		if lots >= 1_000_000 {
			lotStr = fmt.Sprintf("%.2fM lot", float64(lots)/1_000_000)
		} else if lots >= 1_000 {
			lotStr = fmt.Sprintf("%.1fK lot", float64(lots)/1_000)
		} else {
			lotStr = fmt.Sprintf("%d lot", lots)
		}

		sb.WriteString(fmt.Sprintf("%s **[%s](%s)** — %s\n", medal, q.DisplaySymbol, tradingViewURL, q.Name))
		sb.WriteString(fmt.Sprintf("> 💰 `Rp %s` • 📉 `%.2f%%` (%s) • 📦 `%s` • %s\n\n",
			FormatPrice(q.Price),
			q.ChangePercent,
			changeStr,
			lotStr,
			statusBadge,
		))
	}

	color := 0xED4245
	title := "📉 Top 10 Saham ARB & Top Losers Hari Ini (IDX)"
	if hasARB {
		title = "⚠️ [ARB DETECTED] Top 10 Saham ARB & Losers (IDX)"
	}

	return &discordgo.MessageEmbed{
		Title:       title,
		Description: sb.String(),
		Color:       color,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "IDX Market Scanner • Sumber: Yahoo Finance Feed • Nusa Terminal",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

func BroadcastTop10ARB(s *discordgo.Session, channelID string) ([]service.StockQuote, error) {
	if s == nil {
		return nil, fmt.Errorf("sesi Discord bot belum siap")
	}

	if channelID == "" {
		channelID = GetTopArbChannelID()
	}

	losers, err := service.FetchTopLosers(10)
	if err != nil {
		return nil, fmt.Errorf("gagal memindai saham losers: %w", err)
	}

	if len(losers) == 0 {
		return nil, fmt.Errorf("tidak ada data loser yang ditemukan")
	}

	embed := CreateTop10ARBEmbed(losers)
	_, err = SendAndReplaceBroadcast(s, channelID, embed)
	if err != nil {
		return nil, fmt.Errorf("gagal mengirim pesan top 10 ARB embed ke channel %s: %w", channelID, err)
	}

	log.Printf("[Scanner] Berhasil mengirim rangkuman Top 10 Saham ARB/Losers ke channel %s (1 pesan tunggal)\n", channelID)
	return losers, nil
}

func CreateTopMomentumEmbed(stocks []service.StockQuote) *discordgo.MessageEmbed {
	medals := []string{"🥇", "🥈", "🥉", "4️⃣", "5️⃣", "6️⃣", "7️⃣", "8️⃣", "9️⃣", "🔟"}

	var sb strings.Builder
	sb.WriteString("Berikut radar pantauan **Emiten Momentum & Unusual Volume Spike** di Bursa Efek Indonesia (IDX):\n")
	sb.WriteString("*(Deteksi dini saham akumulasi/potensi ARA sebelum terkunci antrean)*\n\n")

	hasSuperMomentum := false
	for i, q := range stocks {
		if i >= 10 {
			break
		}

		medal := fmt.Sprintf("`#%d`", i+1)
		if i < len(medals) {
			medal = medals[i]
		}

		tradingViewURL := fmt.Sprintf("https://id.tradingview.com/symbols/IDX-%s/", q.DisplaySymbol)

		var statusBadge string
		if q.IsVolumeSpike && q.Is52WeekHighBreakout {
			hasSuperMomentum = true
			statusBadge = fmt.Sprintf("🔥 **SUPER MOMENTUM (~%.1fx Vol + ATH/52W High)**", q.VolumeSpikeRatio)
		} else if q.Is52WeekHighBreakout {
			statusBadge = "👑 **52-WEEK HIGH BREAKOUT (ATH)**"
		} else if q.IsVolumeSpike {
			statusBadge = fmt.Sprintf("🚀 **VOLUME SPIKE (~%.1fx Avg 10D)**", q.VolumeSpikeRatio)
		} else {
			statusBadge = "⚡ *Early Momentum*"
		}

		changeStr := FormatPrice(q.Change)
		if q.Change > 0 {
			changeStr = "+" + changeStr
		}

		lots := q.Volume / 100
		var lotStr string
		if lots >= 1_000_000 {
			lotStr = fmt.Sprintf("%.2fM lot", float64(lots)/1_000_000)
		} else if lots >= 1_000 {
			lotStr = fmt.Sprintf("%.1fK lot", float64(lots)/1_000)
		} else {
			lotStr = fmt.Sprintf("%d lot", lots)
		}

		avgLots := q.AvgVolume10Day / 100
		var avgLotStr string
		if avgLots >= 1_000_000 {
			avgLotStr = fmt.Sprintf("%.2fM lot", float64(avgLots)/1_000_000)
		} else if avgLots >= 1_000 {
			avgLotStr = fmt.Sprintf("%.1fK lot", float64(avgLots)/1_000)
		} else {
			avgLotStr = fmt.Sprintf("%d lot", avgLots)
		}

		sb.WriteString(fmt.Sprintf("%s **[%s](%s)** — %s\n", medal, q.DisplaySymbol, tradingViewURL, q.Name))
		if q.AvgVolume10Day > 0 {
			sb.WriteString(fmt.Sprintf("> 💰 `Rp %s` • 📈 `+%.2f%%` (%s) • 📦 `%s` (Rata2 10D: `%s`)\n",
				FormatPrice(q.Price),
				q.ChangePercent,
				changeStr,
				lotStr,
				avgLotStr,
			))
		} else {
			sb.WriteString(fmt.Sprintf("> 💰 `Rp %s` • 📈 `+%.2f%%` (%s) • 📦 `%s`\n",
				FormatPrice(q.Price),
				q.ChangePercent,
				changeStr,
				lotStr,
			))
		}

		if q.FiftyTwoWeekHigh > 0 {
			sb.WriteString(fmt.Sprintf("> %s • 🎯 `52W High: Rp %s`\n\n", statusBadge, FormatPrice(q.FiftyTwoWeekHigh)))
		} else {
			sb.WriteString(fmt.Sprintf("> %s\n\n", statusBadge))
		}
	}

	color := 0xF1C40F
	title := "🚀 Radar Emiten Momentum & Unusual Volume (IDX)"
	if hasSuperMomentum {
		title = "🔥 [MOMENTUM ALERT] Unusual Volume Spike & 52W High Breakout"
		color = 0xE67E22
	}

	return &discordgo.MessageEmbed{
		Title:       title,
		Description: sb.String(),
		Color:       color,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Emiten Momentum Scanner • Early Entry Detector • Nusa Terminal",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

func BroadcastTopMomentum(s *discordgo.Session, channelID string) ([]service.StockQuote, error) {
	if s == nil {
		return nil, fmt.Errorf("sesi Discord bot belum siap")
	}

	if channelID == "" {
		channelID = GetMomentumChannelID()
	}

	stocks, err := service.FetchMomentumStocks(10)
	if err != nil {
		return nil, fmt.Errorf("gagal memindai saham momentum: %w", err)
	}

	if len(stocks) == 0 {
		return nil, fmt.Errorf("tidak ada saham momentum yang memenuhi kriteria saat ini")
	}

	embed := CreateTopMomentumEmbed(stocks)
	_, err = SendAndReplaceBroadcast(s, channelID, embed)
	if err != nil {
		return nil, fmt.Errorf("gagal mengirim pesan momentum embed ke channel %s: %w", channelID, err)
	}

	log.Printf("[Scanner] Berhasil mengirim rangkuman Top %d Saham Momentum ke channel %s (1 pesan tunggal)\n", len(stocks), channelID)
	return stocks, nil
}

func SendTopStockAlert(s *discordgo.Session, channelID string, quote service.StockQuote) (*discordgo.Message, error) {
	if s == nil {
		return nil, fmt.Errorf("sesi Discord bot belum siap")
	}

	if channelID == "" {
		channelID = GetTopAraChannelID()
	}

	embed := CreateStockEmbed(quote)
	return s.ChannelMessageSendEmbed(channelID, embed)
}

func ScanAndBroadcastTopARA(s *discordgo.Session, channelID string, limit int, bypassCooldown bool) ([]service.StockQuote, error) {
	if limit >= 10 {
		return BroadcastTop10ARA(s, channelID)
	}

	if channelID == "" {
		channelID = GetTopAraChannelID()
	}

	if limit <= 0 {
		limit = 3
	}

	gainers, err := service.FetchTopGainers(limit)
	if err != nil {
		return nil, fmt.Errorf("gagal memindai saham gainers: %w", err)
	}

	if len(gainers) == 0 {
		return nil, fmt.Errorf("tidak ada data gainer yang ditemukan")
	}

	var sent []service.StockQuote
	sentAlertsLock.Lock()
	defer sentAlertsLock.Unlock()

	for _, g := range gainers {
		lastSent, exists := sentAlerts[g.Symbol]
		if bypassCooldown || !exists || time.Since(lastSent) > 30*time.Minute {
			_, err := SendTopStockAlert(s, channelID, g)
			if err != nil {
				log.Printf("[Scanner] Gagal mengirim alert %s ke channel %s: %v\n", g.DisplaySymbol, channelID, err)
			} else {
				log.Printf("[Scanner] Berhasil mengirim alert %s (+%.2f%%) ke channel %s\n", g.DisplaySymbol, g.ChangePercent, channelID)
				sentAlerts[g.Symbol] = time.Now()
				sent = append(sent, g)
			}
		}

		if len(sent) >= limit {
			break
		}
	}

	return sent, nil
}

func StartStockScanner(ctx context.Context, s *discordgo.Session, interval time.Duration) {
	araChannelID := GetTopAraChannelID()
	arbChannelID := GetTopArbChannelID()
	momentumChannelID := GetMomentumChannelID()

	go func() {
		time.Sleep(3 * time.Second)
		log.Printf("[Scanner] Memulai scan awal: Top 10 ARA -> %s & Top 10 ARB -> %s...\n", araChannelID, arbChannelID)
		sentAra, errAra := BroadcastTop10ARA(s, araChannelID)
		if errAra != nil {
			log.Printf("[Scanner] Gagal scan awal Top ARA: %v\n", errAra)
		} else {
			log.Printf("[Scanner] Scan awal selesai, Top %d saham ARA terkirim ke %s\n", len(sentAra), araChannelID)
		}

		sentArb, errArb := BroadcastTop10ARB(s, arbChannelID)
		if errArb != nil {
			log.Printf("[Scanner] Gagal scan awal Top ARB: %v\n", errArb)
		} else {
			log.Printf("[Scanner] Scan awal selesai, Top %d saham ARB terkirim ke %s\n", len(sentArb), arbChannelID)
		}

		if momentumChannelID != "" && momentumChannelID != araChannelID {
			sentMom, errMom := BroadcastTopMomentum(s, momentumChannelID)
			if errMom != nil {
				log.Printf("[Scanner] Info scan awal Momentum: %v\n", errMom)
			} else {
				log.Printf("[Scanner] Scan awal selesai, Top %d saham Momentum terkirim ke %s\n", len(sentMom), momentumChannelID)
			}
		}
	}()

	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				log.Println("[Scanner] Menjalankan pemindaian berkala Top 10 ARA, ARB & Momentum...")
				_, _ = BroadcastTop10ARA(s, araChannelID)
				_, _ = BroadcastTop10ARB(s, arbChannelID)
				if momentumChannelID != "" && momentumChannelID != araChannelID {
					_, _ = BroadcastTopMomentum(s, momentumChannelID)
				}
			}
		}
	}()
}
