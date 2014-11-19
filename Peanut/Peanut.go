// Peanut
package main

import (
	"flag"
	"fmt"
	. "github.com/tinoheth/Peanut"
	"image/png"
	"net/http"
	"os"
	"time"
)

func main() {
	//println("Running server")

	var server Server
	server.Init()

	sources := make(SinglePollerArray, 1)

	input := make(chan ImpulseSample)
	testDatasource := NewDummyDatasource("Datei", input)
	sources[0] = testDatasource

	flag.Parse()
	basepath := flag.Arg(0)
	println("Basepath is " + basepath)

	provider := NewDataProvider("Dummy", basepath, input)
	server.Providers["Dummy"] = provider

	chan0 := make(chan ImpulseSample)
	chan1 := make(chan ImpulseSample)

	tty := "/dev/ttyUSB0"
	if !FileExists(tty) {
		// Maybe we are on a mac?
		tty = "/dev/tty.usbserial-A702PEEC"
	}
	sms := NewSMSDatasource(tty, chan0, chan1)
	sms.StartPolling()

	solar := NewDataProvider("Solar", basepath, chan0)
	server.Providers["Solar"] = solar
	//consume := NewDataProvider("Verbrauch", basepath, chan1)
	//server.Providers["Verbrauch"] = consume

	http.HandleFunc("/png", handlePNG)

	//readTest(provider)
	//go Poll(sources)
	server.Serve()
}

func handlePNG(w http.ResponseWriter, r *http.Request) {
	source, _ := os.OpenFile("/Users/tinoheth/Pictures/User Icon.png", os.O_RDONLY, os.FileMode(0755))
	img, _ := png.Decode(source)
	png.Encode(w, img)
}

func readTest(provider *DataProvider) {
	impulses := provider.ReadInCache(time.Now().AddDate(0, 0, -1), time.Hour*48)
	fmt.Printf("Count = %d\n", len(impulses))
	values := Derive(impulses, provider.ImpulseTranslationFactor)
	for _, c := range values {
		fmt.Printf("%s: %v\n", c.Time.Format(time.ANSIC), c.Value)
	}
}

/*
	Aquiring data
	1. We have an infinite loop that polls for new data (Datasource gets triggered)
	2. Data (value, timestamp) is pushed out of Datasource (chan?)
	3. Delta of data is logged to simple file

	Raw file format
	Each word is one impulse, content depends on time (1 = 1/32s?). Zero means that we've got more than one event in
	that timeframe.
	Filename equals order of file in week (timestamp of creation?) plus time factor.
	Each file should contain exactly 1000 entries - thus allowing good guesses depending on count:
	One file is one KWh, one word is one Wh.

	Folders and cache files
	Raw files aren't effective - they just save memory.
	In addition, CVS files with timestamp and impulse count are used as "catalog".
	A new folder is created when a week begins. This folder contains all raw files.

	Solar/Year2014/Week01.csv
	Solar/Year2014/Week01/4234343334@1:32.raw

	Assuming a power consumption of 500W, we create 12 files per day, 84 per week.
	This should be sufficient for a month-overview.
	For a full day (24h), 720 samples should be enough (sample every two minutes -> keep one of 16 samples in average)

	Load state once at startup, create Samples for current (and last) hour, day, week, month, year and total
	Listen to sources, calculate new values (power consumed, watts) for each timestamp
	Put all Samples into arrays (separate arrays for each Datasource)

	Server (priorities)
	Current values for all Datasources
	Generate datasets on the fly (and cache them)
	Create graphs from datasets
*/
