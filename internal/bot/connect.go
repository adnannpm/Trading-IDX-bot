package bot

import (
	"fmt"
	"os"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var Session *discordgo.Session

func ConnectBot() error {
	token := strings.TrimSpace(os.Getenv("DISCORD_TOKEN"))
	if token == "" {
		return fmt.Errorf("DISCORD_TOKEN wajib diisi")
	}
	bot, err := discordgo.New("Bot " + token)
	if err != nil {
		return err
	}
	bot.AddHandler(ReadyHandler)
	bot.AddHandler(InteractionHandler)
	bot.Identify.Intents = discordgo.IntentsGuilds
	if err = bot.Open(); err != nil {
		return err
	}
	guildID := strings.TrimSpace(os.Getenv("DISCORD_GUILD_ID"))
	if err = registerCommands(bot, bot.State.User.ID, guildID); err != nil {
		_ = bot.Close()
		return err
	}
	if guildID != "" {
		ServerGuildID = guildID
	}
	Session = bot
	return nil
}
