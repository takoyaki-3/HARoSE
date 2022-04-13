package mapraptor

import (
	gtfs "github.com/takoyaki-3/go-gtfs/v2"
)

type RAPTORData struct {
	TimeTables              map[string]TimeTable
	RouteStop2StopSeq       []map[string]int
	Transfer                map[string]map[string]float64
	StopId2Index            map[string]int
	TripId2Index            map[string]int
	TripId2StopPatternIndex map[string]int
	GTFS                    *gtfs.GTFS
}

type TimeTable struct {
	StopPatterns []gtfs.RoutePattern
	StopRoutes   map[string][]int
}
