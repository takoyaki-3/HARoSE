package models

import (
	geojson "github.com/takoyaki-3/go-geojson"
)

// クエリ用構造体
type QueryStr struct {
	Origin      QueryNodeStr `json:"origin"`
	Destination QueryNodeStr `json:"destination"`
	// CostWeight  CostStr        `json:"cost_weight"`
	Properties PropertiesStr `json:"properties"`
}

// 始点発着点重み
type QueryNodeStr struct {
	StopId *string  `json:"stop_id"`
	Lat    *float64 `json:"lat"`
	Lon    *float64 `json:"lon"`
	Time   *int     `json:"time"`
}

// 経路探索時の加重平均用重み
type CostStr struct {
	Time     *float64 `json:"time"`
	Walk     *float64 `json:"walk"`
	Transfer *float64 `json:"transfer"`
	Distance *float64 `json:"distance"`
	Fare     *float64 `json:"fare"`
}

// 新たなコスト関数
func NewCostStr() (c CostStr) {
	c.Time = floatPointer(0.0)
	c.Walk = floatPointer(0.0)
	c.Transfer = floatPointer(0.0)
	c.Distance = floatPointer(0.0)
	c.Fare = floatPointer(0.0)
	return c
}

// レスポンス用構造体
type ResponsStr struct {
	Trips []TripStr `json:"trips,omitempty"`
	Meta  struct {
		EngineVersion string `json:"engine_version,omitempty"`
		LastUpdated   string `json:"last_updated,omitempty"`
	} `json:"meta,omitempty"`
	Status string `json:"status,omitempty"`
}

// リクエストプロパティ
type PropertiesStr struct {
	WalkingSpeed float64 `json:"walking_speed"`
	Timetable    string  `json:"timetable"`
}

// プロパティ
type PropertyStr struct {
	ArrivalTime   string `json:"arrival_time"`
	DepartureTime string `json:"departure_time"`
}

// 時空間グラフにおける辺
type TimeEdgeStr struct {
	FromTime string `json:"from_time"`
	FromNode string `json:"from_node"`
	ToTime   string `json:"to_time"`
	ToNode   string `json:"to_node"`
}

// 経路
type TripStr struct {
	Legs       []LegStr `json:"legs"`
	Properties struct {
		TotalTime     int    `json:"total_time"`
		ArrivalTime   string `json:"arrival_time"`
		DepartureTime string `json:"departure_time"`
	} `json:"properties"`
	Costs CostStr `json:"costs"`
}

type StopTimeStr struct {
	StopId        string  `json:"stop_id"`
	ZoneId        string  `json:"zone_id"`
	StopLat       float64 `json:"stop_lat"`
	StopLon       float64 `json:"stop_lon"`
	StopName      string  `json:"stop_name"`
	ArrivalTime   string  `json:"arrival_time"`
	DepartureTime string  `json:"departure_time"`
	PickupType    string  `json:"pickup_type"`
	DropOffType   string  `json:"drop_off_type"`
	StopSequence  int     `json:"stop_sequence"`
}

// １乗車
type LegStr struct {
	Type       string            `json:"type"`
	Trip       GTFSTripStr       `json:"vehicle"`
	StopTimes  []StopTimeStr     `json:"stop_times"`
	TimeEdges  []TimeEdgeStr     `json:"time_edges"`
	Geometry   *geojson.Geometry `json:"geometry"`
	Costs      CostStr           `json:"cost"`
	Properties PropertyStr       `json:"properties"`
}

type GTFSTripStr struct {
	TripId          string `json:"trip_id" csv:"trip_id"`
	TripDescription string `json:"trip_description" csv:"trip_description"`
	RouteLongName   string `json:"route_long_name" csv:"route_long_name"`
	ServiceId       string `json:"service_id" csv:"service_id"`
	TripType        string `json:"trip_type" csv:"trip_type"`
	RouteColor      string `json:"route_color" csv:"route_color"`
	RouteTextColor  string `json:"route_text_color" csv:"route_text_color"`
	RouteShortName  string `json:"route_short_name" csv:"route_short_name"`
	TripHeadSign    string `json:"trip_headsign" csv:"trip_headsign"`
	RouteId         string `json:"route_id" csv:"route_id"`
}

func floatPointer(i float64) *float64 {
	return &i
}
