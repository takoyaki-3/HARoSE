package main

import (
	"fmt"
	"sync"

	"github.com/takoyaki-3/go-gtfs/pkg"

	// "github.com/takoyaki-3/go-gtfs/stop_pattern"

	// "github.com/takoyaki-3/goraph/loader/osm"
	// "github.com/takoyaki-3/goraph/geometry/h3"
	// "github.com/takoyaki-3/goraph"
	// "github.com/takoyaki-3/goraph/search"
	// "github.com/takoyaki-3/goraph/geometry"
	"github.com/MaaSTechJapan/raptor/loader"
	"github.com/MaaSTechJapan/raptor/routing"
)

func main() {

	raptorData, g := loader.LoadGTFS()

	q := &routing.Query{
		FromTime: 3600 * 4,
		FromStop: "0605-01",
		ToStop:   "0702-01",
		// ToStop: "2189-01",
		MinuteSpeed: 80,
		Round:       20,
		LimitTime:   36000,
	}
	// for r:=1;r<100;r++{
	// 	old := time.Now()
	// 	for i:=0;i<1000;i++{
	// 		q.Round = r
	// 		routing.RAPTOR(raptorData,q)
	// 	}
	// 	fmt.Println(r,time.Now().Sub(old).Milliseconds())
	// }

	StopId2Index := map[string]int{}
	for i, stop := range g.Stops {
		StopId2Index[stop.ID] = i
	}

	numThread := 8
	fmt.Println("Start routing.")
	wg := sync.WaitGroup{}
	wg.Add(numThread)
	for rank := 0; rank < numThread; rank++ {
		go func(rank int) {
			defer wg.Done()
			for i, stop := range g.Stops {
				if i%numThread != rank {
					continue
				}
				q.FromStop = stop.ID
				memo := routing.RAPTOR(raptorData, q)
				notChange := -1
				for r, _ := range memo.Tau {
					if r > 0 {
						if len(memo.Tau[r-1]) != len(memo.Tau[r]) {
							notChange = r
						}
					}
					// fmt.Println(r,len(m),len(g.Stops),float64(len(m)*100)/float64((len(g.Stops))))
				}
				fmt.Println(notChange, float64(len(memo.Tau[q.Round-1])*100)/float64(len(g.Stops)), g.Stops[StopId2Index[stop.ID]].Name)
			}
		}(rank)
	}
	wg.Wait()

	memo := routing.RAPTOR(raptorData, q)
	minTimeRoute := -1
	for r, m := range memo.Tau {
		if m, ok := m[q.ToStop]; ok {
			fmt.Println(r, m, pkg.Sec2HHMMSS(m.ArrivalTime))
			if r > 0 {
				if memo.Tau[r-1][q.ToStop].ArrivalTime > m.ArrivalTime {
					minTimeRoute = r
				}
				if memo.Tau[r-1][q.ToStop].ArrivalTime == 0 {
					minTimeRoute = r
				}
			}
		}
	}
	fmt.Println(minTimeRoute)

	pos := q.ToStop
	for r := minTimeRoute; r >= 0; r-- {
		m := memo.Tau[r][pos]
		fmt.Println(r, pos, pkg.Sec2HHMMSS(m.ArrivalTime), m)
		pos = m.BeforeStop
	}
}
