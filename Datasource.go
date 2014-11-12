package Peanut

import (
	"time"
)

type Datasource struct {
	Name   string
	path   string
	output chan<- PowerSample

	static ImpulseCount
}

func NewDatasource(name string, output chan<- PowerSample) *Datasource {
	result := Datasource{name, "", output, 0.0}
	return &result
}

func (self *Datasource) Init(name string) {
	self.Name = name
}

func (self *Datasource) Poll() {
	self.static++
	self.output <- PowerSample{time.Now(), self.static}
}
