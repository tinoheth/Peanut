// DataProvider+Disk
package Peanut

import (
	"encoding/binary"
	"encoding/csv"
	//"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

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

func (dp *DataProvider) ReadInCache(start time.Time, duration time.Duration) ImpulseSampleData {
	return dp.ReadData(start, duration, defaultTimeFactor())
}
func (dp *DataProvider) ReadData(start time.Time, duration time.Duration, timeFactor time.Duration) []ImpulseSample {
	result := make([]ImpulseSample, 0)
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
	result := make([]ImpulseSample, 0)
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
