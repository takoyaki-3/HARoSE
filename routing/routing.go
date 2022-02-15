package routing

import (
	"sort"

	. "github.com/MaaSTechJapan/raptor"
	"github.com/takoyaki-3/go-gtfs/pkg"
)

type Query struct {
	FromTime    int
	FromStop    string
	ToStop      string
	MinuteSpeed float64
	Round       int
	LimitTime   int
	Date        string
}

type NodeMemo struct {
	ArrivalTime int
	BeforeStop  string
	BeforeEdge  string
}

type Memo struct {
	Tau    []map[string]NodeMemo
	Marked []string
}

func RAPTOR(data *RAPTORData, query *Query) (memo Memo) {

	// Buffer
	Round := query.Round
	fromStop := query.FromStop
	fromTime := query.FromTime

	memo.Tau = make([]map[string]NodeMemo, Round)
	for k, _ := range memo.Tau {
		memo.Tau[k] = map[string]NodeMemo{}
	}
	memo.Marked = []string{}

	memo.Tau[0][fromStop] = NodeMemo{
		ArrivalTime: fromTime,
		BeforeStop:  "init",
		BeforeEdge:  "init",
	}
	memo.Marked = append(memo.Marked, fromStop)

	for r := 0; r < Round-1; r++ {
		newMarked := map[string]bool{}

		// Tau
		for _, fromStopId := range memo.Marked {
			for _, routePatternId := range data.TimeTables[query.Date].StopRoutes[fromStopId] {
				for _, trip := range data.TimeTables[query.Date].StopPatterns[routePatternId].Trips {
					riding := false
					// sort.Slice(trip.StopTimes,func(i, j int) bool {
					// 	return trip.StopTimes[i].Departure < trip.StopTimes[j].Departure
					// })
					if pkg.HHMMSS2Sec(trip.StopTimes[len(trip.StopTimes)-1].Arrival) < memo.Tau[r][fromStopId].ArrivalTime {
						continue
					}
					for _, stopTime := range trip.StopTimes {
						if riding {
							isUpdate := false
							if v, ok := memo.Tau[r][stopTime.StopID]; ok {
								if pkg.HHMMSS2Sec(stopTime.Arrival) < v.ArrivalTime {
									isUpdate = true
								}
							} else {
								isUpdate = true
							}
							if isUpdate {
								memo.Tau[r][stopTime.StopID] = NodeMemo{
									ArrivalTime: pkg.HHMMSS2Sec(stopTime.Arrival),
									BeforeStop:  fromStopId,
									BeforeEdge:  trip.Properties.TripID,
								}
								newMarked[stopTime.StopID] = true
							}
						} else {
							if stopTime.StopID == fromStopId {
								if pkg.HHMMSS2Sec(stopTime.Departure) < memo.Tau[r][fromStopId].ArrivalTime {
									break
								}
								riding = true
							}
						}
					}
					if riding {
						break
					}
				}
			}
		}

		// 乗換
		for _, fromStopId := range memo.Marked {
			if memo.Tau[r][fromStopId].BeforeEdge == "transfer" {
				continue
			}
			for toStopId, v := range data.Transfer[fromStopId] {
				transArrivalTime := memo.Tau[r][fromStopId].ArrivalTime + int(v/query.MinuteSpeed*60)
				isUpdate := false
				if m, ok := memo.Tau[r][toStopId]; ok {
					if m.ArrivalTime > transArrivalTime {
						isUpdate = true
					}
				} else {
					isUpdate = true
				}
				if isUpdate {
					memo.Tau[r][toStopId] = NodeMemo{
						ArrivalTime: transArrivalTime,
						BeforeStop:  fromStopId,
						BeforeEdge:  "transfer",
					}
					newMarked[toStopId] = true
				}
			}
		}

		// そのまま待機
		if r != Round-1 {
			for stopId, n := range memo.Tau[r] {
				memo.Tau[r+1][stopId] = n
			}
		}

		for k, _ := range newMarked {
			memo.Marked = append(memo.Marked, k)
		}
		sort.Slice(memo.Marked, func(i, j int) bool {
			return memo.Marked[i] < memo.Marked[j]
		})
	}

	return memo
}
