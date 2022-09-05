package loader

import (
	"time"

	. "github.com/MaaSTechJapan/raptor"
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

func LoadGTFS() (*RAPTORData, error) {

	raptorData := new(RAPTORData)
	raptorData.Transfer = map[string]map[string]float64{}
	//raptorData.RouteStops = [][]string{}
	//raptorData.RouteStop2StopSeq = []map[string]int{}
	raptorData.TimeTables = map[string]TimeTable{}
	//raptorData.TripId2Index = map[string]int{}
	raptorData.StopId2Index = map[string]int{}
	//raptorData.TripId2StopPatternIndex = map[string]int{}

	var conf Conf
	if err := json.LoadFromPath("./original_data/conf.json", &conf); err != nil {
		return &RAPTORData{}, err
	}

	if g, err := gtfs.Load("./original_data/"+conf.GTFS.Path, nil); err != nil {
		return &RAPTORData{}, err
	} else {
		if !conf.IsUseGTFSTransfer {
			// IsUseGTFSTransferがfalseの場合、transfer.txtをOSMから作成
			if conf.Map.FileName != "" {
				// 地図データ読み込み
				road, err := gm.LoadOSM("./original_data/" + conf.Map.FileName)
				if err != nil {
					return &RAPTORData{}, err
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
					return &RAPTORData{}, err
				}
				g.AddTransfer(conf.ConnectRange, conf.WalkingSpeed, road, conf.NumThread)
			}
		}

		// transfer
		for _, v := range g.Transfers {
			if _, ok := raptorData.Transfer[v.FromStopID]; !ok {
				raptorData.Transfer[v.FromStopID] = map[string]float64{}
			}
			if _, ok := raptorData.Transfer[v.ToStopID]; !ok {
				raptorData.Transfer[v.ToStopID] = map[string]float64{}
			}
			// 停留所間の徒歩時間[単位:秒]
			raptorData.Transfer[v.FromStopID][v.ToStopID] = float64(v.MinTime)
			raptorData.Transfer[v.ToStopID][v.FromStopID] = float64(v.MinTime)
		}

		if date, err := time.Parse("20060102", conf.StartDate); err != nil {
			return &RAPTORData{}, err
		} else {
			for {
				// 日付をベースとした絞り込み
				dateG := g.ExtractByDate(date)

				// route pattern（停車順がユニークな路線）の取得
				routePatterns := dateG.GetRoutePatterns()

				routeStops := [][]string{}
				routeStop2StopSeq := []map[string]int{}
				stopRoutes := map[string][]int{}
				tripId2Index := map[string]int{}
				tripId2RouteIndex := map[string]int{}

				for index, route := range routePatterns {
					for i, trip := range route.Trips {
						// tripId -> 同一route内でのtrip sequence
						tripId2Index[trip.Properties.TripID] = i
						// tripIdが該当するrouteのindex
						tripId2RouteIndex[trip.Properties.TripID] = index
					}

					trip := route.Trips[0]
					// 停留所ごとの停車する路線リスト
					for _, stopTime := range trip.StopTimes {
						stopRoutes[stopTime.StopID] = append(stopRoutes[stopTime.StopID], index)
					}
					// 各路線ごとに停留所IDから停車順を取得するためのリスト
					stopPattern := []string{}
					stopId2Sequence := map[string]int{}
					for i, stopTime := range trip.StopTimes {
						stopPattern = append(stopPattern, stopTime.StopID)
						stopId2Sequence[stopTime.StopID] = i
					}
					// routeが経由するstopIdの順列
					routeStops = append(routeStops, stopPattern)
					// (routeIndex, stopId) -> stopSequence
					routeStop2StopSeq = append(routeStop2StopSeq, stopId2Sequence)
				}

				dateStr := date.Format("20060102")
				raptorData.TimeTables[dateStr] = TimeTable{
					RoutePatterns:     routePatterns,
					RouteStops:        routeStops,
					RouteStop2StopSeq: routeStop2StopSeq,
					StopRoutes:        stopRoutes,
					TripId2Index:      tripId2Index,
					TripId2RouteIndex: tripId2RouteIndex,
				}
				if dateStr == conf.EndDate {
					break
				}
				date = date.AddDate(0, 0, 1)
			}
		}

		raptorData.StopId2Index = map[string]int{}
		for i, stop := range g.Stops {
			raptorData.StopId2Index[stop.ID] = i
		}
		raptorData.GTFS = g

		return raptorData, err
	}
}
