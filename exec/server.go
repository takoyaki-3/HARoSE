package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	"github.com/MaaSTechJapan/raptor/loader"
	"github.com/MaaSTechJapan/raptor/routing"
	. "github.com/takoyaki-3/go-geojson"
	gtfs "github.com/takoyaki-3/go-gtfs/v2"
	json "github.com/takoyaki-3/go-json"
	ri "github.com/takoyaki-3/go-routing-interface"
)

func main() {

	// RAPTOR用データの読み込み
	raptorData, err := loader.LoadGTFS()
	if err != nil {
		log.Fatalln(err)
	}

	// index.html
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		bytes, _ := ioutil.ReadFile("./index.html")
		fmt.Fprintln(w, string(bytes))
	})

	// routing
	http.HandleFunc("/v1/json/routing", func(w http.ResponseWriter, r *http.Request) {

		// Query
		if q, err := GetQuery(r, raptorData.GTFS); err != nil {
			log.Fatalln(err)
		} else {
			// RAPTOR
			memo := routing.RAPTOR(raptorData, q)

			// 出力する検索結果を構成
			pos := q.ToStop
			ro := q.Round - 1

			legs := []ri.LegStr{}

			// 最も到着時刻が早く乗換回数が多い経路1本を出力
			for pos != q.FromStop {
				bef := memo.Tau[ro][pos]
				now := pos
				if bef.ArrivalTime == 0 {
					fmt.Println("not found !")
					break
				}
				pos = bef.BeforeStop

				viaNodes := []ri.StopTimeStr{}
				on := false

				tripId := string(memo.Tau[ro][now].BeforeEdge)
				routePattern := raptorData.TripId2StopPatternIndex[tripId]
				tripIndex := raptorData.TripId2Index[tripId]

				// 乗車した便が経由する停留所の情報をlegに追加
				for _, v := range raptorData.TimeTables[q.Date].StopPatterns[routePattern].Trips[tripIndex].StopTimes {
					if v.StopID == bef.BeforeStop {
						on = true
					}
					if on {
						stopId := v.StopID
						viaNodes = append(viaNodes, ri.StopTimeStr{
							StopID:        string(stopId),
							ArrivalTime:   v.Arrival,
							DepartureTime: v.Departure,
						})
					}
					if v.StopID == now {
						break
					}
				}

				// Legを経路に追加
				if len(viaNodes) > 0 {
					leg := ri.LegStr{
						StopTimes: viaNodes,
					}
					leg.Trip.ID = tripId
					legs = append([]ri.LegStr{leg}, legs...)
				}
				ro = ro - 1
			}

			trip := ri.TripStr{
				Legs: legs,
			}
			// 各便の属性（系統名、停留所名など）を追加
			trip.AddProperty(raptorData.GTFS)

			// jsonで出力
			json.DumpToWriter(ri.ResponsStr{
				Trips: []ri.TripStr{
					trip,
				},
			}, w)
		}
	})

	// 単一出発地・単一出発時刻に対する到達圏検索
	http.HandleFunc("/routing_surface", func(w http.ResponseWriter, r *http.Request) {

		if q, err := GetQuery(r, raptorData.GTFS); err != nil {
			log.Fatalln(err)
		} else {
			StopId2Index := map[string]int{}
			for i, stop := range raptorData.GTFS.Stops {
				StopId2Index[stop.ID] = i
			}

			fc := NewFeatureCollection()

			memo := routing.RAPTOR(raptorData, q)
			for r, m := range memo.Tau {
				for k, v := range m {
					props := map[string]string{}
					props["stop_id"] = k
					props["arrival_time"] = gtfs.Sec2HHMMSS(v.ArrivalTime)
					props["stop_name"] = raptorData.GTFS.Stops[StopId2Index[k]].Name
					props["round"] = strconv.Itoa(r)
					fc.Features = append(fc.Features, Feature{
						Type: "Feature",
						Geometry: Geometry{
							Type:        "Point",
							Coordinates: []float64{raptorData.GTFS.Stops[StopId2Index[k]].Longitude, raptorData.GTFS.Stops[StopId2Index[k]].Latitude},
						},
						Properties: props,
					})
				}
			}
			json.DumpToWriter(fc, w)
		}
	})
	fmt.Println("start server.")
	if err := http.ListenAndServe("0.0.0.0:8000", nil); err != nil {
		log.Fatalln(err)
	}
}

func GetRequestData(r *http.Request, queryStr interface{}) error {
	v := r.URL.Query()
	if v == nil {
		return errors.New("cannot get url query.")
	}
	return json.LoadFromString(v["json"][0], &queryStr)
}
func GetQuery(r *http.Request, g *gtfs.GTFS) (*routing.Query, error) {
	var query ri.QueryStr
	if err := GetRequestData(r, &query); err != nil {
		return &routing.Query{}, err
	}
	if query.Limit.Time == 0 {
		query.Limit.Time = 3600 * 10
	}
	if query.Limit.Transfer == 0 {
		query.Limit.Transfer = 15 // 最大ラウンド数
	}
	if query.Properties.WalkingSpeed == 0 {
		query.Properties.WalkingSpeed = 80 // 単位:[m/分]
	}

	return &routing.Query{
		ToStop:      ri.FindNearestNode(query.Destination, g), // 指定した緯度経度から最も近い停留所
		FromStop:    ri.FindNearestNode(query.Origin, g),
		FromTime:    *query.Origin.Time,
		MinuteSpeed: query.Properties.WalkingSpeed,
		Round:       query.Limit.Transfer,
		LimitTime:   *query.Origin.Time + query.Limit.Time,
		Date:        query.Properties.Timetable,
	}, nil
}
