package loader

import (
	"sync"
	"time"

	"github.com/takoyaki-3/goraph"
	"github.com/takoyaki-3/goraph/geometry"

	. "github.com/MaaSTechJapan/raptor"
	"github.com/takoyaki-3/go-gtfs"
	"github.com/takoyaki-3/go-gtfs/stop_pattern"
	"github.com/takoyaki-3/go-gtfs/tool"
	json "github.com/takoyaki-3/go-json"
	"github.com/takoyaki-3/goraph/geometry/h3"
	"github.com/takoyaki-3/goraph/loader/osm"
	"github.com/takoyaki-3/goraph/search"
	uh3 "github.com/uber/h3-go"
)

type Conf struct {
	StartDate string `json:"start_date"`
	EndDate string `json:"end_date"`
	Map string `json:"map"`
	ConnectRange float64 `json:"connect_range"`
	NumThread int `json:"num_threads"`
}

func LoadGTFS() (*RAPTORData, *gtfs.GTFS, error) {

	raptorData := new(RAPTORData)
	raptorData.Transfer = map[string]map[string]float64{}
	raptorData.TimeTables = map[string]TimeTable{}

	var conf Conf
	if err := json.LoadFromPath("./conf.json",&conf);err != nil{
		return &RAPTORData{}, &gtfs.GTFS{}, err
	}

	if g, err := gtfs.Load("./GTFS", nil);err != nil {
		return &RAPTORData{}, &gtfs.GTFS{}, err
	} else {
		var road goraph.Graph
		var h3index map[uh3.H3Index][]int64
		if conf.Map != ""{
			// 地図データ読み込み
			road = osm.Load(conf.Map)
			h3index = h3.MakeH3Index(road,9)
		}
		wg := sync.WaitGroup{}
		wg.Add(conf.NumThread)
		type Dis struct {
			fromId string
			toId string
			dis float64
		}
		diss := make([][]Dis,conf.NumThread)
		for rank:=0;rank<conf.NumThread;rank++{
			go func(rank int){
				defer wg.Done()
				for i, stopI := range g.Stops {
					for j, stopJ := range g.Stops {
						if (i+j) % conf.NumThread != rank{
							continue
						}
						if i <= j {
							continue
						}
						dis := geometry.HubenyDistance(goraph.LatLon{
							Lat: stopI.Latitude,
							Lon: stopI.Longitude,
						}, goraph.LatLon{
							Lat: stopJ.Latitude,
							Lon: stopJ.Longitude,
						})
						if dis <= conf.ConnectRange && conf.Map != "" {
							// 道のりも計算
							route := search.Search(road,search.Query{
								From: h3.Find(road,h3index,goraph.LatLon{
									Lat: stopI.Latitude,
									Lon: stopI.Longitude,
									},9),
								To: h3.Find(road,h3index,goraph.LatLon{
									Lat: stopJ.Latitude,
									Lon: stopJ.Longitude,
									},9),
								})
							dis = route.Cost
						}
						if dis <= conf.ConnectRange || stopI.Parent == stopJ.Parent {
							diss[rank] = append(diss[rank],Dis{
								fromId: stopI.ID,
								toId: stopJ.ID,
								dis: dis})
						}
					}
				}
			}(rank)
		}
		wg.Wait()
		for _,arr := range diss{
			for _,v:=range arr{
				if _, ok := raptorData.Transfer[v.fromId]; !ok {
					raptorData.Transfer[v.fromId] = map[string]float64{}
				}
				if _, ok := raptorData.Transfer[v.toId]; !ok {
					raptorData.Transfer[v.toId] = map[string]float64{}
				}
				raptorData.Transfer[v.fromId][v.toId] = v.dis
				raptorData.Transfer[v.toId][v.fromId] = v.dis
			}
		}

		if date,err := time.Parse("20060102",conf.StartDate);err != nil {
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
