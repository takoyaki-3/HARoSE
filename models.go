package mapraptor

import (
	"github.com/takoyaki-3/go-gtfs/stop_pattern"
)

type RAPTORData struct {
	TimeTables map[string]TimeTable
	Transfer   map[string]map[string]float64
}

type TimeTable struct {
	StopPatterns []stoppattern.RoutePattern
	StopRoutes   map[string][]int
}
