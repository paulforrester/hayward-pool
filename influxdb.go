package main

import (
	//	"fmt"
	"github.com/influxdata/influxdb1-client/v2"
	"log"
	"time"
)

// CREATE USER admin WITH PASSWORD '$the_usual' WITH ALL PRIVILEGES
// create database BLAH

func influxDBClient(config Config) client.Client {
	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     config.DatabaseURL,
		Username: config.DatabaseUser,
		Password: config.DatabasePassword,
	})
	if err != nil {
		log.Fatalln("Error: ", err)
	}
	return c
}

func influx_post_one_metric(bp client.BatchPoints, key string, tags map[string]string, field_label string , eventTime time.Time, m *Measurement) {
	fields := map[string]interface{}{
	    field_label: m.Reading,
	}
	
	point, err := client.NewPoint(key, tags, fields, eventTime)
	if err != nil {
		log.Fatalln("Error: ", err)
	}
	if m.Reading != NOT_RECORDED {
		bp.AddPoint(point)
	}
}

func influx_push_metrics(c client.Client, config Config) {
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  config.DatabaseDatabase,
		Precision: "s",
	})

	if err != nil {
		log.Fatalln("Error: ", err)
	}

	eventTime := time.Now()

	/*
		Using "Line Protocol", eg: cpu,host=server02,region=uswest value=3 1434055562000010000
		http://goinbigdata.com/working-with-influxdb-in-go/

		key: pool
		tags: none
		fields: pool_temp=blah, etc.
		timestamp in seconds
	*/

	key := "pool"
	tags := map[string]string{}
	
	influx_post_one_metric(bp, key, tags, "air_temp", eventTime, &pool.AirTempF)
	influx_post_one_metric(bp, key, tags, "pool_temp", eventTime, &pool.PoolTempF)
	influx_post_one_metric(bp, key, tags, "filter_speed", eventTime, &pool.FilterSpeedRPM)
	influx_post_one_metric(bp, key, tags, "salt_ppm", eventTime, &pool.SaltPPM)
	influx_post_one_metric(bp, key, tags, "filter_on", eventTime, &pool.FilterOn)
	influx_post_one_metric(bp, key, tags, "cleaner_on", eventTime, &pool.CleanerOn)
	influx_post_one_metric(bp, key, tags, "lights_on", eventTime, &pool.LightOn)
	influx_post_one_metric(bp, key, tags, "chlorinator_percent", eventTime, &pool.ChlorinatorPct)
	influx_post_one_metric(bp, key, tags, "heater_on", eventTime, &pool.HeaterOn)
	influx_post_one_metric(bp, key, tags, "pool_mode", eventTime, &pool.OperatingMode)
	
	// Write out the button states, too.
	for ii, label := range pool.Buttons {
        if label == "" {
            // Don't write metrics for empty buttons
            continue
        }
        influx_post_one_metric(bp, key, tags, "button_" + label, eventTime, &pool.ButtonValues[ii])
        tags["name"] = label
        influx_post_one_metric(bp, key, tags, "button_state", eventTime, &pool.ButtonValues[ii])
        delete(tags, "name")
	}

	err = c.Write(bp)
	if err != nil {
		log.Fatal(err)
	}

}

func deliver_stats_to_influxdb(c client.Client, config Config) {

	influx_push_metrics(c, config)
}
