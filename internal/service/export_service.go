package service

import (
	"encoding/csv"
	"encoding/json"
	"strconv"
	"path/filepath"
	"fmt"
	"os"
	"time"

	"github.com/xuri/excelize/v2"
)

type RepoJSON struct {
	ID             int64  `json:"id"`
	FullName       string `json:"fullname"`
	Archived       bool   `json:"archived"`
	StarsCount     int    `json:"starscount"`
	Size           int    `json:"size"`
	ReleaseCounter int    `json:"releasecounter"`
	TagCount       int    `json:"tagcount"`

	CreatedAt struct {
		Date time.Time `json:"$date"`
	} `json:"createdat"`

	UpdatedAt struct {
		Date time.Time `json:"$date"`
	} `json:"updatedat"`
}

func ConvertJSONToCSVXLSX(jsonPath, csvPath string) (string, error) {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return "", fmt.Errorf("read json: %w", err)
	}

	var raw []RepoJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return "", fmt.Errorf("unmarshal: %w", err)
	}

	outputDir := "dataset/tables"
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("mkdir: %w", err)
	}

	filename := fmt.Sprintf("repos_%s", time.Now().Format("2006-01-02_15-04-05"))
	csvPath = filepath.Join(outputDir, filename+".csv")
	xlsxPath := filepath.Join(outputDir, filename+".xlsx")

	file, err := os.Create(csvPath)
	if err != nil {
		return "", fmt.Errorf("create csv: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	headers := []string{
		"id", "full_name", "created_at", "updated_at",
		"archived", "stars_count", "size", "release_counter", "tag_count",
	}
	writer.Write(headers)

	f := excelize.NewFile()
	sheet := "Repos"
	f.NewSheet(sheet)
	f.DeleteSheet("Sheet1")
	for i, h := range headers {
		col := string(rune('A' + i))
		f.SetCellValue(sheet, col+"1", h)
	}

	for i, r := range raw {
		row := []string{
			strconv.FormatInt(r.ID, 10),
			r.FullName,
			r.CreatedAt.Date.Format(time.RFC3339),
			r.UpdatedAt.Date.Format(time.RFC3339),
			strconv.FormatBool(r.Archived),
			strconv.Itoa(r.StarsCount),
			strconv.Itoa(r.Size),
			strconv.Itoa(r.ReleaseCounter),
			strconv.Itoa(r.TagCount),
		}
		writer.Write(row)

		for j, v := range row {
			col := string(rune('A' + j))
			f.SetCellValue(sheet, fmt.Sprintf("%s%d", col, i+2), v)
		}
	}

	if err := f.SaveAs(xlsxPath); err != nil {
		return "", fmt.Errorf("save xlsx: %w", err)
	}

	fmt.Printf("CSV saved to %s\n", csvPath)
	fmt.Printf("XLSX saved to %s (%d records)\n", xlsxPath, len(raw))

	return filename, nil
}
