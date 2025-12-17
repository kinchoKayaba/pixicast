package twitch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type Client struct {
	clientID     string
	clientSecret string
	accessToken  string
	tokenExpiry  time.Time
	httpClient   *http.Client
}

type TwitchUser struct {
	ID              string `json:"id"`
	Login           string `json:"login"`
	DisplayName     string `json:"display_name"`
	ProfileImageURL string `json:"profile_image_url"`
}

type TwitchVideo struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	UserLogin   string    `json:"user_login"`
	UserName    string    `json:"user_name"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	PublishedAt time.Time `json:"published_at"`
	URL         string    `json:"url"`
	ThumbnailURL string   `json:"thumbnail_url"`
	Viewable    string    `json:"viewable"`
	ViewCount   int       `json:"view_count"`
	Type        string    `json:"type"`
	Duration    string    `json:"duration"`
}

func NewClient() *Client {
	return &Client{
		clientID:     os.Getenv("TWITCH_CLIENT_ID"),
		clientSecret: os.Getenv("TWITCH_CLIENT_SECRET"),
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) ensureAccessToken(ctx context.Context) error {
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		return nil
	}

	tokenURL := "https://id.twitch.tv/oauth2/token"
	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create token request: %w", err)
	}

	q := req.URL.Query()
	q.Add("client_id", c.clientID)
	q.Add("client_secret", c.clientSecret)
	q.Add("grant_type", "client_credentials")
	req.URL.RawQuery = q.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to request token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token request failed: %s, body: %s", resp.Status, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("failed to decode token response: %w", err)
	}

	c.accessToken = tokenResp.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	log.Printf("✅ Twitch access token acquired (expires in %d seconds)", tokenResp.ExpiresIn)
	return nil
}

func (c *Client) GetUserByLogin(ctx context.Context, login string) (*TwitchUser, error) {
	if err := c.ensureAccessToken(ctx); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://api.twitch.tv/helix/users?login=%s", login)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Client-ID", c.clientID)
	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("user request failed: %s, body: %s", resp.Status, string(body))
	}

	var usersResp struct {
		Data []TwitchUser `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&usersResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(usersResp.Data) == 0 {
		return nil, fmt.Errorf("user not found: %s", login)
	}

	return &usersResp.Data[0], nil
}

func (c *Client) GetVideos(ctx context.Context, userID string, first int) ([]TwitchVideo, error) {
	if err := c.ensureAccessToken(ctx); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://api.twitch.tv/helix/videos?user_id=%s&first=%d", userID, first)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Client-ID", c.clientID)
	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get videos: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("videos request failed: %s, body: %s", resp.Status, string(body))
	}

	var videosResp struct {
		Data []TwitchVideo `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&videosResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return videosResp.Data, nil
}

