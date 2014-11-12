// Peanut
package main

import (
	//"fmt"
	"flag"
	. "github.com/tinoheth/Peanut"
)

func main() {
	println("Running server")

	var server Server
	server.Init()

	sources := make([]*Datasource, 1)

	input := make(chan PowerSample)
	testDatasource := NewDatasource("Datei", input)
	testDatasource.Init("Port")
	sources[0] = testDatasource

	flag.Parse()
	basepath := flag.Arg(0)
	println("Basepath is " + basepath)

	provider := NewDataProvider("Solar", basepath, input)
	server.Providers["Solar"] = provider

	go Poll(sources)
	server.Serve()
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
