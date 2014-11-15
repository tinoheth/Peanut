package Peanut

import (
	"bufio"
	"github.com/dustin/go-rs232"
	"time"
)

type SiglePoller interface {
	Poll()
}

type LoopPoller interface {
	StartPolling()
}

type DummyDatasource struct {
	Name   string
	path   string
	output chan<- ImpulseSample

	static ImpulseCount
}

func NewDatasource(name string, output chan<- ImpulseSample) *DummyDatasource {
	result := Datasource{name, "", output, 0.0}
	return &result
}

func (self *DummyDatasource) Poll() {
	self.static++
	self.output <- ImpulseSample{time.Now(), self.static}
}

type SMSDatasource struct {
	portString string
	output0    chan<- ImpulseSample
	output1    chan<- ImpulseSample
}

func (d *SMSDatasource) StartPolling() {
	go d.loop()
}

func (d *SMSDatasource) loop() {
	// Mac:	portString := "/dev/tty.usbserial-A702PEEC"
	log.Printf("Opening '%s'", portString)
	port, err := rs232.OpenPort(portString, 115200, rs232.S_8N1)
	if err != nil {
		log.Fatalf("Error opening port: %s", err)
	}
	defer port.Close()

	println("Port open")

	r := bufio.NewReader(port)
	for {
		n, err := port.Write([]byte("$?\n"))
		if err != nil {
			log.Fatalf("Error writing port: %s", err)
		} else {
			println(n)
		}

		line, _, err := r.ReadLine()
		if err != nil {
			log.Fatalf("Error reading:  %s", err)
		}
		log.Printf("<: %s", line)
	}
}
