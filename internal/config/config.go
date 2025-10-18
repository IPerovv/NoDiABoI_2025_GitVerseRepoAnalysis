package config

import "time"

const (
	BaseURL     = "https://gitverse.ru/sc/sbt/api/v1"
	SearchPath  = "/repos/search"
	UserAgent   = "gitverse-scraper/1.0"
	PageLimit   = 50
	MaxRetries  = 6
	HTTPTimeout = 30 * time.Second

	PathToJSONDataset = "dataset/repos_pretty.json"

	MaxWorkers     = 40
	RatePerSecond  = 100
	ProgressPeriod = 30 * time.Second

	InitialBackoff = 1 * time.Second

	BatchSizeDbWrite = 1000

	TopStarsCount = 10
)
