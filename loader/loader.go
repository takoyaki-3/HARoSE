package loader

import (
	"time"

	. "github.com/MaaSTechJapan/raptor"
	fare "github.com/takoyaki-3/go-gtfs-fare"
	gtfs "github.com/takoyaki-3/go-gtfs/v2"
	json "github.com/takoyaki-3/go-json"
	gm "github.com/takoyaki-3/go-map/v2"
)

type ConfMap struct {
	MaxLat   float64 `json:"max_lat"`
	MaxLon   float64 `json:"max_lon"`
	MinLat   float64 `json:"min_lat"`
	MinLon   float64 `json:"min_lon"`
	FileName string  `json:"file_name"`
}

type ConfGtfs struct {
	Path string `json:"path"`
}

type Conf struct {
	StartDate         string   `json:"start_date"`
	EndDate           string   `json:"end_date"`
	GTFS              ConfGtfs `json:"gtfs"`
	Map               ConfMap  `json:"map"`
	ConnectRange      float64  `json:"connect_range"`
	NumThread         int      `json:"num_threads"`
	WalkingSpeed      float64  `json:"walking_speed"`
	IsUseGTFSTransfer bool     `json:"is_use_GTFS_transfer"`
}

func LoadGTFS() (*RAPTORData, *gtfs.GTFS, error) {

	raptorData := new(RAPTORData)
	raptorData.Transfer = map[string]map[string]float64{}
	raptorData.TimeTables = map[string]TimeTable{}
	raptorData.TripId2Index = map[string]int{}
	raptorData.StopId2Index = map[string]int{}
	raptorData.TripId2StopPatternIndex = map[string]int{}

	var conf Conf
	if err := json.LoadFromPath("./conf.json", &conf); err != nil {
		return &RAPTORData{}, &gtfs.GTFS{}, err
	}

	if g, err := gtfs.Load(conf.GTFS.Path, nil); err != nil {
		return &RAPTORData{}, &gtfs.GTFS{}, err
	} else {
		if !conf.IsUseGTFSTransfer {
			if conf.Map.FileName != "" {
				// 地図データ読み込み
				road, err := gm.LoadOSM(conf.Map.FileName)
				if err != nil {
					return &RAPTORData{}, &gtfs.GTFS{}, err
				}
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
				if conf.WalkingSpeed == 0 {
					conf.WalkingSpeed = 80
				}
				if conf.ConnectRange == 0 {
					conf.ConnectRange = 100
				}
				if err := road.CutGraph(gm.Node{
					Lat: conf.Map.MaxLat,
					Lon: conf.Map.MinLon,
				}, gm.Node{
					Lat: conf.Map.MinLat,
					Lon: conf.Map.MaxLon,
				}); err != nil {
					return &RAPTORData{}, &gtfs.GTFS{}, err
				}
				g.AddTransfer(conf.ConnectRange, conf.WalkingSpeed, road, conf.NumThread)
			}
		}

		for _, v := range g.Transfers {
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
				dateG := g.ExtractByDate(date)

				// 停車パターンの取得
				routePatterns := dateG.GetRoutePatterns()

				// 駅ごとの停車する路線リスト
				stopRoutes := map[string][]int{}
				for index, route := range routePatterns {
					for i, trip := range route.Trips {
						raptorData.TripId2Index[trip.Properties.TripID] = i
						raptorData.TripId2StopPatternIndex[trip.Properties.TripID] = index
					}
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

		raptorData.StopId2Index = map[string]int{}
		for i, stop := range g.Stops {
			raptorData.StopId2Index[stop.ID] = i
		}

		// 運賃情報の読み込み
		raptorData.Fare, err = fare.LoadGTFS(conf.GTFS.Path)

		return raptorData, g, err
	}
}
