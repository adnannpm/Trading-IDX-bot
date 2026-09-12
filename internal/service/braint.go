package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const (
	baseURL = "http://127.0.0.1:8000/api/v1/agent"
	PingInterval = 20 * time.Second
	PollInterval = 1 * time.Minute
)

type HearbeatPayload struct {
	AgentName 		string `json:"agent_name"`
	Version	 		string `json:"version"`
	Interval	 	string  `json:"interval"`
	Status   		string `json:"status"`
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
	data, _ := json.Marshal(payload)
	resp, err := http.Post(baseURL+endpoint, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func ValidateToken(endpoint string, req TokenRequest) (*TokenResponse, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(baseURL+endpoint, "application/json", bytes.NewBuffer(data))
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
	reqURL := fmt.Sprintf("%s/discord-users/status?discord_id=%s", baseURL, url.QueryEscape(discordID))
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