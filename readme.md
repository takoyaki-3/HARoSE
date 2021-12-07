
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

[http://localhost:8000/routing_geojson?json={"origin":{"lat":35.498816884357325,"lon":139.66437429943267,"time":28800},"destination":{"lat":35.51179352021772,"lon":139.6039574427234},"json_only":true}](http://localhost:8000/routing_geojson?json={"origin":{"lat":35.498816884357325,"lon":139.66437429943267,"time":28800},"destination":{"lat":35.51179352021772,"lon":139.6039574427234},"json_only":true})のようにjsonに出発地と到着地の緯度経度を指定する。

表示されたGeoJSONを右クリックで保存し、[kepler.gl](https://kepler.gl/demo)にドラックアンドドロップすると可視化される。