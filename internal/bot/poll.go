package bot

import (
	"agent-bot/internal/database"
	"agent-bot/internal/model"
	"agent-bot/internal/service"
	"context"
	"log"
	"time"
)

func PollUserStatuses() {
	if !service.LaravelEnabled || !service.IsLaravelConnected() {
		return
	}

	if database.DB == nil || Session == nil {
		return
	}

	var users []model.User
	if err := database.DB.Find(&users).Error; err != nil {
		log.Printf("Gagal mengambil data user dari DB: %v\n", err)
		return
	}

	for _, u := range users {
		if u.DiscordID == "" {
			continue
		}

		statusResp, err := service.CheckUserStatus(u.DiscordID)
		if err != nil {
			log.Printf("[Poller] Gagal memeriksa status user %s (koneksi Laravel bermasalah): %v\n", u.DiscordID, err)
			return
		}

		if !statusResp.Exists {
			continue
		}

		if statusResp.IsRevoked || statusResp.Status == "REVOKED" || !statusResp.HasActiveLicense {
			if u.Status != "REVOKED" {
				log.Printf("User %s (%s) status REVOKED di Nusa Admin. Menurunkan role...\n", u.Username, u.DiscordID)
				if err := RevokeMemberRole(u.DiscordID); err != nil {
					log.Printf("Gagal menurunkan role user %s: %v\n", u.DiscordID, err)
				} else {
					u.Status = "REVOKED"
					u.UpdatedAt = time.Now().Unix()
					database.DB.Save(&u)
					log.Printf("Role user %s berhasil diturunkan ke FREE dan status DB diupdate ke REVOKED\n", u.DiscordID)
				}
			}
		} else if statusResp.HasActiveLicense && statusResp.Status == "ACTIVE" {
			if u.Status != "ACTIVE" {
				u.Status = "ACTIVE"
				u.UpdatedAt = time.Now().Unix()
				database.DB.Save(&u)
			}
		}
	}
}

func StartStatusPoller(ctx context.Context, interval time.Duration) {
	go func() {
		time.Sleep(3 * time.Second)
		PollUserStatuses()
	}()

	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				PollUserStatuses()
			}
		}
	}()
}
