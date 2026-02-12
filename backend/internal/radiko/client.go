package radiko

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// RadikoのAPIエンドポイント
	StationListURL   = "http://radiko.jp/v3/station/list/%s.xml"        // エリアID
	WeeklyProgramURL = "http://radiko.jp/v3/program/station/weekly/%s.xml" // ステーションID
	NowOnAirURL      = "http://radiko.jp/v3/program/now/%s.xml"         // エリアID
)

// Client はRadiko APIクライアント
type Client struct {
	httpClient *http.Client
	areaID     string // デフォルトエリア（例: JP13 = 東京）
}

// NewClient は新しいRadikoクライアントを作成
func NewClient(areaID string) *Client {
	if areaID == "" {
		areaID = "JP13" // デフォルトは東京
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		areaID: areaID,
	}
}

// Station はラジオ局情報
type Station struct {
	ID       string `xml:"id,attr"`
	Name     string `xml:"name"`
	AreaID   string `xml:"area_id"`
	LogoURL  string `xml:"logo>large"` // ロゴ画像URL
	BannerURL string `xml:"banner"`     // バナー画像URL
}

// Program は番組情報
type Program struct {
	ID          string    `xml:"id,attr"`
	Title       string    `xml:"title"`
	Description string    `xml:"desc"`
	Info        string    `xml:"info"`
	StartTime   time.Time `xml:"ft,attr"`
	EndTime     time.Time `xml:"to,attr"`
	Duration    int       `xml:"dur,attr"` // 秒単位
	ImageURL    string    `xml:"img"`
	Personality string    `xml:"pfm"`      // パーソナリティ
	URL         string    `xml:"url"`
}

// StationList はラジオ局リスト
type StationList struct {
	XMLName  xml.Name  `xml:"stations"`
	Stations []Station `xml:"station"`
}

// WeeklyPrograms は週間番組表
type WeeklyPrograms struct {
	XMLName  xml.Name `xml:"radiko"`
	Stations []struct {
		ID    string `xml:"id,attr"`
		Progs struct {
			Date     string    `xml:"date"`
			Programs []Program `xml:"prog"`
		} `xml:"progs"`
	} `xml:"stations>station"`
}

// NowOnAir は現在放送中の番組
type NowOnAir struct {
	XMLName  xml.Name `xml:"radiko"`
	Stations []struct {
		ID   string `xml:"id,attr"`
		Prog Program `xml:"prog"`
	} `xml:"stations>station"`
}

// GetStations は指定エリアのラジオ局一覧を取得
func (c *Client) GetStations(ctx context.Context, areaID string) ([]Station, error) {
	if areaID == "" {
		areaID = c.areaID
	}

	url := fmt.Sprintf(StationListURL, areaID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get stations: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var stationList StationList
	if err := xml.Unmarshal(body, &stationList); err != nil {
		return nil, fmt.Errorf("failed to unmarshal XML: %w", err)
	}

	// エリアIDを設定
	for i := range stationList.Stations {
		stationList.Stations[i].AreaID = areaID
	}

	return stationList.Stations, nil
}

// GetWeeklyPrograms は指定局の週間番組表を取得
func (c *Client) GetWeeklyPrograms(ctx context.Context, stationID string) ([]Program, error) {
	url := fmt.Sprintf(WeeklyProgramURL, stationID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get weekly programs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var weekly WeeklyPrograms
	if err := xml.Unmarshal(body, &weekly); err != nil {
		return nil, fmt.Errorf("failed to unmarshal XML: %w", err)
	}

	// 全ての番組を1つのスライスに集約
	var allPrograms []Program
	for _, station := range weekly.Stations {
		allPrograms = append(allPrograms, station.Progs.Programs...)
	}

	return allPrograms, nil
}

// GetNowOnAir は現在放送中の番組一覧を取得
func (c *Client) GetNowOnAir(ctx context.Context, areaID string) (map[string]Program, error) {
	if areaID == "" {
		areaID = c.areaID
	}

	url := fmt.Sprintf(NowOnAirURL, areaID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get now on air: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var nowOnAir NowOnAir
	if err := xml.Unmarshal(body, &nowOnAir); err != nil {
		return nil, fmt.Errorf("failed to unmarshal XML: %w", err)
	}

	// ステーションIDをキーにしたマップに変換
	result := make(map[string]Program)
	for _, station := range nowOnAir.Stations {
		result[station.ID] = station.Prog
	}

	return result, nil
}

// GetProgramsByDate は指定日の番組表を取得
func (c *Client) GetProgramsByDate(ctx context.Context, stationID string, date time.Time) ([]Program, error) {
	// 週間番組表を取得
	allPrograms, err := c.GetWeeklyPrograms(ctx, stationID)
	if err != nil {
		return nil, err
	}

	// 指定日の番組のみフィルタリング
	targetDate := date.Format("2006-01-02")
	var programs []Program
	for _, prog := range allPrograms {
		progDate := prog.StartTime.Format("2006-01-02")
		if progDate == targetDate {
			programs = append(programs, prog)
		}
	}

	return programs, nil
}

// AreaIDs はRadikoのエリアID一覧
var AreaIDs = map[string]string{
	"JP1":  "北海道",
	"JP2":  "青森県",
	"JP3":  "岩手県",
	"JP4":  "宮城県",
	"JP5":  "秋田県",
	"JP6":  "山形県",
	"JP7":  "福島県",
	"JP8":  "茨城県",
	"JP9":  "栃木県",
	"JP10": "群馬県",
	"JP11": "埼玉県",
	"JP12": "千葉県",
	"JP13": "東京都",
	"JP14": "神奈川県",
	"JP15": "新潟県",
	"JP16": "富山県",
	"JP17": "石川県",
	"JP18": "福井県",
	"JP19": "山梨県",
	"JP20": "長野県",
	"JP21": "岐阜県",
	"JP22": "静岡県",
	"JP23": "愛知県",
	"JP24": "三重県",
	"JP25": "滋賀県",
	"JP26": "京都府",
	"JP27": "大阪府",
	"JP28": "兵庫県",
	"JP29": "奈良県",
	"JP30": "和歌山県",
	"JP31": "鳥取県",
	"JP32": "島根県",
	"JP33": "岡山県",
	"JP34": "広島県",
	"JP35": "山口県",
	"JP36": "徳島県",
	"JP37": "香川県",
	"JP38": "愛媛県",
	"JP39": "高知県",
	"JP40": "福岡県",
	"JP41": "佐賀県",
	"JP42": "長崎県",
	"JP43": "熊本県",
	"JP44": "大分県",
	"JP45": "宮崎県",
	"JP46": "鹿児島県",
	"JP47": "沖縄県",
}
