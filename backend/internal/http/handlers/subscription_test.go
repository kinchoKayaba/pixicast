package handlers

import (
	"testing"
)

// TestNormalizeInput は入力正規化のテスト
func TestNormalizeInput(t *testing.T) {
	h := &SubscriptionHandler{}

	tests := []struct {
		name           string
		input          string
		wantChannelID  string
		wantHandle     string
		wantErr        bool
	}{
		{
			name:          "Channel ID",
			input:         "UCxxxxxxxxxxxx",
			wantChannelID: "UCxxxxxxxxxxxx",
			wantHandle:    "",
			wantErr:       false,
		},
		{
			name:          "@handle",
			input:         "@junchannel",
			wantChannelID: "",
			wantHandle:    "junchannel",
			wantErr:       false,
		},
		{
			name:          "URL with @handle",
			input:         "https://www.youtube.com/@junchannel",
			wantChannelID: "",
			wantHandle:    "junchannel",
			wantErr:       false,
		},
		{
			name:          "URL with @handle and path",
			input:         "https://www.youtube.com/@junchannel/featured",
			wantChannelID: "",
			wantHandle:    "junchannel",
			wantErr:       false,
		},
		{
			name:          "URL with channel ID",
			input:         "https://www.youtube.com/channel/UCxxxxxxxxxxxx",
			wantChannelID: "UCxxxxxxxxxxxx",
			wantHandle:    "",
			wantErr:       false,
		},
		{
			name:          "URL with channel ID and path",
			input:         "https://www.youtube.com/channel/UCxxxxxxxxxxxx/featured",
			wantChannelID: "UCxxxxxxxxxxxx",
			wantHandle:    "",
			wantErr:       false,
		},
		{
			name:    "Invalid input",
			input:   "invalid",
			wantErr: true,
		},
		{
			name:    "Empty @handle",
			input:   "@",
			wantErr: true,
		},
		{
			name:    "Non-YouTube URL",
			input:   "https://www.example.com/@test",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channelID, handle, err := h.normalizeInput(tt.input)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("normalizeInput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr {
				if channelID != tt.wantChannelID {
					t.Errorf("normalizeInput() channelID = %v, want %v", channelID, tt.wantChannelID)
				}
				if handle != tt.wantHandle {
					t.Errorf("normalizeInput() handle = %v, want %v", handle, tt.wantHandle)
				}
			}
		})
	}
}

// TestParseYouTubeURL はYouTube URLパースのテスト
func TestParseYouTubeURL(t *testing.T) {
	h := &SubscriptionHandler{}

	tests := []struct {
		name           string
		url            string
		wantChannelID  string
		wantHandle     string
		wantErr        bool
	}{
		{
			name:          "Channel ID URL",
			url:           "https://www.youtube.com/channel/UCxxxxxxxxxxxx",
			wantChannelID: "UCxxxxxxxxxxxx",
			wantHandle:    "",
			wantErr:       false,
		},
		{
			name:          "Handle URL",
			url:           "https://www.youtube.com/@junchannel",
			wantChannelID: "",
			wantHandle:    "junchannel",
			wantErr:       false,
		},
		{
			name:          "Handle URL with path",
			url:           "https://www.youtube.com/@junchannel/videos",
			wantChannelID: "",
			wantHandle:    "junchannel",
			wantErr:       false,
		},
		{
			name:    "Invalid URL",
			url:     "not-a-url",
			wantErr: true,
		},
		{
			name:    "Non-YouTube domain",
			url:     "https://www.example.com/test",
			wantErr: true,
		},
		{
			name:    "YouTube URL without channel info",
			url:     "https://www.youtube.com/watch?v=xxxxx",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channelID, handle, err := h.parseYouTubeURL(tt.url)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("parseYouTubeURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr {
				if channelID != tt.wantChannelID {
					t.Errorf("parseYouTubeURL() channelID = %v, want %v", channelID, tt.wantChannelID)
				}
				if handle != tt.wantHandle {
					t.Errorf("parseYouTubeURL() handle = %v, want %v", handle, tt.wantHandle)
				}
			}
		})
	}
}

