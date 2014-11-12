// Keeps track of samples. Caches data, and writes to disk

package Peanut

import (
	"container/list"
	"encoding/binary"
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
	write(path string, name string, data []PowerSample)
	read(path string, name string) []PowerSample
}

type DataProvider struct {
	Name             string
	impulsesPerKWh   float64
	input            <-chan PowerSample
	providerPath     string
	MaxSamples       uint
	MinSamples       uint
	ringbuffer       [BufferSize]time.Time
	writeBuffer      [DefaultImpulseCount]uint16
	currentIndex     uint
	lastWrittenIndex uint
	lastValue        ImpulseCount
	uncachedSamples  uint
	writeMutex       sync.Mutex
}

func (self *DataProvider) currentValue() float64 {
	return float64(self.lastValue) / self.impulsesPerKWh
}

func NewDataProvider(name string, basepath string, input <-chan PowerSample) *DataProvider {
	result := new(DataProvider)
	result.Name = name
	result.input = input
	result.providerPath = filepath.Join(basepath, name)
	println("Creating directory " + result.providerPath)
	error := os.MkdirAll(result.providerPath, os.ModeDir|os.FileMode(0755))
	if error != nil {
		log.Fatal(error.Error())
	}
	result.impulsesPerKWh = DefaultImpulseCount
	result.MaxSamples = 10 //result.impulsesPerKWh * 20
	result.MinSamples = 2  //result.MaxSamples / 2
	return result.InitFromDisk()
}

func (dp *DataProvider) InitFromDisk() *DataProvider {
	dp.lastWrittenIndex = BufferSize - 1

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

	go dp.listen()
	return dp
}

func (self *DataProvider) currentTendency() (result float64) {
	/*
		if self.samples.Len() > 1 {
			elem := self.samples.Back()
			var last PowerSample = elem.Value.(PowerSample)
			var before = elem.Prev().Value.(PowerSample)
			result = float64(last.Impulses-before.Impulses) / (last.Time.Sub(before.Time).Seconds())
		} else {
	*/
	result = math.NaN()
	//}
	return
}

func (self *DataProvider) pushSample(sample PowerSample) {
	//self.samples.PushBack(sample)
	if self.lastValue > sample.Impulses {
		self.lastValue = sample.Impulses - 1 // should never happen in reality
	}
	log.Printf("Pushing Uncached: %d; lastWritten: %d; currentIndex: %d; sampleImpulses: %d; value: %d\n", self.uncachedSamples, self.lastWrittenIndex, self.currentIndex, sample.Impulses, self.lastValue)
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

func (self *DataProvider) listen() {
	working := true
	for working {
		var sample PowerSample
		sample, working = <-self.input
		println("did listen")
		self.pushSample(sample)
		self.uncachedSamples++
		go self.handleCache(defaultTimeFactor())
	}
}

func (self *DataProvider) handleCache(timeFactor time.Duration) {
	/*
		See if samples is to big - in this case, write the begin to disk and remove from memory
	*/
	log.Printf("Uncached: %d; lastWritten: %d; currentIndex: %d\n", self.uncachedSamples, self.lastWrittenIndex, self.currentIndex)
	if self.uncachedSamples >= DefaultImpulseCount {
		self.writeOutCache(timeFactor)
	}
}

func divide(x time.Duration, y time.Duration) (result time.Duration) {
	result = x / y
	if (x-result*y)*2 >= y {
		result++
	}
	return
}

func (self *DataProvider) writeOutCache(timeFactor time.Duration) {
	self.writeMutex.Lock()
	defer self.writeMutex.Unlock()
	i := (self.lastWrittenIndex + 1) % BufferSize
	start := self.ringbuffer[i]
	year, week := start.ISOWeek()
	weekpath := filepath.Join(self.providerPath, strconv.Itoa(year), strconv.Itoa(week))
	println("Writing to " + weekpath)
	error := os.MkdirAll(weekpath, os.ModeDir|os.FileMode(0755))

	//		catalog, error := os.Create(weekpath + ".csv")
	catalog, error := os.OpenFile(weekpath+".csv", os.O_RDWR|os.O_APPEND|os.O_CREATE, os.FileMode(0755))
	defer catalog.Close()
	if error != nil {
		println(error.Error())
	} else {
		writer := csv.NewWriter(catalog)
		defer writer.Flush()
		t, error := start.MarshalText()
		tstring := string(t)
		line := []string{tstring, strconv.Itoa(int(self.lastValue) - int(self.uncachedSamples))}
		error = writer.Write(line)
		if error != nil {
			println(error.Error())
		}
		println("writing to " + tstring)
		raw, err := os.OpenFile(filepath.Join(weekpath, tstring), os.O_RDWR|os.O_APPEND|os.O_CREATE, os.FileMode(0755))
		if err != nil {
			log.Fatal(err)
		}
		defer raw.Close()
		var deltaSum time.Duration = 0
		for self.uncachedSamples > 0 {
			self.lastWrittenIndex = (self.lastWrittenIndex + 1) % BufferSize
			deltaBig := self.ringbuffer[self.lastWrittenIndex].Sub(start) - deltaSum*timeFactor
			delta := divide(deltaBig, timeFactor)
			deltaSum += delta
			println(delta)
			binary.Write(raw, binary.LittleEndian, uint16(delta))
			self.uncachedSamples--
		}
	}
}

func translateToRaw(elements list.List, start time.Time, timeFactor time.Duration, valueDelta ImpulseCount) []byte {
	result := make([]byte, 0)
	if elements.Len() > 0 {
		startV := elements.Front().Value.(PowerSample).Impulses
		endV := elements.Front().Value.(PowerSample).Impulses
		size := int32(math.Ceil(float64(endV-startV) / float64(valueDelta)))
		result = make([]byte, size)
		var last = start
		i := 0
		for e := elements.Front().Next(); e != nil; e = e.Next() {
			sample := e.Value.(PowerSample)
			currentTime := sample.Time
			delta := currentTime.Sub(last) * timeFactor
			value := sample.Impulses
			for startV < value {
				result[i] = byte(delta)
				i++
				startV += valueDelta
				delta = 0
			}
			startV = value
			last = currentTime
		}
	}
	return result
}

func translateFromRaw(raw []byte, offsetTime time.Time, timeFactor time.Duration, offsetValue ImpulseCount) *list.List {
	result := list.New()
	for _, current := range raw {
		offsetValue += 1
		if current > 0 {
			var delta time.Duration = time.Duration(current) * timeFactor
			offsetTime = offsetTime.Add(delta)
			result.PushBack(PowerSample{offsetTime, offsetValue})
			offsetValue = 0
		}
	}
	return result
}
