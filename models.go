package mapraptor

import (
	"github.com/takoyaki-3/go-gtfs/stop_pattern"
)

type RAPTORData struct {
	StopPatterns []stoppattern.RoutePattern
	StopRoutes map[string][]int
	Transfer map[string]map[string]float64
}
