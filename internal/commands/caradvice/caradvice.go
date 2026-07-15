package caradvice

import (
	_ "embed"
	"encoding/json"
	"math/rand"
	"sync"
)

//go:embed data/caradvice.json
var carAdviceData []byte

var (
	mu      sync.Mutex
	choices []string
)

func Next() string {
	mu.Lock()
	defer mu.Unlock()
	if len(choices) == 0 {
		_ = json.Unmarshal(carAdviceData, &choices)
		choices = append([]string(nil), choices...)
	}
	if len(choices) == 0 {
		return ""
	}
	idx := rand.Intn(len(choices))
	result := choices[idx]
	choices = append(choices[:idx], choices[idx+1:]...)
	return result
}
