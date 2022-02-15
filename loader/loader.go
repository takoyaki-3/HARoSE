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
	json "github.com/takoyaki-3/go-json"
)

type Conf struct {
	Dates []string `json:"dates"`
	Map string `json:"map"`
}

func LoadGTFS() (*RAPTORData, *gtfs.GTFS) {

	raptorData := new(RAPTORData)
	raptorData.Transfer = map[string]map[string]float64{}
	raptorData.TimeTables = map[string]TimeTable{}

	var conf Conf
	if err := json.LoadFromPath("./conf.json",&conf);err != nil{
		log.Fatalln(err)
	}

	if g, err := gtfs.Load("./GTFS", nil);err != nil {
		log.Fatalln(err)
	} else {
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

		// 地図データ読み込み
		// road := osm.Load("./map.osm.pbf")
		// h3index := h3.MakeH3Index(road,9)

		for _, date := range conf.Dates {

			// 日付をベースとした絞り込み
			if t,err := time.Parse("20060102",date);err!=nil{
				log.Fatalln(err)
			} else {
				dateG := tool.ExtractByDate(g, t)

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

				raptorData.TimeTables[date] = TimeTable{
					StopPatterns: routePatterns,
					StopRoutes:   stopRoutes,
				}
			}
		}
		return raptorData, g
	}
	return &RAPTORData{},&gtfs.GTFS{}
}
