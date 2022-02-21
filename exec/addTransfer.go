package main

import (
	"log"

	"github.com/takoyaki-3/go-gtfs"
	"github.com/takoyaki-3/go-gtfs/tool"
	json "github.com/takoyaki-3/go-json"
)

type ConfMap struct {
	FileName string `json:"file_name"`
}
type ConfGtfs struct {
	Path string `json:"path"`
}

type Conf struct {
	GTFS         ConfGtfs `json:"gtfs"`
	Map          ConfMap  `json:"map"`
	ConnectRange float64  `json:"connect_range"`
	WalkingSpeed float64  `json:"walking_speed"`
	NumThread    int      `json:"num_threads"`
}

func main() {

	var conf Conf
	if err := json.LoadFromPath("./conf.json", &conf); err != nil {
		log.Fatalln(err)
	}

	if g, err := gtfs.Load(conf.GTFS.Path, nil); err != nil {
		log.Fatalln(err)
	} else {
		tool.MakeTransferWithOSM(g, conf.ConnectRange, conf.WalkingSpeed, conf.Map.FileName, conf.NumThread)
		if err := gtfs.Dump(g, conf.GTFS.Path, nil); err != nil {
			log.Fatalln(err)
		}
	}
}
