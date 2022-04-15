package routing

import (
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
	ArrivalTime  int
	BoardingTrip string
	GetonStop    string
	GetoffStop   string
	WalkTransfer bool
}

type Memo struct {
	Tau    []map[string]NodeMemo // ラウンドごとの各停留所への最も早い到着時刻
	Marked []string              // memoの中でなくてよいのでは？
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
		ArrivalTime:  fromTime,
		BoardingTrip: "init",
		GetonStop:    fromStop,
		GetoffStop:   fromStop,
		WalkTransfer: false,
	}
	memo.Marked = append(memo.Marked, fromStop)

	for r := 1; r <= query.Round; r++ {
		newMarked := []string{}

		// step-1 前のラウンドからコピー
		// local pruning時は不要
		for stopId, n := range memo.Tau[r-1] {
			memo.Tau[r][stopId] = n
		}

		// scan対象の路線：前のラウンドでmarkされた停留所を経由する路線
		Q := map[int]string{}
		for _, stopId := range memo.Marked {
			for _, routeIndex := range data.TimeTables[query.Date].StopRoutes[stopId] {
				// 各路線で、最も始点側でmarkされた停留所を紐づける
				if anotherStop, ok := Q[routeIndex]; ok {
					// stopIdがその経路の最も始点側なら更新
					if data.RouteStop2StopSeq[routeIndex][stopId] < data.RouteStop2StopSeq[routeIndex][anotherStop] {
						Q[routeIndex] = stopId
					}
				} else {
					Q[routeIndex] = stopId
				}
			}
		}

		// step-2 路線ごとにscanし、tauを更新
		for routeIndex, fromStopId := range Q {
			stopPattern := data.TimeTables[query.Date].StopPatterns[routeIndex]

			// 当該路線で最初にcatchできる便
			currentTrip := -1 // int型

			// currentTripに乗車する停留所
			boardingStop := fromStopId

			// fromStop以降の停留所を順にたどる
			fromStopIndex := data.RouteStop2StopSeq[routeIndex][fromStopId]
			for i, stopId := range data.RouteStops[routeIndex] {
				if i < fromStopIndex {
					continue
				}

				// tau更新
				// fromStopにいる場合はskipされる
				if currentTrip != -1 {
					trip := stopPattern.Trips[currentTrip]
					isUpdate := false
					if v, ok := memo.Tau[r-1][stopId]; ok {
						if gtfs.HHMMSS2Sec(trip.StopTimes[i].Arrival) < v.ArrivalTime {
							isUpdate = true
						}
					} else {
						isUpdate = true
					}
					if isUpdate {
						memo.Tau[r][stopId] = NodeMemo{
							ArrivalTime:  gtfs.HHMMSS2Sec(trip.StopTimes[i].Arrival),
							BoardingTrip: trip.Properties.TripID,
							GetonStop:    boardingStop,
							GetoffStop:   stopId,
							WalkTransfer: false,
						}
						newMarked = append(newMarked, stopId)
					}
				}

				// current tripの更新
				if i == fromStopIndex {
					// fromStopにいる場合、tripを始発側から検索
					for t, trip := range stopPattern.Trips {
						if memo.Tau[r-1][fromStopId].ArrivalTime <= gtfs.HHMMSS2Sec(trip.StopTimes[i].Departure) {
							currentTrip = t
							break
						}
					}
				} else if v, ok := memo.Tau[r-1][stopId]; ok {
					// current tripの到着より前ラウンドでの到着が早い場合
					if v.ArrivalTime >= gtfs.HHMMSS2Sec(stopPattern.Trips[currentTrip].StopTimes[i].Arrival) {
						continue
					}
					for currentTrip > 0 {
						// 1本前の便に乗れるなら、currentTripの値を1減らす
						if v.ArrivalTime < gtfs.HHMMSS2Sec(stopPattern.Trips[currentTrip-1].StopTimes[i].Arrival) {
							currentTrip -= 1
							boardingStop = stopId
						} else {
							break
						}
					}
				}
			}
		}

		// marked stopを再構成
		memo.Marked = nil
		memo.Marked = append(memo.Marked, newMarked...)

		// step-3 徒歩乗換の処理
		for _, fromStopId := range memo.Marked {
			/*
				if memo.Tau[r][fromStopId].BeforeEdge == "transfer" {
					continue
				}
			*/
			for toStopId, v := range data.Transfer[fromStopId] {
				// v: MinTime [単位:秒]
				//transArrivalTime := memo.Tau[r][fromStopId].ArrivalTime + int(v/query.MinuteSpeed*60)
				transArrivalTime := memo.Tau[r][fromStopId].ArrivalTime + int(v)
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
						ArrivalTime:  transArrivalTime,
						BoardingTrip: memo.Tau[r][fromStopId].BoardingTrip,
						GetonStop:    memo.Tau[r][fromStopId].GetonStop,
						GetoffStop:   memo.Tau[r][fromStopId].GetoffStop,
						WalkTransfer: true,
					}
					newMarked = append(newMarked, toStopId)
				}
			}
		}

		// marked stopを再構成
		memo.Marked = nil
		memo.Marked = append(memo.Marked, newMarked...)
		/*
			// marked stopをソート
			// たぶんいらない
			sort.Slice(memo.Marked, func(i, j int) bool {
				return memo.Marked[i] < memo.Marked[j]
			})
		*/
	}

	return memo
}
