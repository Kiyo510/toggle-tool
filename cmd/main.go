package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/najeira/jpholiday"
	"github.com/olekukonko/tablewriter"
)

type TimeEntrySearchRequest struct {
	Billable        bool   `json:"billable,omitempty"`
	ClientIDs       []int  `json:"client_ids,omitempty"`
	Description     string `json:"description,omitempty"`
	EndDate         string `json:"end_date,omitempty"`
	GroupIDs        []int  `json:"group_ids,omitempty"`
	Grouped         bool   `json:"grouped,omitempty"`
	HideAmounts     bool   `json:"hide_amounts,omitempty"`
	MaxDuration     int    `json:"max_duration_seconds,omitempty"`
	MinDuration     int    `json:"min_duration_seconds,omitempty"`
	OrderBy         string `json:"order_by,omitempty"`
	OrderDir        string `json:"order_dir,omitempty"`
	PageSize        int    `json:"page_size,omitempty"`
	ProjectIDs      []int  `json:"project_ids,omitempty"`
	Rounding        int    `json:"rounding,omitempty"`
	RoundingMinutes int    `json:"rounding_minutes,omitempty"`
	StartDate       string `json:"start_date,omitempty"`
	TagIDs          []int  `json:"tag_ids,omitempty"`
	TaskIDs         []int  `json:"task_ids,omitempty"`
	TimeEntryIDs    []int  `json:"time_entry_ids,omitempty"`
	UserIDs         []int  `json:"user_ids,omitempty"`
}

type TimeEntryResponseItem struct {
	TagIDs      []int `json:"tag_ids"`
	TimeEntries []struct {
		Start   string `json:"start"`
		Seconds int    `json:"seconds"`
	} `json:"time_entries"`
}

const BaseURL string = "https://api.track.toggl.com"

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

type TagsResponseItem struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func main() {
	var WorkSpaceId = os.Getenv("WORKSPACE_ID")
	loc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		log.Fatalf("Error loading location: %v", err)
	}

	// コマンドライン引数から年月を取得
	year, month, err := parseMonthArgument(os.Args)
	if err != nil {
		log.Fatalf("Error parsing month argument: %v", err)
	}

	startOfMonth := time.Date(year, month, 1, 0, 0, 0, 0, loc)
	endOfMonth := startOfMonth.AddDate(0, 1, -1)
	requestData := TimeEntrySearchRequest{
		StartDate: startOfMonth.Format("2006-01-02"),
		EndDate:   endOfMonth.Format("2006-01-02"),
		PageSize:  3000,
	}

	b, err := json.Marshal(requestData)
	if err != nil {
		fmt.Println("Error marshalling request data:", err)
		return
	}

	client := NewToggleClient()
	resp, err := client.Post(fmt.Sprintf("%s/reports/api/v3/workspace/%s/search/time_entries", BaseURL, WorkSpaceId), "application/json", bytes.NewBuffer(b))
	if err != nil {
		fmt.Printf("Error while sending request: %v, Error: %v", resp.StatusCode, err)
		return
	}
	defer func(Body io.ReadCloser) {
		closeErr := Body.Close()
		if closeErr != nil {
			fmt.Printf("Error while closing response body: %v, Error: %v", closeErr, err)
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response:", err)
		return
	}

	var items []TimeEntryResponseItem
	if err := json.Unmarshal(body, &items); err != nil {
		log.Fatalf("JSON unmarshaling failed: %s", err)
	}

	resp2, err := client.Get(fmt.Sprintf("%s/api/v9/workspaces/%s/tags", BaseURL, WorkSpaceId))
	if err != nil {
		fmt.Printf("Error while sending request: %v, Error: %v", resp2.StatusCode, err)
		return
	}
	defer func(Body io.ReadCloser) {
		closeErr := Body.Close()
		if closeErr != nil {
			fmt.Printf("Error while closing response body: %v, Error: %v", closeErr, err)
		}
	}(resp2.Body)

	body2, err := io.ReadAll(resp2.Body)
	if err != nil {
		fmt.Println("Error reading response:", err)
		return
	}

	var tags []TagsResponseItem
	if err := json.Unmarshal(body2, &tags); err != nil {
		log.Fatalf("JSON unmarshaling failed: %s", err)
	}

	// タグIDと名前のマッピング
	tagIdNameMap := make(map[int]string)
	for _, tag := range tags {
		tagIdNameMap[tag.ID] = tag.Name
	}

	dateTagSeconds := make(map[string]map[int]int)
	for _, item := range items {
		for _, entry := range item.TimeEntries {
			date, err := time.Parse(time.RFC3339, entry.Start)
			if err != nil {
				log.Fatalf("Date parsing failed: %s", err)
			}
			dateStr := date.Format("2006-01-02")

			if dateTagSeconds[dateStr] == nil {
				dateTagSeconds[dateStr] = make(map[int]int)
			}

			for _, tagID := range item.TagIDs {
				dateTagSeconds[dateStr][tagID] += entry.Seconds
			}
		}
	}

	var dates []string
	for date := range dateTagSeconds {
		dates = append(dates, date)
	}

	sort.Slice(dates, func(i, j int) bool {
		return dates[i] < dates[j]
	})

	// ソートされた日付キーを使用して結果を出力
	for _, date := range dates {
		dateTime, err := time.Parse("2006-01-02", date)
		if err != nil {
			log.Printf("Error while parsing date: %v\n", err)
		}
		if isWeekendOrHoliday(dateTime) {
			continue
		}

		tagSeconds := dateTagSeconds[date]

		fmt.Println(strings.Repeat("=", 60))
		fmt.Printf("Date: %s\n", date)
		fmt.Println(strings.Repeat("=", 60))

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Tag Name", "Hours"})
		dailyTotalSeconds := 0

		for tagID, seconds := range tagSeconds {
			dailyTotalSeconds += seconds
			hours := seconds / 3600
			minutes := (seconds % 3600) / 60
			seconds = seconds % 60
			tagName, exists := tagIdNameMap[tagID]
			if !exists {
				tagName = "Unknown Tag"
			}
			table.Append([]string{tagName, fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)})
		}

		totalHours := dailyTotalSeconds / 3600
		totalMinutes := (dailyTotalSeconds % 3600) / 60
		totalSeconds := dailyTotalSeconds % 60
		table.SetFooter([]string{"Total", fmt.Sprintf("%02d:%02d:%02d", totalHours, totalMinutes, totalSeconds)})
		table.Render()
	}
	fmt.Println(strings.Repeat("=", 60))
}

func NewToggleClient() *http.Client {
	return &http.Client{Transport: &CustomTransport{APIKey: os.Getenv("TOGGLE_API_KEY")}}
}

type CustomTransport struct {
	APIKey string
}

func (c *CustomTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.SetBasicAuth(c.APIKey, "api_token")

	return http.DefaultTransport.RoundTrip(req)
}

func parseMonthArgument(args []string) (int, time.Month, error) {
	// 引数なしの場合は現在の年月を使用
	if len(args) < 2 {
		now := time.Now()
		return now.Year(), now.Month(), nil
	}

	monthArg := args[1]
	
	// YYYY-MM形式で解析
	parts := strings.Split(monthArg, "-")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid month format. Use YYYY-MM (e.g., 2025-06)")
	}

	year, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid year: %s", parts[0])
	}

	monthInt, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid month: %s", parts[1])
	}

	if monthInt < 1 || monthInt > 12 {
		return 0, 0, fmt.Errorf("month must be between 1 and 12, got %d", monthInt)
	}

	return year, time.Month(monthInt), nil
}

func isWeekendOrHoliday(date time.Time) bool {
	// 土日チェック
	if date.Weekday() == time.Saturday || date.Weekday() == time.Sunday {
		return true
	}

	// 祝日チェック
	day, err := time.Parse("2006-01-02", date.Format("2006-01-02"))
	if err != nil {
		log.Fatalf("Error while parsing date: %v", err)
		return false
	}
	name := jpholiday.Name(day)
	if name != "" {
		return true
	}

	return false
}
