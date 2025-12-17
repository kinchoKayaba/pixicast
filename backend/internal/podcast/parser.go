package podcast

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
	}
	if feed.Image != nil {
		podcastFeed.ImageURL = feed.Image.URL
	}

	var episodes []PodcastEpisode
	for _, item := range feed.Items {
		episode := PodcastEpisode{
			GUID:        item.GUID,
			Title:       item.Title,
			Description: item.Description,
			URL:         item.Link,
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

