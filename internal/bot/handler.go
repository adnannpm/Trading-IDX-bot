package bot

import (
	"agent-bot/internal/bot/template"
	"agent-bot/internal/database"
	"agent-bot/internal/model"
	"agent-bot/internal/service"
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
)

var RoleByPlanTier = map[string]string{
	"VIP": "1546438682276528168",
}

var DefaultMemberRoleID = "1546692502785101874"

var FreeRoleID = "1546692502785101874"

var ServerGuildID = ""

func ReadyHandler(s *discordgo.Session, r *discordgo.Ready) {
	log.Printf("Bot logged in as %s#%s (%s)\n", r.User.Username, r.User.Discriminator, r.User.ID)
}

func InteractionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionMessageComponent:
		data := i.MessageComponentData()
		if data.CustomID == template.ModalButtonCustomID {
			modalData := template.GetModalTemplate()

			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseModal,
				Data: &modalData,
			})
			if err != nil {
				log.Printf("Failed to open modal: %v\n", err)
			}
		}

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

			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Flags: discordgo.MessageFlagsEphemeral,
				},
			})
			if err != nil {
				log.Printf("Failed to defer interaction response: %v\n", err)
				return
			}

			var user *discordgo.User
			if i.Member != nil && i.Member.User != nil {
				user = i.Member.User
			} else if i.User != nil {
				user = i.User
			}

			if user == nil {
				log.Println("User not found in interaction")
				return
			}

			discordUsername := user.Username
			if user.Discriminator != "0" && user.Discriminator != "" {
				discordUsername = fmt.Sprintf("%s#%s", user.Username, user.Discriminator)
			}
			avatarURL := user.AvatarURL("256")

			log.Printf("User %s (%s) submitted token: %s\n", discordUsername, user.ID, userInput)

			req := service.TokenRequest{
				Token:           userInput,
				DiscordID:       user.ID,
				DiscordUsername: discordUsername,
				AvatarURL:       avatarURL,
			}

			tokenResp, err := service.ValidateToken("/tokens/validate", req)
			if err != nil || tokenResp == nil || !tokenResp.Valid {
				msg := "❌ Token yang Anda masukkan tidak valid atau kadaluarsa."
				if tokenResp != nil && tokenResp.Message != "" {
					msg = fmt.Sprintf("❌ %s", tokenResp.Message)
					if tokenResp.ClaimedBy != "" {
						msg += fmt.Sprintf("\n*(Sudah diklaim oleh: `%s`)*", tokenResp.ClaimedBy)
					}
				} else if err != nil {
					log.Printf("Error validating token: %v\n", err)
				}

				_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
					Content: msg,
					Flags:   discordgo.MessageFlagsEphemeral,
				})
				return
			}

			targetRoleID := DefaultMemberRoleID
			if tokenResp.Data != nil && tokenResp.Data.PlanTier != "" {
				if roleID, ok := RoleByPlanTier[tokenResp.Data.PlanTier]; ok && roleID != "" {
					targetRoleID = roleID
				}
			}

			if i.GuildID != "" {
				if ServerGuildID == "" {
					ServerGuildID = i.GuildID
				}
				if targetRoleID != "" {
					if err := s.GuildMemberRoleAdd(i.GuildID, user.ID, targetRoleID); err != nil {
						log.Printf("Failed to add role %s to user %s: %v\n", targetRoleID, user.ID, err)
					} else {
						log.Printf("Role %s successfully added to user %s\n", targetRoleID, discordUsername)
					}
				}
			}

			if database.DB != nil {
				var existingUser model.User
				if err := database.DB.Where("discord_id = ?", user.ID).First(&existingUser).Error; err == nil {
					existingUser.Username = discordUsername
					existingUser.Token = userInput
					existingUser.Status = "ACTIVE"
					existingUser.UpdatedAt = time.Now().Unix()
					database.DB.Save(&existingUser)
				} else {
					database.DB.Create(&model.User{
						DiscordID: user.ID,
						Username:  discordUsername,
						Token:     userInput,
						Status:    "ACTIVE",
						CreatedAt: time.Now().Unix(),
						UpdatedAt: time.Now().Unix(),
					})
				}
			}

			successMsg := "✅ Token valid dan akun Discord berhasil diaktivasi!"
			if tokenResp.Message != "" {
				successMsg = fmt.Sprintf("✅ **%s**", tokenResp.Message)
			}
			if tokenResp.Data != nil {
				successMsg += fmt.Sprintf("\n\n📋 **Detail Aktivasi:**\n• **Tier:** %s\n• **Durasi:** %d Hari\n• **Kadaluarsa:** `%s`",
					tokenResp.Data.PlanTier,
					tokenResp.Data.DurationDays,
					tokenResp.Data.ExpiresAt,
				)
			}

			_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Content: successMsg,
				Flags:   discordgo.MessageFlagsEphemeral,
			})
		}
	}
}

func MessageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
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

func RevokeMemberRole(discordID string) error {
	if Session == nil {
		return fmt.Errorf("bot session belum terhubung")
	}

	rolesToRemove := []string{}
	for _, r := range RoleByPlanTier {
		if r != "" {
			rolesToRemove = append(rolesToRemove, r)
		}
	}
	if DefaultMemberRoleID != "" {
		rolesToRemove = append(rolesToRemove, DefaultMemberRoleID)
	}

	guilds := []string{}
	if ServerGuildID != "" {
		guilds = append(guilds, ServerGuildID)
	} else if Session.State != nil && len(Session.State.Guilds) > 0 {
		for _, g := range Session.State.Guilds {
			guilds = append(guilds, g.ID)
		}
	}

	for _, gID := range guilds {
		for _, roleID := range rolesToRemove {
			err := Session.GuildMemberRoleRemove(gID, discordID, roleID)
			if err != nil {
				log.Printf("Gagal mencabut role %s dari user %s di guild %s: %v\n", roleID, discordID, gID, err)
			} else {
				log.Printf("Role %s berhasil dicabut dari user %s di guild %s\n", roleID, discordID, gID)
			}
		}

		if FreeRoleID != "" {
			err := Session.GuildMemberRoleAdd(gID, discordID, FreeRoleID)
			if err != nil {
				log.Printf("Gagal menambahkan role FREE %s ke user %s: %v\n", FreeRoleID, discordID, err)
			} else {
				log.Printf("Role FREE %s berhasil ditambahkan ke user %s\n", FreeRoleID, discordID)
			}
		}
	}

	return nil
}

