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
	WalkTransfer bool // r回目の乗車後に徒歩移動をする or not
}

type Memo struct {
	Tau    []map[string]NodeMemo // ラウンドごとの各停留所への最も早い到着時刻
	Marked []string              // memoの中でなくてよいのでは？
}

func RAPTOR(data *RAPTORData, query *Query) (memo Memo) {

	// Buffer
	fromStop := query.FromStop
	fromTime := query.FromTime
	//fmt.Println("From:", fromStop, fromTime)

	// timetable extracted by date
	timetable := data.TimeTables[query.Date]

	// 初期化
	memo.Tau = make([]map[string]NodeMemo, query.Round+1)
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

	// 近くの停留所への徒歩接続
	for toStopId, v := range data.Transfer[fromStop] {
		memo.Tau[0][toStopId] = NodeMemo{
			ArrivalTime:  fromTime + int(v),
			BoardingTrip: "init",
			GetonStop:    fromStop,
			GetoffStop:   fromStop,
			WalkTransfer: true,
		}
	}

	for r := 1; r <= query.Round; r++ {
		newMarked := []string{}
		//fmt.Println("round", r)

		// step-1 前のラウンドからコピー
		// local pruning時は不要
		for stopId, n := range memo.Tau[r-1] {
			memo.Tau[r][stopId] = n
		}
		//fmt.Println("step 1 is done.")

		// scan対象の路線：前のラウンドでmarkされた停留所を経由する路線
		Q := map[int]string{}
		for _, stopId := range memo.Marked {
			for _, routeIndex := range timetable.StopRoutes[stopId] {
				// 各路線で、最も始点側でmarkされた停留所を紐づける
				if anotherStop, ok := Q[routeIndex]; ok {
					// stopIdがその経路の最も始点側なら更新
					if timetable.RouteStop2StopSeq[routeIndex][stopId] < timetable.RouteStop2StopSeq[routeIndex][anotherStop] {
						Q[routeIndex] = stopId
					}
				} else {
					Q[routeIndex] = stopId
				}
			}
		}
		//fmt.Println("step 2: routes are accumulated in set Q.")

		// step-2 路線ごとにscanし、tauを更新
		for routeIndex, fromStopId := range Q {
			stopPattern := timetable.RoutePatterns[routeIndex]

			// 当該路線で最初にcatchできる便
			currentTrip := -1 // int型

			// currentTripに乗車する停留所
			boardingStop := fromStopId

			// fromStop以降の停留所を順にたどる
			fromStopIndex := timetable.RouteStop2StopSeq[routeIndex][fromStopId]
			//fmt.Println(routeIndex, fromStopId, fromStopIndex)

			for i, stopId := range timetable.RouteStops[routeIndex] {
				if i < fromStopIndex {
					continue
				}
				//fmt.Println("Stop", stopId, "currentTrip", currentTrip)

				// tau更新
				// fromStopにいる場合はskipされる
				if currentTrip >= 0 && currentTrip < len(stopPattern.Trips) {
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
					if currentTrip == -1 {
						// 存在しなかった場合
						currentTrip = len(stopPattern.Trips)
					}
				} else if v, ok := memo.Tau[r-1][stopId]; ok {
					/*if v.ArrivalTime >= gtfs.HHMMSS2Sec(stopPattern.Trips[currentTrip].StopTimes[i].Arrival) {
						continue
					}*/
					// 前ラウンドでの到着時刻が、current tripでの到着時刻より早い場合
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

		//fmt.Println("step 2 is done.")

		// step-3 徒歩乗換の処理
		for _, fromStopId := range memo.Marked {
			for toStopId, minTransferTime := range data.Transfer[fromStopId] {
				transArrivalTime := memo.Tau[r][fromStopId].ArrivalTime + int(minTransferTime)
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
		//fmt.Println("step 3 is done.")
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
