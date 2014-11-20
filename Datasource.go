package peanut

import (
	"bufio"
	"github.com/dustin/go-rs232"
	"io"
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
	self.output <- ImpulseSample{time.Now(), self.static}
	self.static++
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

func poll(destSource <-chan io.Writer) {
	p := <-destSource
	for p != nil {
		n, err := p.Write([]byte("$?\n"))
		if err != nil {
			log.Printf("Wrote %v\n", n)
			println(err.Error())
			time.Sleep(time.Second * 1)
		} else {
			//log.Printf("Wrote %v", n)
			time.Sleep(time.Millisecond * 50)
		}
		select {
		case p = <-destSource:
			println("New port for polling")
		default:
		}
	}
}

func (d *SMSDatasource) readLoop(rw *bufio.Reader) {
	for {
		//println("will read port")
		//time.Sleep(time.Millisecond * 50)
		line, err := rw.ReadString('\n')
		//println("did read port")
		if err != nil {
			log.Printf("Error reading:  %s", err)
			return
		}
		parts := strings.Split(line, ";")
		if len(parts) < 4 {
			log.Printf("Error with serial communication. If this happens often, re-attach USB device (%s)", line)
			continue
		}
		t := time.Now()
		i0, err := strconv.ParseInt((parts[1]), 10, 64)
		if err == nil {
			v0 := ImpulseCount(i0+1) + d.lastValue0
			//fmt.Printf("current = %v; last = %v\n", v0, d.lastValue0)

			if v0 != d.lastValue0 {
				//println("will push sample 0")
				select {
				case d.output0 <- ImpulseSample{t, ImpulseCount(v0)}:
					d.lastValue1 = v0
					break
				default:
					println("Couldn't push for port 0")
				}
				//println("did push sample 0")
				d.lastValue0 = v0
			}
		}
		i1, err := strconv.ParseInt((parts[2]), 10, 64)
		if err == nil {
			v1 := ImpulseCount(i1)
			if v1 != d.lastValue1 {
				//println("will push sample 0")
				//go send(d.output1, ImpulseSample{t, ImpulseCount(v1)})
				select {
				case d.output1 <- ImpulseSample{t, ImpulseCount(v1)}:
					d.lastValue1 = v1
					break
				default:
					println("Couldn't push for port 1")
				}
			}
		}
		//fmt.Printf("%v A = %v; B = %v\n", t, i0, i1)
		//time.Sleep(time.Millisecond * 10)
	}
}

func (d *SMSDatasource) loop() {
	channel := make(chan io.Writer)
	go poll(channel)
	for {
		log.Printf("Opening '%s'", d.portString)
		port, err := rs232.OpenPort(d.portString, 115200, rs232.S_8N1)
		if err != nil {
			log.Printf("Error opening port: %s", err)
		} else {
			defer port.Close()
			println("Serial port open")
			channel <- port
			d.readLoop(bufio.NewReader(port))
		}
		time.Sleep(time.Second * 5) // sleep - may be we reconnect adapter...
	}
}
