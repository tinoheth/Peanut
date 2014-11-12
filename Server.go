package Peanut

import (
	"log"
	"net/http"
	"time"
)

func defaultTimeFactor() time.Duration {
	// Use time.Duration to get rid of casting
	return 10000000
	// One second equals 100 time units, so we can measure nearly 11 minutes (6W)
}

func Poll(sources []*Datasource) {
	for {
		//println(time.Now().String())
		time.Sleep(1 * time.Second)
		for _, current := range sources {
			current.Poll()
		}
	}
}

type Server struct {
	Providers map[string]*DataProvider
}

func (server *Server) Serve() {
	for key, provider := range server.Providers {
		http.HandleFunc("/"+key+"/KWh", provider.handlerKWh)
		http.HandleFunc("/"+key+"/Watt", provider.handlerWatt)
	}
	err := http.ListenAndServe(":8088", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func (self *Server) Init() {
	self.Providers = make(map[string]*DataProvider)
}

type ImpulseCount int64

type KWh float64

type PowerSample struct {
	time.Time
	Impulses ImpulseCount
}
