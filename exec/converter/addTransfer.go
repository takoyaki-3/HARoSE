package main

import (
	"log"

	gtfs "github.com/takoyaki-3/go-gtfs/v2"
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
	if err := json.LoadFromPath("../original_data/conf.json", &conf); err != nil {
		log.Fatalln(err)
	}

	if g, err := gtfs.Load("../original_data/"+conf.GTFS.Path, nil); err != nil {
		log.Fatalln(err)
	} else {
		if err = g.AddTransferWithOSM(conf.ConnectRange, conf.WalkingSpeed, "../original_data/"+conf.Map.FileName, conf.NumThread);err!=nil{
			log.Fatalln(err)
		}
		if err := gtfs.Dump(g, "../original_data/"+conf.GTFS.Path, nil); err != nil {
			log.Fatalln(err)
		}
	}
}
