package mapraptor

import (
	fare "github.com/takoyaki-3/go-gtfs-fare"
	gtfs "github.com/takoyaki-3/go-gtfs/v2"
)

type RAPTORData struct {
	TimeTables              map[string]TimeTable
	Transfer                map[string]map[string]float64
	StopId2Index            map[string]int
	TripId2Index            map[string]int
	TripId2StopPatternIndex map[string]int
	Fare                    *fare.GtfsFareData
}

type TimeTable struct {
	StopPatterns []gtfs.RoutePattern
	StopRoutes   map[string][]int
}
