// DataProvider+Disk
package peanut

import (
	"encoding/binary"
	"encoding/csv"
	//"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func (dp *DataProvider) InitFromDisk() *DataProvider {
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
		} else if len(lines) > 0 {
			ll := lines[len(lines)-1]
			n, _ := strconv.Atoi(ll[1])
			dp.impulseCount = ImpulseCount(n)
			rawpath := filepath.Join(strings.TrimRight(cp, ".csv"), ll[0])
			stat, err := os.Stat(rawpath)
			if err == nil {
				dp.impulseCount += ImpulseCount(stat.Size() / 2)
			}
		}
	}

	return dp
}

func (dp *DataProvider) checkTime(t time.Time) uint16 {
	var result uint16 = 0
	if t.After(dp.checkpoint) {
		println("After check")
		dp.impulseCount++ // we create a new file with a timestamp
		// create a new catalog first
		year, week := t.ISOWeek()
		dp.weekpath = filepath.Join(dp.providerPath, strconv.Itoa(year), strconv.Itoa(week))
		println("Writing to " + dp.weekpath)
		err := os.MkdirAll(dp.weekpath, os.ModeDir|os.FileMode(0755))
		if err != nil {
			log.Fatal(err.Error())
		}
		dp.setupEndpoint(t, dp.impulseCount)
		dp.checkpoint = NextWeekBreak(dp.catalogTime)
		//dp.checkpoint = dp.checkpoint.AddDate(0, 0, 7)
	} else {
		println("Before check")
		// check if delta t fits into int16
		deltaT := t.Sub(dp.catalogTime) - dp.deltaSum*dp.timeFactor
		delta := DivideDuration(deltaT, dp.timeFactor)
		if delta > math.MaxUint16 {
			result = 0
			dp.impulseCount++
			dp.setupEndpoint(t, dp.impulseCount)
		} else {
			dp.deltaSum += delta
		}
	}
	println("Did timecheck")
	return result
}

func consume(diskCache <-chan uint16, path string) {
	println("Consume to " + path)
	raw, err := os.OpenFile(path, os.O_RDWR|os.O_APPEND|os.O_CREATE, os.FileMode(0755))
	if err != nil {
		log.Fatal(err)
	}
	defer raw.Close()
	for value := range diskCache {
		println("Consumed bytes")
		binary.Write(raw, binary.LittleEndian, value)
	}
}

func (dp *DataProvider) setupEndpoint(start time.Time, impulses ImpulseCount) {
	old := dp.disk
	c := make(chan uint16)
	dp.disk = c
	if old != nil {
		close(old)
	}
	t, error := start.MarshalText()
	tstring := string(t)
	line := []string{tstring, strconv.Itoa(int(impulses))}

	catalog, error := os.OpenFile(dp.weekpath+".csv", os.O_RDWR|os.O_APPEND|os.O_CREATE, os.FileMode(0755))
	if error != nil {
		log.Fatal(error.Error())
	}
	defer catalog.Close()
	writer := csv.NewWriter(catalog)
	defer writer.Flush()
	error = writer.Write(line)
	if error != nil {
		log.Fatal(error.Error())
	}
	dp.chanSize = 0
	dp.catalogTime = start
	go consume(c, filepath.Join(dp.weekpath, tstring))
	println("New endpoint")
}

/*
func (dp *DataProvider) writeOutCache(timeFactor time.Duration) {
	dp.writeMutex.Lock()
	defer dp.writeMutex.Unlock()
	i := (dp.lastWrittenIndex + 1) % BufferSize
	start := dp.ringbuffer[i]
	year, week := start.ISOWeek()
	weekpath := filepath.Join(dp.providerPath, strconv.Itoa(year), strconv.Itoa(week))
	//println("Writing to " + weekpath)
	error := os.MkdirAll(weekpath, os.ModeDir|os.FileMode(0755))

	//		catalog, error := os.Create(weekpath + ".csv")
	catalog, error := os.OpenFile(weekpath+".csv", os.O_RDWR|os.O_APPEND|os.O_CREATE, os.FileMode(0755))
	defer catalog.Close()
	if error != nil {
		log.Fatal(error.Error())
	} else {
		writer := csv.NewWriter(catalog)
		defer writer.Flush()
		t, error := start.MarshalText()
		tstring := string(t)
		line := []string{tstring, strconv.Itoa(int(dp.lastValue) - int(dp.uncachedSamples))}
		error = writer.Write(line)
		if error != nil {
			println(error.Error())
		}
		//println("writing to " + tstring)
		raw, err := os.OpenFile(filepath.Join(weekpath, tstring), os.O_RDWR|os.O_APPEND|os.O_CREATE, os.FileMode(0755))
		if err != nil {
			log.Fatal(err)
		}
		defer raw.Close()
		var deltaSum time.Duration = 0
		for dp.uncachedSamples > 0 {
			dp.lastWrittenIndex = (dp.lastWrittenIndex + 1) % BufferSize
			deltaBig := dp.ringbuffer[dp.lastWrittenIndex].Sub(start) - deltaSum*timeFactor
			delta := DivideDuration(deltaBig, timeFactor)
			deltaSum += delta
			//println(delta)
			binary.Write(raw, binary.LittleEndian, uint16(delta))
			dp.uncachedSamples--
		}
	}
}
*/
func (dp *DataProvider) ReadInCache(start time.Time, duration time.Duration) ImpulseSampleData {
	return dp.ReadData(start, duration, defaultTimeFactor())
}
func (dp *DataProvider) ReadData(start time.Time, duration time.Duration, timeFactor time.Duration) []ImpulseSample {
	result := make([]ImpulseSample, 0, DefaultImpulseCount)
	end := start.Add(duration)
	current := start
	for current.Before(end) {
		year, week := current.ISOWeek()
		weekpath := filepath.Join(dp.providerPath, strconv.Itoa(year), strconv.Itoa(week))

		catalog, err := os.Open(weekpath + ".csv")
		if err == nil {
			defer catalog.Close()
			reader := csv.NewReader(catalog)
			for {
				row, err := reader.Read()
				if err != nil {
					break
				} else {
					stamp := row[0]
					t, errT := time.Parse(time.RFC3339, stamp)
					if t.After(end) {
						break
					}
					count, errC := strconv.Atoi(row[1])
					if errT == nil && errC == nil {
						rawPath := filepath.Join(weekpath, stamp)
						//println(rawPath)
						result = append(result, readRawData(rawPath, t, count, timeFactor)...)
					}
				}
			}
		}
		current = current.AddDate(0, 0, 7)
	}
	//println("Done")
	return result
}

func readRawData(path string, offsetTime time.Time, offsetValue int, timeFactor time.Duration) []ImpulseSample {
	result := make([]ImpulseSample, 0, DefaultImpulseCount)
	raw, err := os.Open(path)
	if err == nil {

		defer raw.Close()
		info, err := raw.Stat()
		if err == nil {
			var buffer = make([]uint16, info.Size()/2)
			binary.Read(raw, binary.LittleEndian, buffer)
			//fmt.Printf("Length = %d", len(buffer))
			var sum time.Duration = 0
			for c := range buffer {
				sum += time.Duration(buffer[c])
				//println(buffer[c])
				t := offsetTime.Add(sum * timeFactor)
				offsetValue++
				result = append(result, ImpulseSample{t, ImpulseCount(offsetValue)})
			}
		}
	}
	return result
}
