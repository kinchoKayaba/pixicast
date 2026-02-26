package podcast

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/mmcdole/gofeed"
)

type Client struct {
	parser     *gofeed.Parser
	httpClient *http.Client
}

type PodcastFeed struct {
	Title       string
	Description string
	ImageURL    string
	FeedURL     string
	Link        string // Podcast全体のページURL
	AppleID     string // Apple Podcasts ID（id1234567890）
}

type PodcastEpisode struct {
	GUID        string
	Title       string
	Description string
	PublishedAt time.Time
	URL         string
	ImageURL    string
	Duration    string
}

func NewClient() *Client {
	return &Client{
		parser:     gofeed.NewParser(),
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// PodcastSearchResult はiTunes Search APIの検索結果
type PodcastSearchResult struct {
	CollectionID   int64  `json:"collectionId"`
	TrackName      string `json:"trackName"`
	ArtistName     string `json:"artistName"`
	ArtworkURL     string `json:"artworkUrl600"`
	FeedURL        string `json:"feedUrl"`
	TrackCount     int    `json:"trackCount"`
	CollectionName string `json:"collectionName"`
}

// SearchPodcasts はiTunes Search APIでポッドキャストを検索
func (c *Client) SearchPodcasts(ctx context.Context, term string, limit int) ([]PodcastSearchResult, error) {
	if limit <= 0 {
		limit = 10
	}

	apiURL := fmt.Sprintf("https://itunes.apple.com/search?term=%s&media=podcast&country=jp&limit=%d",
		url.QueryEscape(term), limit)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create iTunes search request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to search podcasts: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("iTunes search returned status %d", resp.StatusCode)
	}

	var result struct {
		ResultCount int                   `json:"resultCount"`
		Results     []PodcastSearchResult `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode iTunes search response: %w", err)
	}

	return result.Results, nil
}

// ResolveFeedURL はApple PodcastsのURLから実際のRSSフィードURLを取得
func (c *Client) ResolveFeedURL(ctx context.Context, input string) (string, error) {
	// すでにRSSフィードURLの場合はそのまま返す
	if regexp.MustCompile(`^https?://.*\.(xml|rss)`).MatchString(input) {
		return input, nil
	}

	// Apple Podcasts URLからIDを抽出
	re := regexp.MustCompile(`id(\d+)`)
	matches := re.FindStringSubmatch(input)
	if len(matches) < 2 {
		// IDが見つからない場合は、入力をそのまま返す（RSSフィードURLかもしれない）
		return input, nil
	}

	podcastID := matches[1]

	// iTunes Search APIでRSSフィードURLを取得
	apiURL := fmt.Sprintf("https://itunes.apple.com/lookup?id=%s&entity=podcast", podcastID)
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch from iTunes API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("iTunes API returned status %d", resp.StatusCode)
	}

	var result struct {
		ResultCount int `json:"resultCount"`
		Results     []struct {
			FeedURL string `json:"feedUrl"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode iTunes API response: %w", err)
	}

	if result.ResultCount == 0 || len(result.Results) == 0 {
		return "", fmt.Errorf("podcast not found")
	}

	feedURL := result.Results[0].FeedURL
	if feedURL == "" {
		return "", fmt.Errorf("feed URL not found in iTunes API response")
	}

	return feedURL, nil
}

func (c *Client) ParseFeed(ctx context.Context, feedURL string) (*PodcastFeed, []PodcastEpisode, error) {
	feed, err := c.parser.ParseURLWithContext(feedURL, ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse feed: %w", err)
	}

	podcastFeed := &PodcastFeed{
		Title:       feed.Title,
		Description: feed.Description,
		FeedURL:     feedURL,
		Link:        feed.Link, // Podcast全体のページURL
	}
	if feed.Image != nil {
		podcastFeed.ImageURL = feed.Image.URL
	}
	
	// Apple Podcasts IDを取得（RSS feedのiTunes拡張から）
	if feed.ITunesExt != nil {
		// ITunesExtからApple IDを検索
		// feedURLからも検索可能
		c.extractAppleID(feed, podcastFeed)
	}

	var episodes []PodcastEpisode
	for _, item := range feed.Items {
		// URLを取得: 
		// 1. item.Link（エピソードページ）
		// 2. feed.Link（Podcast全体のページ）
		// 3. Enclosure URL（音声ファイル - 最終手段）
		episodeURL := item.Link
		if episodeURL == "" {
			episodeURL = feed.Link // Podcast全体のページを使用
		}
		if episodeURL == "" && len(item.Enclosures) > 0 {
			episodeURL = item.Enclosures[0].URL // 音声ファイルURL（最終手段）
		}
		
		episode := PodcastEpisode{
			GUID:        item.GUID,
			Title:       item.Title,
			Description: item.Description,
			URL:         episodeURL,
		}

		if item.PublishedParsed != nil {
			episode.PublishedAt = *item.PublishedParsed
		}

		if item.Image != nil && item.Image.URL != "" {
			episode.ImageURL = item.Image.URL
		} else if feed.Image != nil {
			episode.ImageURL = feed.Image.URL
		}

		if item.ITunesExt != nil && item.ITunesExt.Duration != "" {
			episode.Duration = item.ITunesExt.Duration
		}

		episodes = append(episodes, episode)
	}

	return podcastFeed, episodes, nil
}

// extractAppleID は Feed URLからApple Podcasts IDを取得
func (c *Client) extractAppleID(feed *gofeed.Feed, podcastFeed *PodcastFeed) {
	// iTunes Search APIでfeed URLからApple IDを検索
	if podcastFeed.FeedURL == "" {
		return
	}
	
	// iTunes Search API: /search?entity=podcast&attribute=feedUrl&term={feedURL}
	searchURL := fmt.Sprintf("https://itunes.apple.com/search?entity=podcast&attribute=feedUrl&term=%s", podcastFeed.FeedURL)
	
	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return
	}
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	
	var result struct {
		Results []struct {
			CollectionID int64 `json:"collectionId"`
		} `json:"results"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return
	}
	
	if len(result.Results) > 0 && result.Results[0].CollectionID > 0 {
		podcastFeed.AppleID = fmt.Sprintf("id%d", result.Results[0].CollectionID)
	}
}

