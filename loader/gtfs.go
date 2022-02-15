package loader

import (
	"github.com/takoyaki-3/goraph"
	"github.com/takoyaki-3/goraph/geometry"
	"log"
	"time"

	. "github.com/MaaSTechJapan/raptor"
	"github.com/takoyaki-3/go-gtfs"
	"github.com/takoyaki-3/go-gtfs/stop_pattern"
	"github.com/takoyaki-3/go-gtfs/tool"
)

func LoadGTFS() (*RAPTORData, *gtfs.GTFS) {
	g, err := gtfs.Load("./GTFS", nil)

	if err != nil {
		log.Fatalln(err)
	}

	// 地図データ読み込み
	// road := osm.Load("./map.osm.pbf")
	// h3index := h3.MakeH3Index(road,9)

	// 日付をベースとした絞り込み
	g = tool.ExtractByDate(g, time.Now())

	// 停車パターンの取得
	routePatterns := stoppattern.GetRoutePatterns(g)

	// 駅ごとの停車する路線リスト
	stopRoutes := map[string][]int{}
	for index, route := range routePatterns {
		trip := route.Trips[0]
		for _, stopTime := range trip.StopTimes {
			stopRoutes[stopTime.StopID] = append(stopRoutes[stopTime.StopID], index)
		}
	}

	raptorData := &RAPTORData{
		StopPatterns: routePatterns,
		StopRoutes:   stopRoutes,
		Transfer:     map[string]map[string]float64{},
	}

	// 接続情報の設定
	for i, stopI := range g.Stops {
		for j, stopJ := range g.Stops {
			if i == j {
				continue
			}
			if _, ok := raptorData.Transfer[stopI.ID]; !ok {
				raptorData.Transfer[stopI.ID] = map[string]float64{}
			}
			// route := search.Search(road,search.Query{
			// 	From: h3.Find(road,h3index,goraph.LatLon{
			// 		Lat: stopI.Latitude,
			// 		Lon: stopI.Longitude,
			// 		},9),
			// 	To: h3.Find(road,h3index,goraph.LatLon{
			// 		Lat: stopJ.Latitude,
			// 		Lon: stopJ.Longitude,
			// 		},9),
			// 	})
			// dis := route.Cost
			dis := geometry.HubenyDistance(goraph.LatLon{
				Lat: stopI.Latitude,
				Lon: stopI.Longitude,
			}, goraph.LatLon{
				Lat: stopJ.Latitude,
				Lon: stopJ.Longitude,
			})

			if dis <= 100 || stopI.Parent == stopJ.Parent {
				raptorData.Transfer[stopI.ID][stopJ.ID] = dis
			}
		}
	}
	return raptorData, g
}
