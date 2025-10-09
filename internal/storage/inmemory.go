package storage

import (
	"sync"

	"gitverse-analyser-service/internal/model"
)

var (
	RepoStore = make(map[int64]model.RepoInfo)
	StoreMu   sync.Mutex
)
