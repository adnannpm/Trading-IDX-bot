package api

import (
	"agent-bot/internal/bot"
	"agent-bot/internal/database"
	"agent-bot/internal/model"
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type RevokeRequest struct {
	DiscordID string `json:"discord_id" binding:"required"`
	Reason    string `json:"reason"`
}

type RevokeData struct {
	DiscordID       string `json:"discord_id"`
	DiscordUsername string `json:"discord_username"`
	UserToken       string `json:"user_token"`
	Status          string `json:"status"`
	Role            string `json:"role"`
	RevokedAt       string `json:"revoked_at"`
}

type RevokeResponse struct {
	Success bool       `json:"success"`
	Status  string     `json:"status"`
	Role    string     `json:"role"`
	Message string     `json:"message"`
	Data    RevokeData `json:"data"`
}

func SetupRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	apiGroup := r.Group("/api/v1/agent")
	{
		apiGroup.POST("/tokens/revoke", handleRevokeToken)
	}

	return r
}

func handleRevokeToken(c *gin.Context) {
	var req RevokeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Format request tidak valid: " + err.Error(),
		})
		return
	}

	log.Printf("Menerima job revoke dari Laravel: Discord ID %s, Alasan: %s\n", req.DiscordID, req.Reason)

	err := bot.RevokeMemberRole(req.DiscordID)
	if err != nil {
		log.Printf("Gagal mencabut role Discord untuk user %s: %v\n", req.DiscordID, err)
	}

	var dbUser model.User
	userToken := "NUSA-XXXX-XXXX"
	username := ""

	if database.DB != nil {
		if err := database.DB.Where("discord_id = ?", req.DiscordID).First(&dbUser).Error; err == nil {
			if dbUser.Token != "" {
				userToken = dbUser.Token
			}
			username = dbUser.Username
			dbUser.Status = "REVOKED"
			dbUser.UpdatedAt = time.Now().Unix()
			database.DB.Save(&dbUser)
		}
	}

	if username == "" && bot.Session != nil {
		if u, err := bot.Session.User(req.DiscordID); err == nil && u != nil {
			if u.Discriminator != "0" && u.Discriminator != "" {
				username = fmt.Sprintf("%s#%s", u.Username, u.Discriminator)
			} else {
				username = u.Username
			}
		}
	}

	c.JSON(http.StatusOK, RevokeResponse{
		Success: true,
		Status:  "REVOKED",
		Role:    "FREE",
		Message: "Lisensi token berhasil dicabut. Role Discord diturunkan ke FREE.",
		Data: RevokeData{
			DiscordID:       req.DiscordID,
			DiscordUsername: username,
			UserToken:       userToken,
			Status:          "revoked",
			Role:            "FREE",
			RevokedAt:       time.Now().Format("2006-01-02T15:04:05-07:00"),
		},
	})
}

func StartServer(ctx context.Context, addr string) {
	router := SetupRouter()
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		log.Printf("API Server aktif di %s (siap menerima job dari Laravel)\n", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Gagal menjalankan API server: %v\n", err)
		}
	}()

	<-ctx.Done()
	log.Println("Mematikan API server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}
