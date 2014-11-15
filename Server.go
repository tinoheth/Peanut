package Peanut

import (
	"image/png"
	"log"
	"net/http"
	"time"
)

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
		png := PNGProvider{*provider}
		http.HandleFunc("/"+key+"/png", png.handleRequest)
	}
	err := http.ListenAndServe(":8088", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func (self *Server) Init() {
	self.Providers = make(map[string]*DataProvider)
}

type PNGProvider struct {
	DataProvider
}

func (p *PNGProvider) handleRequest(w http.ResponseWriter, r *http.Request) {
	impulses := p.DataProvider.ReadInCache(time.Now().AddDate(0, 0, -1), time.Hour*24)
	values := Derive(impulses, p.ImpulseTranslationFactor)
	img := powerPlot(values)
	png.Encode(w, img)
}
