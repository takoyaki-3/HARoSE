package loader

import (
	"fmt"
	"time"

	. "github.com/MaaSTechJapan/raptor"
	"github.com/takoyaki-3/go-gtfs"
	"github.com/takoyaki-3/go-gtfs/stop_pattern"
	"github.com/takoyaki-3/go-gtfs/tool"
	json "github.com/takoyaki-3/go-json"
	"github.com/takoyaki-3/goraph"
	"github.com/takoyaki-3/goraph/loader/osm"
	goraphtool "github.com/takoyaki-3/goraph/tool"
)

type ConfMap struct {
	MaxLat   float64 `json:"max_lat"`
	MaxLon   float64 `json:"max_lon"`
	MinLat   float64 `json:"min_lat"`
	MinLon   float64 `json:"min_lon"`
	FileName string  `json:"file_name"`
}

type Conf struct {
	StartDate    string  `json:"start_date"`
	EndDate      string  `json:"end_date"`
	Map          ConfMap `json:"map"`
	ConnectRange float64 `json:"connect_range"`
	NumThread    int     `json:"num_threads"`
	WalkingSpeed float64 `json:"walking_speed"`
	IsUseGTFSTransfer bool `json:"is_use_GTFS_transfer"`
}

func LoadGTFS() (*RAPTORData, *gtfs.GTFS, error) {

	raptorData := new(RAPTORData)
	raptorData.Transfer = map[string]map[string]float64{}
	raptorData.TimeTables = map[string]TimeTable{}

	var conf Conf
	if err := json.LoadFromPath("./conf.json", &conf); err != nil {
		return &RAPTORData{}, &gtfs.GTFS{}, err
	}

	if g, err := gtfs.Load("./GTFS", nil); err != nil {
		return &RAPTORData{}, &gtfs.GTFS{}, err
	} else {
		if !conf.IsUseGTFSTransfer {
			if conf.Map.FileName != "" {
				// 地図データ読み込み
				road := osm.Load(conf.Map.FileName)
				// 緯度経度で切り取り
				if conf.Map.MaxLat == 0 {
					conf.Map.MaxLat = 90
				}
				if conf.Map.MaxLon == 0 {
					conf.Map.MaxLon = 180
				}
				if conf.Map.MinLat == 0 {
					conf.Map.MinLat = -90
				}
				if conf.Map.MinLon == 0 {
					conf.Map.MinLon = -180
				}
				if conf.NumThread == 0 {
					conf.NumThread = 1
				}
				if conf.WalkingSpeed == 0{
					conf.WalkingSpeed = 80
				}
				if conf.ConnectRange == 0{
					conf.ConnectRange = 100
				}
				if err := goraphtool.CutGoraph(&road, goraph.LatLon{
					Lat: conf.Map.MaxLat,
					Lon: conf.Map.MinLon,
				}, goraph.LatLon{
					Lat: conf.Map.MinLat,
					Lon: conf.Map.MaxLon,
				}); err != nil {
					return &RAPTORData{}, &gtfs.GTFS{}, err
				}
				tool.MakeTransfer(g,conf.ConnectRange,conf.WalkingSpeed,road,conf.NumThread)
			}
		}

		for _,v := range g.Transfers{
			if _, ok := raptorData.Transfer[v.FromStopID]; !ok {
				raptorData.Transfer[v.FromStopID] = map[string]float64{}
			}
			if _, ok := raptorData.Transfer[v.ToStopID]; !ok {
				raptorData.Transfer[v.ToStopID] = map[string]float64{}
			}
			raptorData.Transfer[v.FromStopID][v.ToStopID] = float64(v.MinTime)
			raptorData.Transfer[v.ToStopID][v.FromStopID] = float64(v.MinTime)
		}

		if date, err := time.Parse("20060102", conf.StartDate); err != nil {
			return &RAPTORData{}, &gtfs.GTFS{}, err
		} else {
			for {
				// 日付をベースとした絞り込み
				dateG := tool.ExtractByDate(g, date)

				// 停車パターンの取得
				routePatterns := stoppattern.GetRoutePatterns(dateG)

				// 駅ごとの停車する路線リスト
				stopRoutes := map[string][]int{}
				for index, route := range routePatterns {
					trip := route.Trips[0]
					for _, stopTime := range trip.StopTimes {
						stopRoutes[stopTime.StopID] = append(stopRoutes[stopTime.StopID], index)
					}
				}

				dateStr := date.Format("20060102")
				raptorData.TimeTables[dateStr] = TimeTable{
					StopPatterns: routePatterns,
					StopRoutes:   stopRoutes,
				}
				if dateStr == conf.EndDate {
					break
				}
			}
		}
		return raptorData, g, nil
	}
}
