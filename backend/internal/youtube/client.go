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
	call := c.service.Videos.List([]string{"snippet", "liveStreamingDetails", "statistics", "contentDetails"})
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

// GetVideosDetails は複数の動画の詳細情報をバッチで取得
func (c *Client) GetVideosDetails(ctx context.Context, videoIDs []string) ([]*youtube.Video, error) {
	if len(videoIDs) == 0 {
		return []*youtube.Video{}, nil
	}

	// YouTube APIは最大50件までバッチ取得可能
	const batchSize = 50
	var allVideos []*youtube.Video

	for i := 0; i < len(videoIDs); i += batchSize {
		end := i + batchSize
		if end > len(videoIDs) {
			end = len(videoIDs)
		}
		batch := videoIDs[i:end]

		call := c.service.Videos.List([]string{"snippet", "contentDetails", "liveStreamingDetails", "statistics"})
		call = call.Id(batch...)

		response, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("failed to get videos details: %v", err)
		}

		allVideos = append(allVideos, response.Items...)
	}

	return allVideos, nil
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
	return c.GetChannelVideosSince(ctx, channelID, maxResults, "")
}

// GetChannelVideosSince は指定日時以降のチャンネル動画を全て取得（ページング対応）
func (c *Client) GetChannelVideosSince(ctx context.Context, channelID string, maxResults int64, publishedAfter string) ([]*youtube.SearchResult, error) {
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
	
	// 2. プレイリストから動画を取得（ページング対応）
	var allResults []*youtube.SearchResult
	pageToken := ""
	
	for {
		playlistCall := c.service.PlaylistItems.List([]string{"snippet", "contentDetails"})
		playlistCall = playlistCall.PlaylistId(uploadsPlaylistID)
		playlistCall = playlistCall.MaxResults(50) // API最大値
		
		if pageToken != "" {
			playlistCall = playlistCall.PageToken(pageToken)
		}
		
		playlistResponse, err := playlistCall.Do()
		if err != nil {
			return nil, fmt.Errorf("failed to get playlist items: %v", err)
		}
		
		// PlaylistItemをSearchResult形式に変換
		for _, item := range playlistResponse.Items {
			// publishedAfter フィルタリング
			if publishedAfter != "" && item.Snippet.PublishedAt < publishedAfter {
				// これ以降は全て古い動画なので終了
				return allResults, nil
			}
			
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
			allResults = append(allResults, result)
		}
		
		// maxResults制限チェック
		if maxResults > 0 && int64(len(allResults)) >= maxResults {
			return allResults[:maxResults], nil
		}
		
		// 次のページがあるか確認
		if playlistResponse.NextPageToken == "" {
			break
		}
		pageToken = playlistResponse.NextPageToken
	}
	
	return allResults, nil
}

// ChannelSearchResult はチャンネル検索結果
type ChannelSearchResult struct {
	ChannelID       string
	Handle          string
	DisplayName     string
	ThumbnailURL    string
	SubscriberCount int64
}

// SearchChannels はキーワードでチャンネルを検索
// search.list (100 units) + channels.list (1 unit) で enrichment
func (c *Client) SearchChannels(ctx context.Context, query string, maxResults int64) ([]ChannelSearchResult, error) {
	if maxResults <= 0 {
		maxResults = 10
	}

	// search.list でチャンネル検索 (100 units)
	call := c.service.Search.List([]string{"snippet"})
	call = call.Q(query)
	call = call.Type("channel")
	call = call.MaxResults(maxResults)
	call = call.RegionCode("JP")

	response, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to search channels: %v", err)
	}

	if len(response.Items) == 0 {
		return nil, nil
	}

	// チャンネルIDを収集して channels.list で enrichment (1 unit)
	channelIDs := make([]string, 0, len(response.Items))
	for _, item := range response.Items {
		if item.Id != nil && item.Id.ChannelId != "" {
			channelIDs = append(channelIDs, item.Id.ChannelId)
		}
	}

	// channels.list で subscriberCount, handle を取得
	enrichCall := c.service.Channels.List([]string{"snippet", "statistics"})
	enrichCall = enrichCall.Id(channelIDs...)
	enrichResp, err := enrichCall.Do()
	if err != nil {
		// enrichment 失敗時は search.list の結果のみで返す
		var results []ChannelSearchResult
		for _, item := range response.Items {
			if item.Id == nil || item.Id.ChannelId == "" {
				continue
			}
			thumbnailURL := ""
			if item.Snippet.Thumbnails != nil && item.Snippet.Thumbnails.High != nil {
				thumbnailURL = item.Snippet.Thumbnails.High.Url
			}
			results = append(results, ChannelSearchResult{
				ChannelID:    item.Id.ChannelId,
				DisplayName:  item.Snippet.Title,
				ThumbnailURL: thumbnailURL,
			})
		}
		return results, nil
	}

	// enrichment データをマップ化
	channelMap := make(map[string]*youtube.Channel)
	for _, ch := range enrichResp.Items {
		channelMap[ch.Id] = ch
	}

	var results []ChannelSearchResult
	for _, item := range response.Items {
		if item.Id == nil || item.Id.ChannelId == "" {
			continue
		}

		result := ChannelSearchResult{
			ChannelID:   item.Id.ChannelId,
			DisplayName: item.Snippet.Title,
		}

		// サムネイル
		if item.Snippet.Thumbnails != nil && item.Snippet.Thumbnails.High != nil {
			result.ThumbnailURL = item.Snippet.Thumbnails.High.Url
		}

		// enrichment データがあれば使用
		if ch, ok := channelMap[item.Id.ChannelId]; ok {
			if ch.Statistics != nil {
				result.SubscriberCount = int64(ch.Statistics.SubscriberCount)
			}
			if ch.Snippet != nil && ch.Snippet.CustomUrl != "" {
				handle := ch.Snippet.CustomUrl
				if len(handle) > 0 && handle[0] == '@' {
					handle = handle[1:]
				}
				result.Handle = handle
			}
		}

		results = append(results, result)
	}

	return results, nil
}

// ChannelDetails はチャンネルの詳細情報
type ChannelDetails struct {
	ChannelID         string
	Handle            string
	DisplayName       string
	ThumbnailURL      string
	UploadsPlaylistID string
}

// ResolveHandle は @handle からチャンネルIDを解決
// handle は "@junchannel" の形式（先頭の@は含まない）
func (c *Client) ResolveHandle(ctx context.Context, handle string) (string, error) {
	// Search APIでチャンネルを検索
	// 注: YouTube Data API v3には直接handle→channelIDの変換APIがないため、
	// forHandle または forUsername パラメータを使う
	call := c.service.Channels.List([]string{"id"})
	call = call.ForHandle(handle)
	
	response, err := call.Do()
	if err != nil {
		return "", fmt.Errorf("failed to resolve handle: %v", err)
	}
	
	if len(response.Items) == 0 {
		return "", fmt.Errorf("channel not found for handle: @%s", handle)
	}
	
	return response.Items[0].Id, nil
}

// GetChannelDetails はチャンネルの詳細情報を取得
// channelID は UCxxx... の形式
func (c *Client) GetChannelDetails(ctx context.Context, channelID string) (*ChannelDetails, error) {
	call := c.service.Channels.List([]string{"id", "snippet", "contentDetails"})
	call = call.Id(channelID)
	
	response, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get channel details: %v", err)
	}
	
	if len(response.Items) == 0 {
		return nil, fmt.Errorf("channel not found: %s", channelID)
	}
	
	channel := response.Items[0]
	
	// サムネイルURL取得（優先順位: high > medium > default）
	thumbnailURL := ""
	if channel.Snippet.Thumbnails != nil {
		if channel.Snippet.Thumbnails.High != nil {
			thumbnailURL = channel.Snippet.Thumbnails.High.Url
		} else if channel.Snippet.Thumbnails.Medium != nil {
			thumbnailURL = channel.Snippet.Thumbnails.Medium.Url
		} else if channel.Snippet.Thumbnails.Default != nil {
			thumbnailURL = channel.Snippet.Thumbnails.Default.Url
		}
	}
	
	// CustomUrl が @handle の形式の場合がある（@を除去）
	handle := ""
	if channel.Snippet.CustomUrl != "" {
		handle = channel.Snippet.CustomUrl
		if len(handle) > 0 && handle[0] == '@' {
			handle = handle[1:]
		}
	}
	
	return &ChannelDetails{
		ChannelID:         channel.Id,
		Handle:            handle,
		DisplayName:       channel.Snippet.Title,
		ThumbnailURL:      thumbnailURL,
		UploadsPlaylistID: channel.ContentDetails.RelatedPlaylists.Uploads,
	}, nil
}

