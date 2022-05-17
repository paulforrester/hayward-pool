package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
    BUTTON_OFF        = 4
    BUTTON_ON         = 5
	MAXRPM            = 3450.0
	MODE_OFF          = 0
	MODE_ON           = 1 
	MODE_POOL         = 2
	MODE_SPA          = 3
	MODE_SPILLOVER    = 4
)

func standardize_whitespace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}


func status_update(label string, status int, datum *Measurement ) {
    if status == BUTTON_ON {
        status = 1
    } else {
        status = 0
    }
    
    report_if_change(datum.Reading, status, label)
    datum.Reading = status
    datum.Last = time.Now()
}
// Figure out what data we're dealing with by matching strings
// This isn't fun but it's all we have to work with

func parse_and_update(payload string) {

	var work_str string

	re := regexp.MustCompile("\r?\n")
	payload = re.ReplaceAllString(payload, "")

	re_cleanup := regexp.MustCompile("(?m)<body>(.*)</body>")

	if len(re_cleanup.FindStringSubmatch(payload)) != 2 {

		fmt.Printf("Can't parse, skipping:\n%s\n", payload)
		return
	}

	work_str = standardize_whitespace(re_cleanup.FindStringSubmatch(payload)[1])

	// fmt.Printf("Working with: %s\n", work_str)

	// For each possible string in the LCD stream, see if we can match and extract its vaule

	re = regexp.MustCompile("^Air Temp (\\d+)")
	if len(re.FindStringSubmatch(work_str)) == 2 {

		pool.AirTempF.Reading, _ = strconv.Atoi(re.FindStringSubmatch(work_str)[1])
		pool.AirTempF.Last = time.Now()
	}

	re = regexp.MustCompile("^Salt Level \\w+ (\\d+) PPM")
	if len(re.FindStringSubmatch(work_str)) == 2 {

		pool.SaltPPM.Reading, _ = strconv.Atoi(re.FindStringSubmatch(work_str)[1])
		pool.SaltPPM.Last = time.Now()
	}

	re = regexp.MustCompile("^Filter Speed \\w+ ([0-9A-Za-z_%]+) ")
	if len(re.FindStringSubmatch(work_str)) == 2 {

		if re.FindStringSubmatch(work_str)[1] == "Off" {

			pool.FilterSpeedRPM.Reading = 0
		} else if re.FindStringSubmatch(work_str)[1] == "RPM" {

			pool.FilterSpeedRPM.Reading, _ = strconv.Atoi(strings.Replace(re.FindStringSubmatch(work_str)[1], "RPM", "", -1))
		} else {
			var speedpct int
			speedpct, _ = strconv.Atoi(strings.Replace(re.FindStringSubmatch(work_str)[1], "%", "", -1))
			pool.FilterSpeedRPM.Reading = int(float64(speedpct)/100 * MAXRPM)
		}

		pool.FilterSpeedRPM.Last = time.Now()
	}

	re = regexp.MustCompile("^Pool Chlorinator \\w+ (\\d+)%")
	if len(re.FindStringSubmatch(work_str)) == 2 {

		pool.ChlorinatorPct.Reading, _ = strconv.Atoi(re.FindStringSubmatch(work_str)[1])
		pool.ChlorinatorPct.Last = time.Now()
	}

	re = regexp.MustCompile("^Pool Temp (\\d+)&")
	if len(re.FindStringSubmatch(work_str)) == 2 {

		pool.PoolTempF.Reading, _ = strconv.Atoi(re.FindStringSubmatch(work_str)[1])
		pool.PoolTempF.Last = time.Now()
	}

    // get the button statuses.
    
    
	re = regexp.MustCompile(".*xxx.*xxx(.{12})xxx")
	statuses := []byte(re.FindStringSubmatch(work_str)[1])
	fmt.Printf("statuses: %s\n", statuses)
	var buttonstats []int
	for _, stat := range statuses {
		buttonstats = append(buttonstats, (int(stat)&0xF0)>>4)
		buttonstats = append(buttonstats, int(stat)&0x0F)
	}
	poolMode := MODE_OFF
	heater := BUTTON_OFF
	for ii, stat := range buttonstats {
		fmt.Printf("buttons %02d: %d\n", ii, stat)
		
		if stat != BUTTON_OFF && stat != BUTTON_ON {
		    // Not a valid status.  Skip it.
		    continue
		}
		
		// Handle for backwards compatibility.
		if strings.Contains(pool.Buttons[ii], "FILTER")  {
            status_update("Filter", stat, &pool.FilterOn )
            if poolMode == MODE_OFF && stat == BUTTON_ON{
                poolMode = MODE_ON
            }
        } else if strings.Contains(pool.Buttons[ii], "LIGHTS") {
            status_update("Lights", stat, &pool.LightOn )
        } else if strings.Contains(pool.Buttons[ii], "CLEANER") {
            status_update("Cleaner", stat, &pool.CleanerOn )
        } else if strings.Contains(pool.Buttons[ii], "HEATER") || pool.Buttons[ii] == "SOLAR VALVE"{
            if stat == BUTTON_ON {
                heater = BUTTON_ON
            }
        } else if pool.Buttons[ii] == "POOL" && stat == BUTTON_ON {
            poolMode = MODE_POOL
        } else if pool.Buttons[ii] == "SPA" && stat == BUTTON_ON {
            poolMode = MODE_SPA
        } else if pool.Buttons[ii] == "SPILLOVER" && stat == BUTTON_ON {
            poolMode = MODE_SPILLOVER
        }
        
        // Save the native button statuses for the button.
        
        status_update(pool.Buttons[ii], stat, &(pool.ButtonValues[ii]))

    }
    
    // Update the state of the heater
    status_update("Heater", heater, &pool.HeaterOn )

    // Set the pool mode to one of 'OFF', 'POOL', 'SPA', or 'FOUNTAIN'
    // This might need to be flagged to execute based on a config setting.
    report_if_change(pool.OperatingMode.Reading, poolMode, "Operating Mode")
    pool.OperatingMode.Reading = poolMode
    pool.OperatingMode.Last = time.Now()
    
}

func report_if_change(old, new int, var_name string) {

	if old != new {
		t := time.Now()

		switch new {
		case MODE_OFF:
			fmt.Printf("%s OFF at %s\n", var_name, t.Format("2006-01-02 15:04:05"))
		case MODE_ON:
			fmt.Printf("%s ON at %s\n", var_name, t.Format("2006-01-02 15:04:05"))
		case MODE_POOL:
			fmt.Printf("%s POOL at %s\n", var_name, t.Format("2006-01-02 15:04:05"))
		case MODE_SPA:
			fmt.Printf("%s POOL at %s\n", var_name, t.Format("2006-01-02 15:04:05"))
		case MODE_SPILLOVER:
			fmt.Printf("%s POOL at %s\n", var_name, t.Format("2006-01-02 15:04:05"))
		}
	}
}
