package main

import (
	"fmt"
	"github.com/influxdata/influxdb1-client/v2"
	"io/ioutil"
	"golang.org/x/net/html"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	ASSUME_GONE    = -1 * time.Minute
	ENDPOINT_PAUSE = time.Second * 2
	HTTP_TIMEOUT   = time.Second * 30
	DATA_UPDATE    = time.Minute * 1
	NOT_RECORDED   = -1
	version        = "0.1"
)

var pool PoolData

/*
	The heater is now wired to the controller 11/22/2019 after 8+ years.

*/

func init() {
    var err error
    
    config = ReadConfig()
    for trycount := 0;; trycount++ {
        pool.Buttons, err = get_button_info("http://" + config.PoolHost + "/")
        if err == nil {
            break
        } else if  trycount > 10 {
            log.Fatalln("Error fetching button info.  Can't proceed.\n")
        }
    }
    fmt.Printf("buttons found:\n")
    for idx,key := range pool.Buttons {
        fmt.Printf("key %d: '%s'\n", idx, key)
    }
    pool.ButtonValues = make([]Measurement, len(pool.Buttons))
}

// Helper function to pull the href attribute from a Token
func getTdId(t html.Token) (ok bool, id string) {
	// Iterate over token attributes until we find an "href"
	for _, a := range t.Attr {
		if a.Key == "id" {
			id = a.Val
			ok = true
		}
	}
	return
}

func get_button_info(url string) (buttons []string, err error) {

    var resp *http.Response
	var req *http.Request
	var http_err error

	client := &http.Client{Timeout: HTTP_TIMEOUT}
	
	req, http_err = http.NewRequest("GET", url, nil)
	resp, http_err = client.Do(req)
	if http_err != nil {
		return buttons, http_err
	}

	defer resp.Body.Close()
	parser := html.NewTokenizer(resp.Body)
	buttonMap := make(map[int]string)
	done := false
	inKeyTD := false
	keyNum := -1
	
	for ! done {
		token := parser.Next()

		switch {
		case token == html.ErrorToken:
			// End of the document, we're done
			done = true
		case token == html.StartTagToken:
			tag := parser.Token()
		    inKeyTD = false
		    keyNum = -1

			// Check if the token is an <td> tag
			if tag.Data != "td" {
				continue
			}

			// Extract the href value, if there is one
			ok, tdID := getTdId(tag)
			if !ok {
			    // Not every TD will be for our table.
			    // fmt.Printf("can't find id for td!\n")
				continue
			}

            // Make sure the td ID is of the form "key_##"
            if strings.Index(tdID, "Key_") == 0 {
                keyNum, _ = strconv.Atoi(tdID[4:len(tdID)])
                inKeyTD = true
            }
        case token == html.TextToken:
            if inKeyTD && keyNum >= 0 {
                element := parser.Token()
                keyName := strings.TrimSpace(element.Data) 
                buttonMap[keyNum] = keyName
            }
		default:
            inKeyTD = false
            keyNum = -1
		}
	}
	
    for key:=0 ; key < len(buttonMap); key++ {
        keyName, found := buttonMap[key] 
        if !found {
            keyName = ""
        }
        buttons = append(buttons, keyName)
    }
    
	return

}

func get_lcd_payload(url string) (payload string, err error) {

	var resp *http.Response
	var req *http.Request
	var http_err error
	var data []byte

	client := &http.Client{Timeout: HTTP_TIMEOUT}
	req, http_err = http.NewRequest("GET", url, nil)

	if http_err != nil {
		return "", http_err
	}

	resp, http_err = client.Do(req)

	if http_err != nil {
		return "", http_err
	}

	defer resp.Body.Close()
	data, http_err = ioutil.ReadAll(resp.Body)

	payload = string(data)
	return
}

func watch_http_endpoint(config Config) {

	// Treat the "LCD display" like a serial endpoint over which we have no control on
	// the sending side.  Keep polling to see what it currently has to say and update
	// our tracking to match.  Availability will come and go.

	for {

		payload, err := get_lcd_payload("http://" + config.PoolHost + "/WNewSt.htm")

		if err != nil {

			fmt.Printf("Error fetching data from HTTP endpoint: %v\n", err)

		} else {

			// Send it over to get parsed

			parse_and_update(payload)
		}

		time.Sleep(ENDPOINT_PAUSE)
	}
}

func update_datastore(c client.Client, config Config) {

	// Every DATA_UPDATE interval:
	// 1. Check if our data is stale, zero it out if so
	// 2. Write what we have to datastore

	for {

		time.Sleep(DATA_UPDATE) // don't deliver first thing before we have data

		if pool.AirTempF.Last.Before(time.Now().Add(ASSUME_GONE)) {
			pool.AirTempF.Reading = NOT_RECORDED
		}

		if pool.PoolTempF.Last.Before(time.Now().Add(ASSUME_GONE)) {
			pool.PoolTempF.Reading = NOT_RECORDED
		}

		if pool.FilterSpeedRPM.Last.Before(time.Now().Add(ASSUME_GONE)) {
			pool.FilterSpeedRPM.Reading = NOT_RECORDED
		}

		if pool.SaltPPM.Last.Before(time.Now().Add(ASSUME_GONE)) {
			pool.SaltPPM.Reading = NOT_RECORDED
		}

		if pool.FilterOn.Last.Before(time.Now().Add(ASSUME_GONE)) {
			pool.FilterOn.Reading = NOT_RECORDED
		}

		if pool.CleanerOn.Last.Before(time.Now().Add(ASSUME_GONE)) {
			pool.CleanerOn.Reading = NOT_RECORDED
		}

		if pool.LightOn.Last.Before(time.Now().Add(ASSUME_GONE)) {
			pool.LightOn.Reading = NOT_RECORDED
		}

		if pool.ChlorinatorPct.Last.Before(time.Now().Add(ASSUME_GONE)) {
			pool.ChlorinatorPct.Reading = NOT_RECORDED
		}

		if pool.HeaterOn.Last.Before(time.Now().Add(ASSUME_GONE)) {
			pool.HeaterOn.Reading = NOT_RECORDED
		}

		if pool.OperatingMode.Last.Before(time.Now().Add(ASSUME_GONE)) {
			pool.OperatingMode.Reading = NOT_RECORDED
		}
		
		for ii := range pool.ButtonValues {
            if pool.ButtonValues[ii].Last.Before(time.Now().Add(ASSUME_GONE)) {
                pool.ButtonValues[ii].Reading = NOT_RECORDED
            }
		}

		
			fmt.Printf("AirTempF: %d\n", pool.AirTempF.Reading)
			fmt.Printf("PoolTempF: %d\n", pool.PoolTempF.Reading)
			fmt.Printf("FilterSpeedRPM: %d\n", pool.FilterSpeedRPM.Reading)
			fmt.Printf("SaltPPM: %d\n", pool.SaltPPM.Reading)
			fmt.Printf("ChlorinatorPct: %d\n", pool.ChlorinatorPct.Reading)
			fmt.Printf("FilterOn: %d\n", pool.FilterOn.Reading)
			fmt.Printf("CleanerOn: %d\n", pool.CleanerOn.Reading)
			fmt.Printf("LightOn: %d\n", pool.LightOn.Reading)
			fmt.Printf("HeaterOn: %d\n", pool.HeaterOn.Reading)
			fmt.Printf("OperatingMode: %d\n", pool.OperatingMode.Reading)
			for ii := range pool.Buttons {
			    fmt.Printf("%s: %d\n", pool.Buttons[ii], pool.ButtonValues[ii].Reading)
			}

		// Now deliver this data to the influxdb backend
		deliver_stats_to_influxdb(c, config)

	}

}
