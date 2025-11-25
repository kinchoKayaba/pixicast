package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"connectrpc.com/connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	// 生成されたコードのインポート
	pixicastv1 "github.com/kinchoKayaba/pixicast/backend/gen/pixicast/v1"
	"github.com/kinchoKayaba/pixicast/backend/gen/pixicast/v1/pixicastv1connect"
)

type TimelineServer struct{}

// ★ここが新機能: タイムライン取得
// フロントエンドからの「GetTimeline」リクエストを受け取る部分
func (s *TimelineServer) GetTimeline(
	ctx context.Context,
	req *connect.Request[pixicastv1.GetTimelineRequest],
) (*connect.Response[pixicastv1.GetTimelineResponse], error) {
	log.Printf("GetTimeline called for date: %s", req.Msg.Date)

	// モックデータ（偽物）を作る
	// 実際はここでDBから取得するようになります
	mockPrograms := []*pixicastv1.Program{
		{
			Id:           "1",
			Title:        "19時までマリカワールド",
			StartAt:      "2025-11-25T17:00:00+09:00",
			EndAt:        "2025-11-25T19:00:00+09:00",
			PlatformName: "YouTube",
			IsLive:       true,
			ImageUrl:     "https://placehold.jp/150x150.png",
		},
		{
			Id:           "2",
			Title:        "12時までウイポ",
			StartAt:      "2025-11-25T22:00:00+09:00",
			EndAt:        "2025-11-25T24:00:00+09:00",
			PlatformName: "Twitch",
			IsLive:       false,
			ImageUrl:     "https://placehold.jp/150x150.png",
		},
	}

	// レスポンスを返す
	return connect.NewResponse(&pixicastv1.GetTimelineResponse{
		Programs: mockPrograms,
	}), nil
}

func main() {
	// ハンドラー登録
	// NewTimelineServiceHandler を使う
	path, handler := pixicastv1connect.NewTimelineServiceHandler(&TimelineServer{})
	
	mux := http.NewServeMux()
	mux.Handle(path, handler)

	fmt.Println("Starting Pixicast Server (Timeline Mode) on localhost:8080 ...")
	
	// サーバー起動
	err := http.ListenAndServe(
		"localhost:8080",
		h2c.NewHandler(mux, &http2.Server{}),
	)
	if err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}