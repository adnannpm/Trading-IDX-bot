package bot

import (
	"agent-bot/internal/config"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

var Session *discordgo.Session

func ConnectBot() error {
	cfg := config.Get()
	if cfg.DiscordToken == "" {
		return fmt.Errorf("DISCORD_TOKEN wajib diisi (.env atau environment variable)")
	}
	bot, err := discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		return err
	}
	bot.AddHandler(ReadyHandler)
	bot.AddHandler(InteractionHandler)
	bot.Identify.Intents = discordgo.IntentsGuilds
	if err = bot.Open(); err != nil {
		return err
	}
	if err = registerCommands(bot, bot.State.User.ID, cfg.DiscordGuildID); err != nil {
		_ = bot.Close()
		return err
	}
	if cfg.DiscordGuildID != "" {
		ServerGuildID = cfg.DiscordGuildID
	}
	Session = bot
	return nil
}
