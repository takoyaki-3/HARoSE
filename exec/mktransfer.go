package main

import (
	"fmt"
	"log"
	"time"

	csvtag "github.com/artonge/go-csv-tag/v2"

	"github.com/takoyaki-3/go-gtfs"
	"github.com/takoyaki-3/go-gtfs/tool"
	"github.com/takoyaki-3/goraph"
	"github.com/takoyaki-3/goraph/geometry/h3"
	"github.com/takoyaki-3/goraph/loader/osm"
	"github.com/takoyaki-3/goraph/search"
)

type Line struct {
	FromLat float64 `csv:"from_lat"`
	FromLon float64 `csv:"from_lon"`
	ToLat   float64 `csv:"to_lat"`
	ToLon   float64 `csv:"to_lon"`
}

func main() {
	g, err := gtfs.Load("./GTFS", nil)
	// 日付をベースとした絞り込み
	g = tool.ExtractByDate(g, time.Now())

	lines := []Line{}
	err = csvtag.LoadFromPath("line.csv", &lines)
	if err != nil {
		log.Fatalln("Error loading file (%v)\n	==> %v", "input.csv", err)
	}

	// for i,stopI:=range g.Stops{
	// 	for j,stopJ:=range g.Stops{
	// 		if i!=j{
	// 			lines = append(lines, Line{
	// 				FromLat: ,
	// 			})
	// 		}
	// 	}
	// }

	// 地図データ読み込み
	road := osm.Load("./map.osm.pbf")
	h3index := h3.MakeH3Index(road, 9)

	for _, line := range lines {
		route := search.Search(road, search.Query{
			From: h3.Find(road, h3index, goraph.LatLon{
				Lat: line.FromLat,
				Lon: line.FromLon,
			}, 9),
			To: h3.Find(road, h3index, goraph.LatLon{
				Lat: line.ToLat,
				Lon: line.ToLon,
			}, 9),
		})
		fmt.Println(route)
	}
}
