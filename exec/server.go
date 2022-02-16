package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/MaaSTechJapan/raptor/loader"
	"github.com/MaaSTechJapan/raptor/routing"
	"github.com/google/uuid"
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
	Transportation string     `json:"transportation"`
	load           string     `json:"load"`
	WKT            string     `json:WKT`
	Geometry       Geometry   `json:"geometry"`
}

func main() {

	raptorData, g, err := loader.LoadGTFS()
	if err != nil {
		log.Fatalln(err)
	}

	StopId2Index := map[string]int{}
	for i, stop := range g.Stops {
		StopId2Index[stop.ID] = i
	}

	mapStops := map[string]int{}
	for i, s := range g.Stops {
		mapStops[s.ID] = i
	}

	// index.html
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		bytes, _ := ioutil.ReadFile("./index.html")
		fmt.Fprintln(w, string(bytes))
	})
	http.HandleFunc("/routing", func(w http.ResponseWriter, r *http.Request) {

		// Query
		if q, err := GetQuery(r, g); err != nil {
			log.Fatalln(err)
		} else {
			memo := routing.RAPTOR(raptorData, q)

			pos := q.ToStop
			ro := q.Round - 1

			legs := []MTJLegStr{}

			fmt.Println("---")
			lastTime := memo.Tau[ro][pos].ArrivalTime
			for pos != q.FromStop {
				bef := memo.Tau[ro][pos]
				now := pos
				if bef.ArrivalTime == 0 {
					fmt.Println("not found !")
					break
				}
				fmt.Println(bef, pkg.Sec2HHMMSS(bef.ArrivalTime), pkg.Sec2HHMMSS(lastTime), g.Stops[StopId2Index[bef.BeforeStop]].Name, "->", g.Stops[StopId2Index[pos]].Name)
				lastTime = bef.ArrivalTime
				pos = bef.BeforeStop

				uuidObj, _ := uuid.NewUUID()
				id := uuidObj.String()
				legs = append(legs, MTJLegStr{
					FromNode: MTJNodeStr{
						Id:    string(bef.BeforeStop),
						Lat:   g.Stops[StopId2Index[pos]].Latitude,
						Lon:   g.Stops[StopId2Index[pos]].Longitude,
						Title: g.Stops[StopId2Index[bef.BeforeStop]].Name,
					},
					ToNode: MTJNodeStr{
						Id:    string(now),
						Lat:   g.Stops[StopId2Index[now]].Latitude,
						Lon:   g.Stops[StopId2Index[now]].Longitude,
						Title: g.Stops[StopId2Index[now]].Name,
					},
					Transportation: string(memo.Tau[ro][now].BeforeEdge),
					Id:             id,
					Uid:            id,
					Oid:            id,
					Created:        time.Now().Format("2006-01-02 15:04:05"),
					Geometry: *NewLineString([][]float64{
						[]float64{g.Stops[StopId2Index[bef.BeforeStop]].Longitude, g.Stops[StopId2Index[bef.BeforeStop]].Latitude},
						[]float64{g.Stops[StopId2Index[now]].Longitude, g.Stops[StopId2Index[now]].Latitude},
					}, nil),
				})
				ro = ro - 1
			}

			type TripStr struct {
				Legs []MTJLegStr `json:"legs"`
			}
			type Resp struct {
				Trips []TripStr `json:"trips"`
			}
			json.DumpToWriter(Resp{
				Trips: []TripStr{
					TripStr{
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
				s := g.Stops[mapStops[stopId]]
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
	http.ListenAndServe("0.0.0.0:8000", nil)
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
