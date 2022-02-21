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
	"github.com/MaaSTechJapan/raptor/models"
	"github.com/MaaSTechJapan/raptor/routing"
	. "github.com/takoyaki-3/go-geojson"
	"github.com/takoyaki-3/go-gtfs"
	"github.com/takoyaki-3/go-gtfs/pkg"
	json "github.com/takoyaki-3/go-json"
	"github.com/takoyaki-3/goraph"
	"github.com/takoyaki-3/goraph/geometry"
)

type MTJNodeStr struct {
	Id    string  `json:"id"`
	Lat   float64 `json:"lat"`
	Lon   float64 `json:"lon"`
	Title string  `json:"title"`
	ArrivalTime string `json:"arrival_time"`
	DepartureTime string `json:"departure_time"`
}
type MTJLegStr struct {
	Id             string     `json:"id"`
	Uid            string     `json:"uid"`
	Oid            string     `json:"oid"`
	Title          string     `json:"title"`
	Created        string     `json:"created"`
	Issued         string     `json:"issued"`
	Available      string     `json:"available"`
	Valid          string     `json:"valid"`
	Type           string     `json:"type"`
	SubType        string     `json:"subtype"`
	FromNode       MTJNodeStr `json:"from_node"`
	ToNode         MTJNodeStr `json:"to_node"`
	ViaStops			 []MTJNodeStr `json:"stop_times"`
	Transportation string     `json:"transportation"`
	load           string     `json:"load"`
	WKT            string     `json:WKT`
	Geometry       Geometry   `json:"geometry"`
}
type MTJTripStr struct {
	Legs []MTJLegStr `json:"legs"`
}
type MTJResp struct {
	Trips []MTJTripStr `json:"trips"`
}

func main() {

	// RAPTOR用データの読み込み
	raptorData, g, err := loader.LoadGTFS()
	if err != nil {
		log.Fatalln(err)
	}

	// index.html
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		bytes, _ := ioutil.ReadFile("./index.html")
		fmt.Fprintln(w, string(bytes))
	})
	http.HandleFunc("/v1/json/routing", func(w http.ResponseWriter, r *http.Request) {

		// Query
		if q, err := GetQuery(r, g); err != nil {
			log.Fatalln(err)
		} else {
			memo := routing.RAPTOR(raptorData, q)

			pos := q.ToStop
			ro := q.Round - 1

			legs := []models.LegStr{}

			for pos != q.FromStop {
				bef := memo.Tau[ro][pos]
				now := pos
				if bef.ArrivalTime == 0 {
					fmt.Println("not found !")
					break
				}
				pos = bef.BeforeStop

				viaNodes := []models.StopTimeStr{}
				on := false
				
				tripId := string(memo.Tau[ro][now].BeforeEdge)
				routePattern := raptorData.TripId2StopPatternIndex[tripId]
				tripIndex := raptorData.TripId2Index[tripId]

				for _,v := range raptorData.TimeTables[q.Date].StopPatterns[routePattern].Trips[tripIndex].StopTimes{
					if v.StopID == bef.BeforeStop {
						on = true
					}
					if on {
						stopId := v.StopID
						viaNodes = append(viaNodes, models.StopTimeStr{
							StopId:    string(stopId),
							StopLat:   g.Stops[raptorData.StopId2Index[stopId]].Latitude,
							StopLon:   g.Stops[raptorData.StopId2Index[stopId]].Longitude,
							StopName: g.Stops[raptorData.StopId2Index[stopId]].Name,
							ArrivalTime: v.Arrival,
							DepartureTime: v.Departure,
						})
					}
					if v.StopID == now{
						break
					}
				}

				legs = append(legs, models.LegStr{
					Type: "bus",
					Trip:			 models.GTFSTripStr{
						TripId: string(memo.Tau[ro][now].BeforeEdge),
					},
					StopTimes: viaNodes,
					TimeEdges: []models.TimeEdgeStr{},
					Geometry: NewLineString([][]float64{
						[]float64{g.Stops[raptorData.StopId2Index[bef.BeforeStop]].Longitude, g.Stops[raptorData.StopId2Index[bef.BeforeStop]].Latitude},
						[]float64{g.Stops[raptorData.StopId2Index[now]].Longitude, g.Stops[raptorData.StopId2Index[now]].Latitude},
					}, nil),
				})
				ro = ro - 1
			}

			json.DumpToWriter(models.ResponsStr{
				Trips: []models.TripStr{
					models.TripStr{
						Legs: legs,
					},
				},
			}, w)
		}
	})
	http.HandleFunc("/routing_geojson", func(w http.ResponseWriter, r *http.Request) {

		if q, err := GetQuery(r, g); err != nil {
			log.Fatalln(err)
		} else {
			// Query
			Round := 10

			memo := routing.RAPTOR(raptorData, q)

			ro := Round - 1

			fc := NewFeatureCollection()
			for stopId, m := range memo.Tau[ro] {
				s := g.Stops[raptorData.StopId2Index[stopId]]
				props := map[string]string{}
				props["time"] = strconv.Itoa(m.ArrivalTime - q.FromTime)
				props["arrival_time"] = pkg.Sec2HHMMSS(m.ArrivalTime)
				props["stop_id"] = stopId
				props["name"] = s.Name
				tr := ro
				for tr >= 0 {
					if memo.Tau[tr][stopId].ArrivalTime != m.ArrivalTime {
						break
					}
					tr--
				}
				props["transfer"] = strconv.Itoa(tr)
				fc.Features = append(fc.Features, Feature{
					Type: "Feature",
					Geometry: Geometry{
						Type:        "Point",
						Coordinates: []float64{s.Longitude, s.Latitude},
					},
					Properties: props,
				})
			}
			json.DumpToWriter(fc, w)
		}
	})
	http.HandleFunc("/routing_surface", func(w http.ResponseWriter, r *http.Request) {

		if q, err := GetQuery(r, g); err != nil {
			log.Fatalln(err)
		} else {
			StopId2Index := map[string]int{}
			for i, stop := range g.Stops {
				StopId2Index[stop.ID] = i
			}

			fc := NewFeatureCollection()

			memo := routing.RAPTOR(raptorData, q)
			for r, m := range memo.Tau {
				for k, v := range m {
					props := map[string]string{}
					props["stop_id"] = k
					props["arrival_time"] = pkg.Sec2HHMMSS(v.ArrivalTime)
					props["stop_name"] = g.Stops[StopId2Index[k]].Name
					props["round"] = strconv.Itoa(r)
					fc.Features = append(fc.Features, Feature{
						Type: "Feature",
						Geometry: Geometry{
							Type:        "Point",
							Coordinates: []float64{g.Stops[StopId2Index[k]].Longitude, g.Stops[StopId2Index[k]].Latitude},
						},
						Properties: props,
					})
				}
			}
			json.DumpToWriter(fc, w)
		}
	})
	fmt.Println("start server.")
	if err := http.ListenAndServe("0.0.0.0:8000", nil);err != nil {
		log.Fatalln(err)
	}
}

func FindODNode(qns QueryNodeStr, g *gtfs.GTFS) string {
	stopId := ""
	minD := math.MaxFloat64
	if qns.StopId == nil {
		for _, stop := range g.Stops {
			d := geometry.HubenyDistance(goraph.LatLon{
				Lat: stop.Latitude,
				Lon: stop.Longitude},
				goraph.LatLon{
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
type QueryStr struct {
	Origin        QueryNodeStr  `json:"origin"`
	Destination   QueryNodeStr  `json:"destination"`
	LimitTime     int           `json:"limit_time"`
	LimitTransfer int           `json:"limit_transfer"`
	WalkSpeed     float64       `json:"walk_speed"`
	Property      QueryProperty `json:"properties"`
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
	if query.LimitTime == 0 {
		query.LimitTime = 3600 * 10
	}
	if query.LimitTransfer == 0 {
		query.LimitTransfer = 5
	}
	if query.WalkSpeed == 0 {
		query.WalkSpeed = 80
	}

	fmt.Println(query.Origin,query.Destination)

	return &routing.Query{
		ToStop:      FindODNode(query.Destination, g),
		FromStop:    FindODNode(query.Origin, g),
		FromTime:    *query.Origin.Time,
		MinuteSpeed: query.WalkSpeed,
		Round:       query.LimitTransfer,
		LimitTime:   *query.Origin.Time + query.LimitTime,
		Date:        query.Property.Timetable,
	}, nil
}
