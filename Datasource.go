package peanut

import (
	"bufio"
	"fmt"
	"github.com/dustin/go-rs232"
	"log"
	"strconv"
	"strings"
	"time"
)

type SinglePoller interface {
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

func NewDummyDatasource(name string, output chan<- ImpulseSample) *DummyDatasource {
	result := DummyDatasource{name, "", output, 0.0}
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
	lastValue0 ImpulseCount
	lastValue1 ImpulseCount
}

func NewSMSDatasource(p string, o0 chan<- ImpulseSample, o1 chan<- ImpulseSample) *SMSDatasource {
	return &SMSDatasource{p, o0, o1, -1, -1}
}

func (d *SMSDatasource) StartPolling() {
	go d.loop()
}

func (d *SMSDatasource) loop() {
	// Mac:	portString := "/dev/tty.usbserial-A702PEEC"
	log.Printf("Opening '%s'", d.portString)
	port, err := rs232.OpenPort(d.portString, 115200, rs232.S_8N1)
	if err != nil {
		log.Fatalf("Error opening port: %s", err)
	}
	defer port.Close()

	println("Port open")

	r := bufio.NewReader(port)
	for {
		_, err := port.Write([]byte("$?\n"))
		if err != nil {
			log.Fatalf("Error writing port: %s", err)
		}

		line, _, err := r.ReadLine()
		if err != nil {
			log.Fatalf("Error reading:  %s", err)
		}
		parts := strings.Split(string(line), ";")
		if len(parts) < 4 {
			println("Error with serial communication. If this happens often, re-attach USB device")
			continue
		}
		t := time.Now()
		i0, err := strconv.ParseInt((parts[1]), 10, 64)
		if err == nil {
			v0 := ImpulseCount(i0+1) + d.lastValue0
			if v0 != d.lastValue0 {
				d.output0 <- ImpulseSample{t, ImpulseCount(v0)}
				d.lastValue0 = v0
			}
		}
		i1, err := strconv.ParseInt((parts[2]), 10, 64)
		if err == nil {
			v1 := ImpulseCount(i1)
			if v1 != d.lastValue1 {
				d.output1 <- ImpulseSample{t, ImpulseCount(v1)}
				d.lastValue1 = v1
			}
		}
		fmt.Printf("%v A = %v; B = %v\n", t, i0, i1)
		time.Sleep(time.Millisecond * 10)
	}
}
