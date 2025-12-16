package youtube

import (
	"context"
	"fmt"

	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

// Client は YouTube Data API v3 のクライアント
type Client struct {
	service *youtube.Service
}

// NewClient は YouTube クライアントを作成
func NewClient(apiKey string) (*Client, error) {
	ctx := context.Background()
	service, err := youtube.NewService(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create YouTube client: %v", err)
	}

	return &Client{
		service: service,
	}, nil
}

// SearchLiveStreams はライブ配信を検索
// query: 検索キーワード
// maxResults: 取得する最大件数（最大50）
func (c *Client) SearchLiveStreams(ctx context.Context, query string, maxResults int64) ([]*youtube.SearchResult, error) {
	call := c.service.Search.List([]string{"snippet"})
	call = call.Q(query)
	call = call.Type("video")
	call = call.EventType("live") // ライブ配信のみ
	call = call.MaxResults(maxResults)

	response, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to search live streams: %v", err)
	}

	return response.Items, nil
}

// GetVideoDetails は動画の詳細情報を取得
func (c *Client) GetVideoDetails(ctx context.Context, videoID string) (*youtube.Video, error) {
	call := c.service.Videos.List([]string{"snippet", "liveStreamingDetails", "statistics"})
	call = call.Id(videoID)

	response, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get video details: %v", err)
	}

	if len(response.Items) == 0 {
		return nil, fmt.Errorf("video not found: %s", videoID)
	}

	return response.Items[0], nil
}

// GetChannelInfo はチャンネル情報を取得
func (c *Client) GetChannelInfo(ctx context.Context, channelID string) (*youtube.Channel, error) {
	call := c.service.Channels.List([]string{"snippet", "statistics"})
	call = call.Id(channelID)

	response, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get channel info: %v", err)
	}

	if len(response.Items) == 0 {
		return nil, fmt.Errorf("channel not found: %s", channelID)
	}

	return response.Items[0], nil
}

// SearchUpcomingStreams は今後予定されているライブ配信を検索
func (c *Client) SearchUpcomingStreams(ctx context.Context, query string, maxResults int64) ([]*youtube.SearchResult, error) {
	call := c.service.Search.List([]string{"snippet"})
	call = call.Q(query)
	call = call.Type("video")
	call = call.EventType("upcoming") // 今後予定されている配信
	call = call.MaxResults(maxResults)

	response, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to search upcoming streams: %v", err)
	}

	return response.Items, nil
}

// GetChannelVideos はチャンネルの最新動画を取得
// PlaylistItems APIを使用してチャンネルのアップロード動画を正確に取得
func (c *Client) GetChannelVideos(ctx context.Context, channelID string, maxResults int64) ([]*youtube.SearchResult, error) {
	// 1. チャンネル情報を取得して、uploadsプレイリストIDを取得
	channelCall := c.service.Channels.List([]string{"contentDetails"})
	channelCall = channelCall.Id(channelID)
	
	channelResponse, err := channelCall.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get channel info: %v", err)
	}
	
	if len(channelResponse.Items) == 0 {
		return nil, fmt.Errorf("channel not found: %s", channelID)
	}
	
	// uploadsプレイリストID（UUから始まる）
	uploadsPlaylistID := channelResponse.Items[0].ContentDetails.RelatedPlaylists.Uploads
	
	// 2. プレイリストから動画を取得
	playlistCall := c.service.PlaylistItems.List([]string{"snippet", "contentDetails"})
	playlistCall = playlistCall.PlaylistId(uploadsPlaylistID)
	playlistCall = playlistCall.MaxResults(maxResults)
	
	playlistResponse, err := playlistCall.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get playlist items: %v", err)
	}
	
	// PlaylistItemをSearchResult形式に変換
	var results []*youtube.SearchResult
	for _, item := range playlistResponse.Items {
		result := &youtube.SearchResult{
			Id: &youtube.ResourceId{
				Kind:    "youtube#video",
				VideoId: item.ContentDetails.VideoId,
			},
			Snippet: &youtube.SearchResultSnippet{
				ChannelId:            item.Snippet.ChannelId,
				ChannelTitle:         item.Snippet.ChannelTitle,
				Description:          item.Snippet.Description,
				LiveBroadcastContent: "none", // PlaylistItemには含まれないのでデフォルト値
				PublishedAt:          item.Snippet.PublishedAt,
				Thumbnails:           item.Snippet.Thumbnails,
				Title:                item.Snippet.Title,
			},
		}
		results = append(results, result)
	}
	
	return results, nil
}

