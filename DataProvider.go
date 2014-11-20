// Keeps track of samples. Caches data, and writes to disk

package peanut

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const DefaultImpulseCount = 10
const BufferSize = 8 * DefaultImpulseCount

type DataStore interface {
	write(path string, name string, data []ImpulseSample)
	read(path string, name string) []ImpulseSample
}

type DataProvider struct {
	Name                     string
	ImpulseTranslationFactor Float
	input                    <-chan ImpulseSample
	providerPath             string
	timeFactor               time.Duration

	impulseCount ImpulseCount

	weekpath    string
	checkpoint  time.Time // weekpath valid until that time
	chanSize    ImpulseCount
	catalogTime time.Time
	disk        chan<- uint16
	deltaSum    time.Duration
}

func (self *DataProvider) currentValue() Float {
	return Float(self.impulseCount) / self.ImpulseTranslationFactor
}

func NewDataProvider(name string, basepath string, input <-chan ImpulseSample) *DataProvider {
	result := new(DataProvider)
	result.Name = name
	result.input = input
	result.providerPath = filepath.Join(basepath, name)
	println("Creating directory " + result.providerPath)
	error := os.MkdirAll(result.providerPath, os.ModeDir|os.FileMode(0755))
	if error != nil {
		log.Fatal(error.Error())
	}
	result.ImpulseTranslationFactor = DefaultImpulseCount
	result.timeFactor = defaultTimeFactor()
	result.chanSize = math.MaxInt64 // force a new raw file
	return result.InitFromDisk()
}

func (self *DataProvider) currentTendency() (result float64) {
	/*
		if self.samples.Len() > 1 {
			elem := self.samples.Back()
			var last ImpulseSample = elem.Value.(ImpulseSample)
			var before = elem.Prev().Value.(ImpulseSample)
			result = float64(last.Impulses-before.Impulses) / (last.Time.Sub(before.Time).Seconds())
		} else {
	*/
	result = math.NaN()
	//}
	return
}

func (self *DataProvider) handlerKWh(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "%f", self.currentValue())
}

func (self *DataProvider) handlerWatt(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "%f", self.currentTendency())
}

func (dp *DataProvider) startListening() {
	go dp.listen()
}

func (self *DataProvider) listen() {
	for sample := range self.input {
		var value uint16
		//println("listen")
		if self.impulseCount > sample.Impulses {
			self.impulseCount = sample.Impulses - 1 // should never happen in reality - exept on fresh systems
		} else if self.impulseCount < sample.Impulses {
			// Check if we need a new file because time difference is to big to hold
			// println("Sample has fresh impulses")
			value = self.checkTime(sample.Time)
		}
		//log.Printf("Time: %v; chanSize: %v; impulseCount: %v; sampleI: %v\n", sample.Time, self.chanSize, self.impulseCount, sample.Impulses)

		for self.impulseCount < sample.Impulses {
			//println("consumed impulse")
			self.impulseCount++
			// Decide if we need a new file because the current file is "full"
			if self.chanSize < DefaultImpulseCount {
				//println("will push bytes")
				self.disk <- value
				//println("did push bytes")
				self.chanSize++
			} else {
				println("File is full")
				self.setupEndpoint(sample.Time, self.impulseCount)
			}
		}
		//println("listened")
	}
	println("Stop listenening")
}
