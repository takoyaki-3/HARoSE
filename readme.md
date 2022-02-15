
## 必要なもの

- golang が動く環境

## 使い方

GTFSが１つの場合

``/GTFS``に展開して格納。

サーバ起動

```
$ go run .\server.go
```

アクセス

a) 1対1の経路検索をしたい場合

[http://localhost:8000/](http://localhost:8000/) にアクセスし、適当な2点をクリックすると経路が検索される

b) 到達圏検索をしたい場合

```
http://localhost:8000/routing_surface?json={%22origin%22:{%22time%22:28800,%22lat%22:34.38291392395102,%22lon%22:132.4260767353174},%22destination%22:{%22time%22:32400,%22lat%22:34.39101899572567,%22lon%22:132.45094923488853},%22json_only%22:true,%22properties%22:{%22timetable%22:%2220210215%22}}
```
のようにjsonに出発地と到着地の緯度経度を指定する。

表示されたGeoJSONを右クリックで保存し、[kepler.gl](https://kepler.gl/demo)にドラックアンドドロップすると可視化される。