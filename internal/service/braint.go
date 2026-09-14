package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
)

const (
	defaultBaseURL = "http://127.0.0.1:8000/api/v1/agent"
	PingInterval   = 20 * time.Second
	PollInterval   = 1 * time.Minute
)

var (
	LaravelEnabled     = true
	isLaravelConnected = false
	connMutex          sync.RWMutex
)

func getBaseURL() string {
	if envURL := os.Getenv("LARAVEL_API_URL"); envURL != "" {
		return envURL
	}
	return defaultBaseURL
}

func IsLaravelConnected() bool {
	connMutex.RLock()
	defer connMutex.RUnlock()
	return isLaravelConnected
}

func SetLaravelConnected(val bool) {
	connMutex.Lock()
	defer connMutex.Unlock()
	isLaravelConnected = val
}

type HearbeatPayload struct {
	AgentName string `json:"agent_name"`
	Version   string `json:"version"`
	Interval  string `json:"interval"`
	Status    string `json:"status"`
}

type TokenRequest struct {
	Token           string `json:"token"`
	DiscordID       string `json:"discord_id"`
	DiscordUsername string `json:"discord_username"`
	AvatarURL       string `json:"avatar_url"`
}

type TokenData struct {
	Token           string `json:"token"`
	ServerToken     string `json:"server_token"`
	PlanTier        string `json:"plan_tier"`
	DurationDays    int    `json:"duration_days"`
	DiscordID       string `json:"discord_id"`
	DiscordUsername string `json:"discord_username"`
	ActivatedAt     string `json:"activated_at"`
	ExpiresAt       string `json:"expires_at"`
}

type TokenResponse struct {
	Valid     bool       `json:"valid"`
	Status    string     `json:"status"`
	Message   string     `json:"message"`
	ClaimedBy string     `json:"claimed_by,omitempty"`
	ClaimedAt string     `json:"claimed_at,omitempty"`
	Data      *TokenData `json:"data,omitempty"`
}

func SendStatus(endpoint string, payload HearbeatPayload) error {
	return SendStatusWithTimeout(endpoint, payload, 5*time.Second)
}

func SendStatusWithTimeout(endpoint string, payload HearbeatPayload, timeout time.Duration) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: timeout}
	resp, err := client.Post(getBaseURL()+endpoint, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("server mengembalikan status %d", resp.StatusCode)
	}

	return nil
}

func StartHeartbeatWorker(ctx context.Context, payload HearbeatPayload) {
	if !LaravelEnabled {
		log.Println("[Nusa Admin] Komunikasi dengan Laravel dinonaktifkan.")
		return
	}

	log.Println("[Nusa Admin] Menghubungkan Agent ke Nusa Admin...")
	if err := SendStatus("/online", payload); err != nil {
		SetLaravelConnected(false)
		log.Printf("[Nusa Admin] Server Laravel belum aktif / tidak terhubung (%v).", err)
		log.Println("[Nusa Admin] Bot tetap berjalan normal. Sistem otomatis menghubungkan dan mengirim status ketika Laravel dijalankan.")
	} else {
		SetLaravelConnected(true)
		log.Println("[Nusa Admin] Terhubung ke Nusa Admin! Status Online berhasil dikirim.")
	}

	checkTicker := time.NewTicker(5 * time.Second)
	defer checkTicker.Stop()

	lastHeartbeat := time.Now()

	for {
		select {
		case <-ctx.Done():
			return
		case <-checkTicker.C:
			if !LaravelEnabled {
				return
			}

			if !IsLaravelConnected() {
				if err := SendStatus("/online", payload); err == nil {
					SetLaravelConnected(true)
					lastHeartbeat = time.Now()
					log.Println("[Nusa Admin] 🎉 Laravel terdeteksi aktif! Status Online berhasil dikirim.")
				}
			} else {
				if time.Since(lastHeartbeat) >= PingInterval {
					if err := SendStatus("/heartbeat", payload); err != nil {
						SetLaravelConnected(false)
						log.Printf("[Nusa Admin] Gagal mengirim heartbeat ke Laravel (%v). Menunggu Laravel aktif kembali...", err)
					} else {
						lastHeartbeat = time.Now()
					}
				}
			}
		}
	}
}

func ShutdownHeartbeat(payload HearbeatPayload) {
	if !LaravelEnabled || !IsLaravelConnected() {
		return
	}

	payload.Status = "Offline"
	log.Println("[Nusa Admin] Mengirim status offline ke Laravel...")
	if err := SendStatusWithTimeout("/offline", payload, 2*time.Second); err != nil {
		log.Printf("[Nusa Admin] Gagal mengirim status offline: %v\n", err)
	} else {
		log.Println("[Nusa Admin] Status offline berhasil dikirim.")
	}
}

func ValidateToken(endpoint string, req TokenRequest) (*TokenResponse, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(getBaseURL()+endpoint, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}

type DiscordUserStatusResponse struct {
	Exists           bool   `json:"exists"`
	DiscordID        string `json:"discord_id"`
	Status           string `json:"status"`
	HasActiveLicense bool   `json:"has_active_license"`
	Role             string `json:"role"`
	IsRevoked        bool   `json:"is_revoked"`
}

func CheckUserStatus(discordID string) (*DiscordUserStatusResponse, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	reqURL := fmt.Sprintf("%s/discord-users/status?discord_id=%s", getBaseURL(), url.QueryEscape(discordID))
	resp, err := client.Get(reqURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var statusResp DiscordUserStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
		return nil, err
	}

	return &statusResp, nil
}