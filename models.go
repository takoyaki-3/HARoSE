package mapraptor

import (
	"github.com/takoyaki-3/go-gtfs/stop_pattern"
)

type RAPTORData struct {
	TimeTables              map[string]TimeTable
	Transfer                map[string]map[string]float64
	StopId2Index            map[string]int
	TripId2Index            map[string]int
	TripId2StopPatternIndex map[string]int
}

type TimeTable struct {
	StopPatterns []stoppattern.RoutePattern
	StopRoutes   map[string][]int
}
