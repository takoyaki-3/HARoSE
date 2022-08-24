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
		if q, err := GetRoutingQuery(r, raptorData.GTFS); err != nil {
			log.Fatalln(err)
		} else {
			// RAPTOR
			memo := routing.RAPTOR(raptorData, q)

			// 計算結果から出力する経路を構成
			trips := []ri.TripStr{}
			//pos := q.ToStop
			//ro := q.Round

			// 到着側から逆順で経路を構成
			for round := q.Round; round > 0; round-- {
				pos := q.ToStop
				legs := []ri.LegStr{}

				// tauが更新されたラウンドまでskip
				if memo.Tau[round][pos] == memo.Tau[round-1][pos] {
					//fmt.Println(pos, round, "skip")
					continue
				}

				for k := round; k > 0; k-- {
					bef := memo.Tau[k][pos]
					now := pos
					if bef.ArrivalTime == 0 {
						fmt.Println("not found !")
						break
					}

					// 徒歩乗換
					if bef.WalkTransfer {
						now = bef.GetoffStop
					}

					// r回目の乗車便
					pos = bef.GetonStop
					viaNodes := []ri.StopTimeStr{}
					on := false
					tripId := string(bef.BoardingTrip)
					routePattern := raptorData.TimeTables[q.Date].TripId2RouteIndex[tripId]
					tripIndex := raptorData.TimeTables[q.Date].TripId2Index[tripId]

					// 乗車した便が経由する停留所の情報をlegに追加
					for _, v := range raptorData.TimeTables[q.Date].RoutePatterns[routePattern].Trips[tripIndex].StopTimes {
						if v.StopID == pos {
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
						legs = append([]ri.LegStr{leg}, legs...) // 配列の前に追加
					}
				}

				trip := ri.TripStr{
					Legs: legs,
				}

				// 各便の属性（系統名、停留所名など）を追加
				if len(trip.Legs) > 0 {
					trip.AddProperty(raptorData.GTFS)
				}

				trips = append(trips, trip)
			}

			// console出力
			fmt.Println("== Results ==")
			for i, trip := range trips {
				fmt.Println("[ Trip ", i, "]")
				for _, leg := range trip.Legs {
					fmt.Println(leg.Trip.ID, leg.Trip.Name)
					for _, stopTime := range leg.StopTimes {
						fmt.Println(stopTime.StopID, stopTime.ArrivalTime)
					}
				}
			}
			fmt.Println("END")

			fmt.Println(json.DumpToString(trips))

			// jsonで出力
			json.DumpToWriter(ri.ResponsStr{
				Trips: trips,
			}, w)
		}
	})

	// 単一出発点・単一出発時刻に対する到達圏を検索
	http.HandleFunc("/routing_surface", func(w http.ResponseWriter, r *http.Request) {

		if q, err := GetIsochroneQuery(r, raptorData.GTFS); err != nil {
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

func GetRoutingQuery(r *http.Request, g *gtfs.GTFS) (*routing.Query, error) {
	// 地点間検索クエリ
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

func GetIsochroneQuery(r *http.Request, g *gtfs.GTFS) (*routing.Query, error) {
	// 到達圏検索クエリ
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
		ToStop:      "nil", // 到着ノードは空を指定
		FromStop:    ri.FindNearestNode(query.Origin, g),
		FromTime:    *query.Origin.Time,
		MinuteSpeed: query.Properties.WalkingSpeed,
		Round:       query.Limit.Transfer,
		LimitTime:   *query.Origin.Time + query.Limit.Time,
		Date:        query.Properties.Timetable,
	}, nil
}
