package main

import (
	"net/http"
	"fmt"
	"log"
	"time"
)

func main() {
	println("Running server")
	d := NewDatasource("Datei")
	d.Init("Port")
	println(d.Name)
	s := PowerSample{time.Now(), 0, 0}
	println(s.Time.String())

	go poll()

	serve()
}

func serve() {
	http.HandleFunc("/KWh", handlerKWh)
	http.HandleFunc("/Watt", handlerWatt)
	err := http.ListenAndServe(":8088", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func poll() {
	for {
		println(time.Now().String())
		time.Sleep(3 * time.Second)
	}
}

func handlerKWh(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "%f", 3.1415926)
}

func handlerWatt(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "%f", 2.79)
}

type PowerSample struct {
	time.Time
	Watts		float64
	KWh			float64
}


/*
	Aquiring data
	1. We have an infinite loop that polls for new data (Datasource gets triggered)
	2. Data (value, timestamp) is pushed out of Datasource (chan?)
	3. Delta of data is logged to simple file

	File formate
	Each byte is one impulse, content depends on time (1 = 1/8s?). Byte 255 means no impulse at all,
	zero means that we've got more than one event in that timeframe

	Load state once at startup, create Samples for current (and last) hour, day, week, month, year and total
	Listen to sources, calculate new values (power consumed, watts) for each timestamp
	Put all Samples into arrays (separate arrays for each Datasource)

	Server (priorities)
	Current values for all Datasources
	Generate datasets on the fly (and cache them)
	Create graphs from datasets
 */

type DataProvider struct {
	Name		string
	samples		[]PowerSample
}

func (self *DataProvider) currentValue() float64 {
	return 0.0
}

func (self *DataProvider) currentTendency() float64 {
	return 0.0
}

func (self *DataProvider) registerSample(sample PowerSample) {
	self.samples.append(sample)
}

func (self *DataProvider) valuesInTimeframe(start, end time.Timer) []PowerSample {
	return self.samples
}

