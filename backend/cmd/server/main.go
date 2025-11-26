package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	// 生成されたコードのインポート
	"github.com/kinchoKayaba/pixicast/backend/db" // ★sqlcが作ったコード
	pixicastv1 "github.com/kinchoKayaba/pixicast/backend/gen/pixicast/v1"
	"github.com/kinchoKayaba/pixicast/backend/gen/pixicast/v1/pixicastv1connect"
)

// サーバー構造体
// 生のDB接続ではなく、sqlcが生成した「Queries」を持ちます
type TimelineServer struct {
	queries *db.Queries
}

// タイムライン取得
func (s *TimelineServer) GetTimeline(
	ctx context.Context,
	req *connect.Request[pixicastv1.GetTimelineRequest],
) (*connect.Response[pixicastv1.GetTimelineResponse], error) {
	log.Printf("GetTimeline called for date: %s", req.Msg.Date)

	// 1. DBからデータを取得 (SQL実行)
	// たったこれだけで "SELECT * FROM programs..." が走ります！
	programsData, err := s.queries.ListPrograms(ctx)
	if err != nil {
		log.Printf("Failed to fetch programs: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database error"))
	}

	// 2. DBの型(db.Program) を gRPCの型(pixicastv1.Program) に変換
	var responsePrograms []*pixicastv1.Program
	for _, p := range programsData {
		// 放送中かどうかの簡易判定 (現在時刻が start と end の間なら true)
		now := time.Now()
		isLive := now.After(p.StartAt.Time) && now.Before(p.EndAt.Time)

		// ImageUrlなどはNULL許容(pgtype.Text)なので、取り出し方に注意
		imageUrl := ""
		if p.ImageUrl.Valid {
			imageUrl = p.ImageUrl.String
		}
		linkUrl := ""
		if p.LinkUrl.Valid {
			linkUrl = p.LinkUrl.String
		}

		responsePrograms = append(responsePrograms, &pixicastv1.Program{
			Id:           p.ID.String(), // UUIDを文字列に
			Title:        p.Title,
			StartAt:      p.StartAt.Time.Format(time.RFC3339), // 時間を文字列に
			EndAt:        p.EndAt.Time.Format(time.RFC3339),
			PlatformName: p.PlatformName,
			ImageUrl:     imageUrl,
			LinkUrl:      linkUrl,
			IsLive:       isLive,
		})
	}

	return connect.NewResponse(&pixicastv1.GetTimelineResponse{
		Programs: responsePrograms,
	}), nil
}

func main() {
	_ = godotenv.Load()

	dbUrl := os.Getenv("DATABASE_URL")
	if dbUrl == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	// DB接続
	pool, err := pgxpool.New(context.Background(), dbUrl)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer pool.Close()

	// 疎通確認
	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	fmt.Println("✅ Connected to CockroachDB successfully!")

	// ★ここがポイント: DB接続を使って sqlc の Queries を作成
	queries := db.New(pool)

	// サーバーに渡す
	server := &TimelineServer{
		queries: queries,
	}

	path, handler := pixicastv1connect.NewTimelineServiceHandler(server)
	mux := http.NewServeMux()
	mux.Handle(path, handler)

	fmt.Println("Starting Pixicast Server (Timeline Mode) on localhost:8080 ...")
	err = http.ListenAndServe(
		"localhost:8080",
		h2c.NewHandler(mux, &http2.Server{}),
	)
	if err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}