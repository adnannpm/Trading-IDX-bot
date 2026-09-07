package bot

import (
	"agent-bot/internal/bot/template"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

// ReadyHandler is triggered when the bot successfully connects to the Discord Gateway
func ReadyHandler(s *discordgo.Session, r *discordgo.Ready) {
	log.Printf("Bot logged in as %s#%s (%s)\n", r.User.Username, r.User.Discriminator, r.User.ID)
}

// InteractionHandler handles button clicks and modal submissions
func InteractionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	// Handle component interactions such as button clicks
	case discordgo.InteractionMessageComponent:
		data := i.MessageComponentData()
		if data.CustomID == template.ModalButtonCustomID {
			modalData := template.GetModalTemplate()

			// Open modal in response to the button click
			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseModal,
				Data: &modalData,
			})
			if err != nil {
				log.Printf("Failed to open modal: %v\n", err)
			}
		}

	// Handle modal submit events
	case discordgo.InteractionModalSubmit:
		data := i.ModalSubmitData()
		if data.CustomID == template.ModalCustomID {
			var userInput string

			for _, comp := range data.Components {
				if row, ok := comp.(*discordgo.ActionsRow); ok {
					for _, c := range row.Components {
						if input, ok := c.(*discordgo.TextInput); ok {
							if input.CustomID == template.TextInputCustomID {
								userInput = input.Value
							}
						}
					}
				}
			}

			var username string
			if i.Member != nil && i.Member.User != nil {
				username = i.Member.User.Username
			} else if i.User != nil {
				username = i.User.Username
			}

			log.Printf("User %s submitted modal input: %s\n", username, userInput)

			// Respond back to user (ephemeral = only visible to the user)
			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("✅ Thank you, your token has been received:\n`%s`", userInput),
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			if err != nil {
				log.Printf("Failed to respond to modal submission: %v\n", err)
			}
		}
	}
}

// MessageHandler handles standard text commands (e.g. !get-token to send the modal button)
func MessageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore messages sent by the bot itself
	if m.Author.Bot {
		return
	}

	if m.Content == "!get-token" {
		_, err := SendModalButton(s, m.ChannelID)
		if err != nil {
			log.Printf("Failed to send modal button: %v\n", err)
		}
	}
}

// SendModalButton sends a message with the button to open the modal
func SendModalButton(s *discordgo.Session, channelID string) (*discordgo.Message, error) {
	return s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content: "Click the button below to submit your token:",
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					template.Button,
				},
			},
		},
	})
}
