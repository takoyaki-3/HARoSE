package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"strconv"

	"github.com/MaaSTechJapan/raptor/loader"
	"github.com/MaaSTechJapan/raptor/routing"
	. "github.com/takoyaki-3/go-geojson"
	gtfs "github.com/takoyaki-3/go-gtfs/v2"
	json "github.com/takoyaki-3/go-json"
	gm "github.com/takoyaki-3/go-map/v2"
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
			memo := routing.RAPTOR(raptorData, q)

			pos := q.ToStop
			ro := q.Round - 1

			legs := []ri.LegStr{}

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

				// trip情報の取得
				trip := raptorData.GTFS.GetTrip(tripId)
				if len(viaNodes) > 0 {
					legs = append([]ri.LegStr{ri.LegStr{
						Trip:      trip,
						StopTimes: viaNodes,
					}}, legs...)
				}
				ro = ro - 1
			}

			// 追加情報の設定
			for i, _ := range legs {
				if err := legs[i].AddProperty(raptorData.GTFS); err != nil {
					log.Fatalln(err)
				}
			}

			// コストの合計
			c := ri.NewCostStr()
			for _, leg := range legs {
				c = ri.CostAdder(c, leg.Costs)
			}

			json.DumpToWriter(ri.ResponsStr{
				Trips: []ri.TripStr{
					ri.TripStr{
						Legs:  legs,
						Costs: c,
					},
				},
			}, w)
		}
	})
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

func FindODNode(qns QueryNodeStr, g *gtfs.GTFS) string {
	stopId := ""
	minD := math.MaxFloat64
	if qns.StopId == nil {
		for _, stop := range g.Stops {
			d := gm.HubenyDistance(gm.Node{
				Lat: stop.Latitude,
				Lon: stop.Longitude},
				gm.Node{
					Lat: *qns.Lat,
					Lon: *qns.Lon})
			if d < minD {
				stopId = stop.ID
				minD = d
			}
		}
	} else {
		stopId = *qns.StopId
	}
	return stopId
}

type QueryNodeStr struct {
	StopId *string  `json:"stop_id"`
	Lat    *float64 `json:"lat"`
	Lon    *float64 `json:"lon"`
	Time   *int     `json:"time"`
}
type QueryLimit struct {
	Time     int `json:"time"`
	Transfer int `json:"transfer"`
}
type QueryStr struct {
	Origin      QueryNodeStr  `json:"origin"`
	Destination QueryNodeStr  `json:"destination"`
	Limit       QueryLimit    `json:"limit"`
	WalkSpeed   float64       `json:"walk_speed"`
	Property    QueryProperty `json:"properties"`
}
type QueryProperty struct {
	Timetable string `json:"timetable"`
}

func GetRequestData(r *http.Request, queryStr interface{}) error {
	v := r.URL.Query()
	if v == nil {
		return errors.New("cannot get url query.")
	}
	return json.LoadFromString(v["json"][0], &queryStr)
}
func GetQuery(r *http.Request, g *gtfs.GTFS) (*routing.Query, error) {
	var query QueryStr
	if err := GetRequestData(r, &query); err != nil {
		return &routing.Query{}, err
	}
	if query.Limit.Time == 0 {
		query.Limit.Time = 3600 * 10
	}
	if query.Limit.Transfer == 0 {
		query.Limit.Transfer = 15
	}
	if query.WalkSpeed == 0 {
		query.WalkSpeed = 80
	}

	return &routing.Query{
		ToStop:      FindODNode(query.Destination, g),
		FromStop:    FindODNode(query.Origin, g),
		FromTime:    *query.Origin.Time,
		MinuteSpeed: query.WalkSpeed,
		Round:       query.Limit.Transfer,
		LimitTime:   *query.Origin.Time + query.Limit.Time,
		Date:        query.Property.Timetable,
	}, nil
}
