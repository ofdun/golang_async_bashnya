package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
)

func ExecutePipeline(jobs ...job) {
	var wg sync.WaitGroup
	in := make(chan interface{})
	out := make(chan interface{})
	for i := range jobs {
		wg.Add(1)
		go func(in, out chan interface{}, worker job, wg *sync.WaitGroup) {
			defer wg.Done()
			defer close(out)
			worker(in, out)
		}(in, out, jobs[i], &wg)
		in, out = out, make(chan interface{})
	}
	wg.Wait()
}

func calcCrc32SingleHash(data string, c chan string, wg *sync.WaitGroup) {
	defer wg.Done()
	defer close(c)
	c <- DataSignerCrc32(data)
}

func asyncCalcOneSingleHash(data interface{}, out chan interface{}, wg *sync.WaitGroup, m *sync.Mutex) {
	defer wg.Done()
	var wgLocal sync.WaitGroup
	wgLocal.Add(2)
	stringifiedData := fmt.Sprintf("%v", data)

	md5OutChan := make(chan string)
	go func(data string, c chan string, m *sync.Mutex) {
		defer close(c)
		m.Lock()
		c <- DataSignerMd5(data)
		m.Unlock()
	}(stringifiedData, md5OutChan, m)

	crc32OutChan := make(chan string)
	go calcCrc32SingleHash(stringifiedData, crc32OutChan, &wgLocal)

	md5 := <-md5OutChan
	crc32md5OutChan := make(chan string)
	go calcCrc32SingleHash(md5, crc32md5OutChan, &wgLocal)

	result := <-crc32OutChan + "~" + <-crc32md5OutChan
	out <- result
}

func SingleHash(in, out chan interface{}) {
	var wgGlobal sync.WaitGroup
	var mutex sync.Mutex
	for data := range in {
		wgGlobal.Add(1)
		go asyncCalcOneSingleHash(data, out, &wgGlobal, &mutex)
	}
	wgGlobal.Wait()
}

func asyncCalcOneMultiHash(data interface{}, out chan interface{}, wgGlobal *sync.WaitGroup, th int) {
	defer wgGlobal.Done()

	slice := make([]string, th)
	var wg sync.WaitGroup
	stringifiedData := fmt.Sprintf("%v", data)
	for i := 0; i < th; i++ {
		wg.Add(1)
		go func(index int) {
			slice[index] = DataSignerCrc32(strconv.Itoa(index) + stringifiedData)
			wg.Done()
		}(i)
	}
	wg.Wait()
	str := func(slice []string) string {
		contactedStr := ""
		for i := range slice {
			contactedStr += slice[i]
		}
		return contactedStr
	}(slice)
	out <- str
}

func MultiHash(in, out chan interface{}) {
	var wgGlobal sync.WaitGroup
	const th int = 6
	for data := range in {
		wgGlobal.Add(1)
		go asyncCalcOneMultiHash(data, out, &wgGlobal, th)
	}
	wgGlobal.Wait()
}

func CombineResults(in, out chan interface{}) {
	var slice []string
	for val := range in {
		slice = append(slice, fmt.Sprintf("%v", val))
	}
	sort.Strings(slice)
	str := strings.Join(slice, "_")
	out <- str
}
