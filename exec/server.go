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
	"github.com/takoyaki-3/go-gtfs/tool"
	json "github.com/takoyaki-3/go-json"
	"github.com/takoyaki-3/goraph"
	"github.com/takoyaki-3/goraph/geometry"
	fare "github.com/takoyaki-3/go-gtfs-fare"
)

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

				latlons := [][]float64{}

				for _, v := range raptorData.TimeTables[q.Date].StopPatterns[routePattern].Trips[tripIndex].StopTimes {
					if v.StopID == bef.BeforeStop {
						on = true
					}
					if on {
						stopId := v.StopID
						s := g.Stops[raptorData.StopId2Index[stopId]]
						viaNodes = append(viaNodes, models.StopTimeStr{
							StopId:        string(stopId),
							StopLat:       s.Latitude,
							StopLon:       s.Longitude,
							StopName:      s.Name,
							ArrivalTime:   v.Arrival,
							DepartureTime: v.Departure,
						})
						latlons = append(latlons, []float64{s.Longitude, s.Latitude})
					}
					if v.StopID == now {
						break
					}
				}

				// trip情報の取得
				trip := tool.GetTrip(g, string(memo.Tau[ro][now].BeforeEdge))
				route := tool.GetRoute(g, trip.RouteID)
				headSign := ""
				if len(viaNodes) > 0 {
					headSign = tool.GetHeadSign(g, trip.ID, viaNodes[0].StopId)

					from := viaNodes[0]
					to := viaNodes[len(viaNodes)-1]
					p,err := fare.GetFareAttribute(raptorData.Fare,from.StopId,to.StopId,trip.RouteID)
					if err != nil {
						p = fare.FareAttribute{
							Price: -1,
						}
					}
					cost := models.NewCostStr()
					*cost.Fare = p.Price
					*cost.Distance = -1
					*cost.Time = float64(pkg.HHMMSS2Sec(to.ArrivalTime) - pkg.HHMMSS2Sec(from.DepartureTime))
					*cost.Transfer = 0
					*cost.Walk = 0

					legs = append([]models.LegStr{models.LegStr{
						Type: "bus",
						Trip: models.GTFSTripStr{
							TripId:          trip.ID,
							TripDescription: trip.DirectionID,
							RouteLongName:   route.LongName,
							ServiceId:       trip.ServiceID,
							TripType:        strconv.Itoa(route.Type),
							RouteColor:      route.Color,
							RouteTextColor:  route.TextColor,
							RouteShortName:  route.ShortName,
							TripHeadSign:    headSign,
							RouteId:         trip.RouteID,
						},
						StopTimes: viaNodes,
						TimeEdges: []models.TimeEdgeStr{},
						Geometry:  NewLineString(latlons, nil),
						Costs: cost,
					}},legs...)
				}
				ro = ro - 1
			}

			// コストの合計
			c := models.NewCostStr()
			for _,leg := range legs{
				c = models.CostAdder(c,leg.Costs)
			}

			json.DumpToWriter(models.ResponsStr{
				Trips: []models.TripStr{
					models.TripStr{
						Legs: legs,
						Costs: c,
					},
				},
			}, w)
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
	if err := http.ListenAndServe("0.0.0.0:8000", nil); err != nil {
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
