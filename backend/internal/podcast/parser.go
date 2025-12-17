package podcast

import (
	"context"
	"fmt"
	"time"

	"github.com/mmcdole/gofeed"
)

type Client struct {
	parser *gofeed.Parser
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
		parser: gofeed.NewParser(),
	}
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

