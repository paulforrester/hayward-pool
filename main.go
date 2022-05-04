package main

import (
	"fmt"
)

//const (
//   POOL_TEMP_TARGET_INVALID = -1
//)

var config Config

func main() {

	fmt.Println("pool-data-collector polls a Hayward Aqua Connect Local network device.")

	c := influxDBClient(config)

	go update_datastore(c, config)
	watch_http_endpoint(config)
}
