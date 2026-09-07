package template

import "github.com/bwmarrin/discordgo"

const (
	ModalButtonCustomID = "open_modal"
	ModalCustomID       = "modal_input_form"
	TextInputCustomID   = "user_input_field"
)

var Button = discordgo.Button{
	CustomID: ModalButtonCustomID,
	Label:    "Open Modal",
	Style:    discordgo.PrimaryButton,
}

func GetModalTemplate() discordgo.InteractionResponseData {
	return discordgo.InteractionResponseData{
		CustomID: ModalCustomID,
		Title:    "Input Token",
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    TextInputCustomID,
						Label:       "Input your token",
						Style:       discordgo.TextInputParagraph,
						Placeholder: "token...",
						Required:    true,
						MinLength:   1,
						MaxLength:   500,
					},
				},
			},
		},
	}
}