package util

import (
	"time"
)

func NewRateLimiter(ratePerSec int) <-chan time.Time {
	interval := time.Second / time.Duration(ratePerSec)
	if interval <= 0 {
		interval = time.Millisecond
	}
	rateTicker := time.NewTicker(interval)
	rateChan := make(chan time.Time)
	go func() {
		for t := range rateTicker.C {
			select {
			case rateChan <- t:
			default:
			}
		}
	}()
	return rateChan
}
