## iwps

using google/apple's [WPS](https://en.wikipedia.org/wiki/Wi-Fi_positioning_system) to try location experiments


<!-- TEST: 3a:80:88:ef:62:89 -->
```sh
protoc --go_out=. iwps.proto
go run index.go <BSSID>
```