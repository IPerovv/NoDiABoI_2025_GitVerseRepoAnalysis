package model

import "time"

type RepoInfo struct {
	ID             int64     `json:"id"`
	FullName       string    `json:"full_name"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	Archived       bool      `json:"archived"`
	StarsCount     int       `json:"stars_count"`
	Size           int       `json:"size"`
	ReleaseCounter int       `json:"release_counter"`
	TagCount       int       `json:"tag_count"`
}
