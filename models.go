package mapraptor

import (
	gtfs "github.com/takoyaki-3/go-gtfs/v2"
)

type RAPTORData struct {
	TimeTables map[string]TimeTable // key: date("yyyyMMdd")
	//RouteStops              [][]string  // depends on dates
	//RouteStop2StopSeq       []map[string]int  // depends on dates?
	Transfer     map[string]map[string]float64
	StopId2Index map[string]int
	//TripId2Index            map[string]int  // depends on dates?
	//TripId2StopPatternIndex map[string]int  // depends on dates
	GTFS *gtfs.GTFS
}

type TimeTable struct {
	RoutePatterns     []gtfs.RoutePattern // tripTimetables in each unique route
	RouteStops        [][]string          // (routeIndex, stopSequence) -> stopId
	RouteStop2StopSeq []map[string]int    // (routeIndex, stopId) -> stopSequence
	StopRoutes        map[string][]int    // stopId -> list of routeIndex
	TripId2Index      map[string]int      // tripId -> tripIndex in route
	TripId2RouteIndex map[string]int      // tripId -> routeIndex
}
