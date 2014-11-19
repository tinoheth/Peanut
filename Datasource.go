package peanut

import (
	"bufio"
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

func (d *SMSDatasource) readLoop(rw *bufio.ReadWriter, port *rs232.SerialPort) {
	buf := make([]byte, 64)
	for {
		for {
			//println("will write port")
			n, err := port.Write([]byte("$?\n"))
			//println("did write port")
			rw.Flush()
			if err != nil {
				log.Printf("Error writing port: %s", err)
				continue
				//		} else {
				//println(r.)
			} else if n != 3 {
				println(n)
				log.Fatal(n)
			}
			time.Sleep(time.Millisecond * 500)
			if rw.Available() > 0 {
				break
			}
		}
		println("will read port")
		//line, err := rw.ReadString('\n')
		n, err := port.Read(buf)
		println("did read port")
		if err != nil {
			log.Printf("Error reading:  %s", err)
			continue
		}
		println(n)
		line := string(buf)
		parts := strings.Split(line, ";")
		if len(parts) < 4 {
			println("Error with serial communication. If this happens often, re-attach USB device " + line)
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
				case d.output1 <- ImpulseSample{t, ImpulseCount(v0)}:
					d.lastValue1 = v0
				default:
				}
				//println("did push sample 0")
				d.lastValue0 = v0
			}
		}
		i1, err := strconv.ParseInt((parts[2]), 10, 64)
		if err == nil {
			v1 := ImpulseCount(i1)
			if v1 != d.lastValue1 {
				println("will push sample 0")

				//go send(d.output1, ImpulseSample{t, ImpulseCount(v1)})
				select {
				case d.output1 <- ImpulseSample{t, ImpulseCount(v1)}:
					d.lastValue1 = v1
				default:
				}
				println("did push sample 0")
			}
		}
		//fmt.Printf("%v A = %v; B = %v\n", t, i0, i1)
		//time.Sleep(time.Millisecond * 10)
	}
}

func (d *SMSDatasource) loop() {
	for {
		log.Printf("Opening '%s'", d.portString)
		port, err := rs232.OpenPort(d.portString, 115200, rs232.S_8N1)
		if err != nil {
			log.Printf("Error opening port: %s", err)
		} else {
			defer port.Close()

			println("Serial port open")

			r := bufio.NewReader(port)
			w := bufio.NewWriter(port)
			rw := bufio.NewReadWriter(r, w)
			d.readLoop(rw, port)
		}
		time.Sleep(time.Second * 5) // sleep - may be we reconnect adapter...
	}
}

func send(c chan<- ImpulseSample, v ImpulseSample) {
	c <- v
}
