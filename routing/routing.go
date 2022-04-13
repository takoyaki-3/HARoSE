package routing

import (
	"sort"

	. "github.com/MaaSTechJapan/raptor"
	gtfs "github.com/takoyaki-3/go-gtfs/v2"
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
	fromStop := query.FromStop
	fromTime := query.FromTime

	// 初期化
	memo.Tau = make([]map[string]NodeMemo, query.Round)
	for k, _ := range memo.Tau {
		memo.Tau[k] = map[string]NodeMemo{}
	}
	memo.Marked = []string{}

	// 出発停留所
	memo.Tau[0][fromStop] = NodeMemo{
		ArrivalTime: fromTime,
		BeforeStop:  "init",
		BeforeEdge:  "init",
	}
	memo.Marked = append(memo.Marked, fromStop)

	for r := 1; r <= query.Round; r++ {
		newMarked := map[string]bool{}

		// step-1 前のラウンドからコピー
		// local pruning時は不要
		for stopId, n := range memo.Tau[r-1] {
			memo.Tau[r][stopId] = n
		}

		// step-2 路線ごとにscanし、tauを更新
		for _, fromStopId := range memo.Marked {
			for _, routePatternId := range data.TimeTables[query.Date].StopRoutes[fromStopId] {
				for _, trip := range data.TimeTables[query.Date].StopPatterns[routePatternId].Trips {
					riding := false
					if gtfs.HHMMSS2Sec(trip.StopTimes[len(trip.StopTimes)-1].Arrival) < memo.Tau[r][fromStopId].ArrivalTime {
						continue
					}
					for _, stopTime := range trip.StopTimes {
						if riding {
							isUpdate := false
							if v, ok := memo.Tau[r][stopTime.StopID]; ok {
								if gtfs.HHMMSS2Sec(stopTime.Arrival) < v.ArrivalTime {
									isUpdate = true
								}
							} else {
								isUpdate = true
							}
							if isUpdate {
								memo.Tau[r][stopTime.StopID] = NodeMemo{
									ArrivalTime: gtfs.HHMMSS2Sec(stopTime.Arrival),
									BeforeStop:  fromStopId,
									BeforeEdge:  trip.Properties.TripID,
								}
								newMarked[stopTime.StopID] = true
							}
						} else {
							if stopTime.StopID == fromStopId {
								if gtfs.HHMMSS2Sec(stopTime.Departure) < memo.Tau[r][fromStopId].ArrivalTime {
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
		// memo.Marked周りを修正する
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

		// ここも違う
		for k, _ := range newMarked {
			memo.Marked = append(memo.Marked, k)
		}
		sort.Slice(memo.Marked, func(i, j int) bool {
			return memo.Marked[i] < memo.Marked[j]
		})
	}

	return memo
}
