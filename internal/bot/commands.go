package bot

import (
	"agent-bot/internal/bot/template"
	"agent-bot/internal/service"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func commandDefinitions() []*discordgo.ApplicationCommand {
	dm := false
	return []*discordgo.ApplicationCommand{
		{Name: "saham", Description: "Lihat harga saham IDX terbaru", DMPermission: &dm, Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionString, Name: "kode", Description: "Kode saham (contoh BBCA) atau IHSG", Required: true, Autocomplete: true},
		}},
		{Name: "scan", Description: "Kirim rangkuman Top 10 ke channel pasar", DMPermission: &dm, Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "ara", Description: "Top 10 ARA / gainers"},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "arb", Description: "Top 10 ARB / losers"},
			{Type: discordgo.ApplicationCommandOptionSubCommand, Name: "all", Description: "Scan ARA dan ARB"},
		}},
		{Name: "verif", Description: "Aktivasi akun Discord dengan token Nusa", DMPermission: &dm},
	}
}

func registerCommands(s *discordgo.Session, appID, guildID string) error {
	for _, command := range commandDefinitions() {
		if _, err := s.ApplicationCommandCreate(appID, guildID, command); err != nil {
			return fmt.Errorf("registrasi /%s: %w", command.Name, err)
		}
	}
	return nil
}

var popularSymbols = []string{"ADRO", "AMMN", "ANTM", "ASII", "BBCA", "BBNI", "BBRI", "BBTN", "BMRI", "BRIS", "BRPT", "BUMI", "CPIN", "EXCL", "GOTO", "ICBP", "IHSG", "INCO", "INDF", "INKP", "ISAT", "ITMG", "KLBF", "MDKA", "MEDC", "PGAS", "PTBA", "SMGR", "TLKM", "TOWR", "UNTR", "UNVR"}
var symbolPattern = regexp.MustCompile(`^[A-Z][A-Z0-9-]{1,14}$`)

func stockSymbol(input string) (string, bool) {
	symbol := strings.TrimSuffix(strings.ToUpper(strings.TrimSpace(input)), ".JK")
	if symbol == "JKSE" || symbol == "^JKSE" {
		symbol = "IHSG"
	}
	return symbol, symbolPattern.MatchString(symbol)
}

func stockChoices(input string) []*discordgo.ApplicationCommandOptionChoice {
	query := strings.ToUpper(strings.TrimSpace(input))
	choices := make([]*discordgo.ApplicationCommandOptionChoice, 0, 25)
	for _, symbol := range popularSymbols {
		if strings.HasPrefix(symbol, query) {
			choices = append(choices, &discordgo.ApplicationCommandOptionChoice{Name: symbol, Value: symbol})
			if len(choices) == 25 {
				break
			}
		}
	}
	return choices
}

func respond(s *discordgo.Session, i *discordgo.InteractionCreate, response *discordgo.InteractionResponse) bool {
	if err := s.InteractionRespond(i.Interaction, response); err != nil {
		log.Printf("Interaction response failed: %v", err)
		return false
	}
	return true
}

func privateMessage(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	respond(s, i, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Content: message, Flags: discordgo.MessageFlagsEphemeral, AllowedMentions: &discordgo.MessageAllowedMentions{}}})
}

func editResponse(s *discordgo.Session, i *discordgo.InteractionCreate, edit *discordgo.WebhookEdit) {
	edit.AllowedMentions = &discordgo.MessageAllowedMentions{}
	if _, err := s.InteractionResponseEdit(i.Interaction, edit); err != nil {
		log.Printf("Interaction edit failed: %v", err)
	}
}

func handleAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	choices := []*discordgo.ApplicationCommandOptionChoice{}
	data := i.ApplicationCommandData()
	if data.Name == "saham" {
		for _, option := range data.Options {
			if option.Name == "kode" && option.Focused && option.Type == discordgo.ApplicationCommandOptionString {
				choices = stockChoices(option.StringValue())
			}
		}
	}
	respond(s, i, &discordgo.InteractionResponse{Type: discordgo.InteractionApplicationCommandAutocompleteResult, Data: &discordgo.InteractionResponseData{Choices: choices}})
}

func handleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.GuildID == "" || i.Member == nil || i.Member.User == nil {
		privateMessage(s, i, "Gunakan command ini di server Discord.")
		return
	}
	data := i.ApplicationCommandData()
	switch data.Name {
	case "verif":
		modal := template.GetModalTemplate()
		respond(s, i, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseModal, Data: &modal})
	case "saham":
		if len(data.Options) != 1 || data.Options[0].Name != "kode" || data.Options[0].Type != discordgo.ApplicationCommandOptionString {
			privateMessage(s, i, "Masukkan kode saham, contoh: /saham kode:BBCA.")
			return
		}
		symbol, valid := stockSymbol(data.Options[0].StringValue())
		if !valid {
			privateMessage(s, i, "Kode saham tidak valid. Contoh: BBCA atau IHSG.")
			return
		}
		if !respond(s, i, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredChannelMessageWithSource}) {
			return
		}
		updateQuote(s, i, symbol, i.Member.User.ID)
	case "scan":
		if len(data.Options) != 1 || data.Options[0].Type != discordgo.ApplicationCommandOptionSubCommand || (data.Options[0].Name != "ara" && data.Options[0].Name != "arb" && data.Options[0].Name != "all") {
			privateMessage(s, i, "Pilih /scan ara, /scan arb, atau /scan all.")
			return
		}
		if !respond(s, i, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral}}) {
			return
		}
		mode := data.Options[0].Name
		lines := []string{}
		for _, kind := range []string{"ara", "arb"} {
			if mode != "all" && mode != kind {
				continue
			}
			channel := GetTopAraChannelID()
			broadcast := BroadcastTop10ARA
			if kind == "arb" {
				channel = GetTopArbChannelID()
				broadcast = BroadcastTop10ARB
			}
			stocks, err := broadcast(s, channel)
			if err != nil {
				log.Printf("Slash scan %s failed: %v", kind, err)
				lines = append(lines, fmt.Sprintf("❌ %s gagal dikirim. Data mungkin tidak tersedia atau channel tidak dapat diakses. Coba lagi nanti.", strings.ToUpper(kind)))
			} else {
				lines = append(lines, fmt.Sprintf("✅ Top %d %s terkirim ke <#%s>.", len(stocks), strings.ToUpper(kind), channel))
			}
		}
		message := strings.Join(lines, "\n")
		editResponse(s, i, &discordgo.WebhookEdit{Content: &message})
	default:
		privateMessage(s, i, "Command tidak dikenal.")
	}
}

func updateQuote(s *discordgo.Session, i *discordgo.InteractionCreate, symbol, owner string) {
	quote, err := service.FetchStockQuote(symbol)
	if err != nil || quote == nil {
		log.Printf("Quote %s unavailable: %v", symbol, err)
		message := "❌ Data saham tidak tersedia. Periksa kode atau coba Refresh beberapa saat lagi."
		// Keep the last successful embed on a refresh failure.
		components := refreshComponents(symbol, owner)
		editResponse(s, i, &discordgo.WebhookEdit{Content: &message, Components: &components})
		return
	}
	message := ""
	embeds := []*discordgo.MessageEmbed{CreateStockEmbed(*quote)}
	components := refreshComponents(symbol, owner)
	editResponse(s, i, &discordgo.WebhookEdit{Content: &message, Embeds: &embeds, Components: &components})
}

func refreshComponents(symbol, owner string) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.Button{Label: "Refresh", Style: discordgo.SecondaryButton, CustomID: "stock_refresh:" + owner + ":" + symbol}}}}
}

func handleRefresh(s *discordgo.Session, i *discordgo.InteractionCreate) bool {
	id := i.MessageComponentData().CustomID
	if !strings.HasPrefix(id, "stock_refresh:") {
		return false
	}
	parts := strings.Split(id, ":")
	if len(parts) != 3 || i.GuildID == "" || i.Member == nil || i.Member.User == nil || parts[1] != i.Member.User.ID {
		privateMessage(s, i, "Tombol ini khusus pemanggil command. Gunakan /saham untuk membuka data sendiri.")
		return true
	}
	symbol, valid := stockSymbol(parts[2])
	if !valid {
		privateMessage(s, i, "Kode saham tidak valid.")
		return true
	}
	if respond(s, i, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate}) {
		updateQuote(s, i, symbol, parts[1])
	}
	return true
}
