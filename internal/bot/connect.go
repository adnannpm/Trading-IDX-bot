package bot

import (
	"github.com/bwmarrin/discordgo"
)

var Session *discordgo.Session

func ConnectBot() error {
	token := "MTU0MDE4ODgxNzk5NzQzMDgxNA.GSjMRL.3D41R_RBpXm3iBHsNu5zVKfsUof7aRhhzUWP2E"
	bot, err := discordgo.New("Bot " + token)
	if err != nil {
		return err
	}

	bot.AddHandler(ReadyHandler)
	bot.AddHandler(InteractionHandler)
	bot.AddHandler(MessageHandler)

	bot.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildMessages |
		discordgo.IntentsMessageContent

	err = bot.Open()
	if err != nil {
		return err
	}

	Session = bot
	return nil
}