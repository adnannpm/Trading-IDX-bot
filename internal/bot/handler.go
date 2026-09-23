package bot

import (
	"agent-bot/internal/bot/template"
	"agent-bot/internal/config"
	"agent-bot/internal/database"
	"agent-bot/internal/model"
	"agent-bot/internal/service"
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
)

var RoleByPlanTier = map[string]string{
	"VIP": config.DefaultRoleVIPID,
}

var DefaultMemberRoleID = config.DefaultRoleMemberID

var FreeRoleID = config.DefaultRoleFreeID

var ServerGuildID = ""

func init() {
	SyncConfig()
}

func SyncConfig() {
	cfg := config.Get()
	RoleByPlanTier = map[string]string{
		"VIP": cfg.RoleVIPID,
	}
	DefaultMemberRoleID = cfg.RoleDefaultID
	FreeRoleID = cfg.RoleFreeID
	if cfg.DiscordGuildID != "" {
		ServerGuildID = cfg.DiscordGuildID
	}
}

func ReadyHandler(s *discordgo.Session, r *discordgo.Ready) {
	log.Printf("Bot logged in as %s#%s (%s)\n", r.User.Username, r.User.Discriminator, r.User.ID)
}

func InteractionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		handleCommand(s, i)
	case discordgo.InteractionApplicationCommandAutocomplete:
		handleAutocomplete(s, i)
	case discordgo.InteractionMessageComponent:
		if handleRefresh(s, i) {
			return
		}
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

			log.Printf("User %s (%s) submitted verification\n", discordUsername, user.ID)

			if !service.LaravelEnabled {
				_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
					Content: "⚠️ Layanan validasi token dinonaktifkan saat ini.",
					Flags:   discordgo.MessageFlagsEphemeral,
				})
				return
			}

			req := service.TokenRequest{
				Token:           userInput,
				DiscordID:       user.ID,
				DiscordUsername: discordUsername,
				AvatarURL:       avatarURL,
			}

			tokenResp, err := service.ValidateToken("/tokens/validate", req)
			if err != nil {
				log.Printf("Error validating token via Laravel: %v\n", err)
				_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
					Content: "⚠️ Layanan Nusa Admin (Laravel) sedang offline atau tidak dapat dihubungi. Silakan coba beberapa saat lagi.",
					Flags:   discordgo.MessageFlagsEphemeral,
				})
				return
			}

			if tokenResp == nil || !tokenResp.Valid {
				msg := "❌ Token yang Anda masukkan tidak valid atau kadaluarsa."
				if tokenResp != nil && tokenResp.Message != "" {
					msg = fmt.Sprintf("❌ %s", tokenResp.Message)
					if tokenResp.ClaimedBy != "" {
						msg += fmt.Sprintf("\n*(Sudah diklaim oleh: `%s`)*", tokenResp.ClaimedBy)
					}
				}

				_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
					Content: msg,
					Flags:   discordgo.MessageFlagsEphemeral,
				})
				return
			}

			cfg := config.Get()
			targetRoleID := cfg.RoleDefaultID
			if tokenResp.Data != nil && tokenResp.Data.PlanTier != "" {
				if roleID := cfg.GetRoleForTier(tokenResp.Data.PlanTier); roleID != "" {
					targetRoleID = roleID
				} else if roleID, ok := RoleByPlanTier[tokenResp.Data.PlanTier]; ok && roleID != "" {
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

	cfg := config.Get()
	rolesToRemove := []string{}
	addUnique := func(id string) {
		if id == "" {
			return
		}
		for _, r := range rolesToRemove {
			if r == id {
				return
			}
		}
		rolesToRemove = append(rolesToRemove, id)
	}

	for _, r := range RoleByPlanTier {
		addUnique(r)
	}
	addUnique(cfg.RoleVIPID)
	addUnique(cfg.RoleDefaultID)
	addUnique(DefaultMemberRoleID)

	guilds := []string{}
	if ServerGuildID != "" {
		guilds = append(guilds, ServerGuildID)
	} else if Session.State != nil && len(Session.State.Guilds) > 0 {
		for _, g := range Session.State.Guilds {
			guilds = append(guilds, g.ID)
		}
	}

	freeRole := cfg.RoleFreeID
	if freeRole == "" {
		freeRole = FreeRoleID
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

		if freeRole != "" {
			err := Session.GuildMemberRoleAdd(gID, discordID, freeRole)
			if err != nil {
				log.Printf("Gagal menambahkan role FREE %s ke user %s: %v\n", freeRole, discordID, err)
			} else {
				log.Printf("Role FREE %s berhasil ditambahkan ke user %s\n", freeRole, discordID)
			}
		}
	}

	return nil
}
