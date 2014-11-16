// Keeps track of samples. Caches data, and writes to disk

package peanut

import (
	"encoding/csv"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
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
	MaxSamples               uint
	MinSamples               uint
	ringbuffer               [BufferSize]time.Time
	writeBuffer              [DefaultImpulseCount]uint16
	currentIndex             uint
	lastWrittenIndex         uint
	lastValue                ImpulseCount
	uncachedSamples          uint
	writeMutex               sync.Mutex
}

func (self *DataProvider) currentValue() Float {
	return Float(self.lastValue) / self.ImpulseTranslationFactor
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
	result.MaxSamples = 10 //result.impulsesPerKWh * 20
	result.MinSamples = 2  //result.MaxSamples / 2
	return result.InitFromDisk()
}

func (dp *DataProvider) InitFromDisk() *DataProvider {
	dp.lastWrittenIndex = BufferSize - 1

	defer dp.startListening()

	l, err := filepath.Glob(filepath.Join(dp.providerPath, "*/*.csv"))
	if err != nil {
		log.Fatal(err.Error())
	} else if len(l) == 0 {
		return dp
	}
	cp := l[len(l)-1]
	catalog, error := os.OpenFile(cp, os.O_RDONLY, os.FileMode(0755))
	defer catalog.Close()
	if error != nil {
		println(error.Error())
	} else {
		reader := csv.NewReader(catalog)
		lines, error := reader.ReadAll()
		if error != nil {
			println(error.Error())
		} else {
			ll := lines[len(lines)-1]
			n, _ := strconv.Atoi(ll[1])
			dp.lastValue = ImpulseCount(n)
			rawpath := filepath.Join(strings.TrimRight(cp, ".csv"), ll[0])
			stat, err := os.Stat(rawpath)
			if err != nil {
				dp.lastValue += ImpulseCount(stat.Size() / 2)
			}
		}
	}
	log.Printf("Init Uncached: %d; lastWritten: %d; currentIndex: %d; value: %d\n", dp.uncachedSamples, dp.lastWrittenIndex, dp.currentIndex, dp.lastValue)

	return dp
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

func (self *DataProvider) pushSample(sample ImpulseSample) {
	//self.samples.PushBack(sample)
	if self.lastValue > sample.Impulses {
		self.lastValue = sample.Impulses - 1 // should never happen in reality
	}
	log.Printf("Pushing Uncached: %d; lastWritten: %d; currentIndex: %d; sampleImpulses: %d; value: %d\n", self.uncachedSamples, self.lastWrittenIndex, self.currentIndex, sample.Impulses, self.lastValue)
	//log.Printf("Time: %v\n", sample.Time)
	for self.lastValue < sample.Impulses {
		self.ringbuffer[self.currentIndex] = sample.Time
		self.currentIndex = (self.currentIndex + 1) % BufferSize
		self.lastValue++
	}
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
	working := true
	for working {
		var sample ImpulseSample
		sample, working = <-self.input
		self.pushSample(sample)
		self.uncachedSamples++
		go self.handleCache(defaultTimeFactor())
	}
	println("Stop listenening")
}

func (self *DataProvider) handleCache(timeFactor time.Duration) {
	/*
		See if samples is to big - in this case, write the begin to disk and remove from memory
	*/
	//log.Printf("Uncached: %d; lastWritten: %d; currentIndex: %d\n", self.uncachedSamples, self.lastWrittenIndex, self.currentIndex)
	if self.uncachedSamples >= DefaultImpulseCount {
		self.writeOutCache(timeFactor)
	}
}
