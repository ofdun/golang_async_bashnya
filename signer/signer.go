package main

import (
	"fmt"
	"sort"
	"strconv"
	"sync"
)

func ExecutePipeline(jobs ...job) {
	var wg sync.WaitGroup
	in := make(chan interface{})
	out := make(chan interface{})
	wg.Add(len(jobs))
	for i := range jobs {
		go func(in, out chan interface{}, worker job, wg *sync.WaitGroup) {
			defer wg.Done()
			defer close(out)
			worker(in, out)
		}(in, out, jobs[i], &wg)
		in, out = out, make(chan interface{})
	}
	wg.Wait()
}

func SingleHash(in, out chan interface{}) {
	var mutex sync.Mutex
	var wg sync.WaitGroup
	for data := range in {
		wg.Add(2)
		stringifiedData := fmt.Sprintf("%v", data)

		md5OutChan := make(chan string)
		go func(data string, c chan string, m *sync.Mutex) {
			defer close(c)
			m.Lock()
			c <- DataSignerMd5(data)
			m.Unlock()
		}(stringifiedData, md5OutChan, &mutex)

		crc32OutChan := make(chan string)
		go func(data string, c chan string, wg *sync.WaitGroup) {
			defer wg.Done()
			defer close(c)
			c <- DataSignerCrc32(data)
		}(stringifiedData, crc32OutChan, &wg)

		md5 := <-md5OutChan
		crc32md5OutChan := make(chan string)
		go func(data string, c chan string, wg *sync.WaitGroup) {
			defer wg.Done()
			defer close(c)
			c <- DataSignerCrc32(data)
		}(md5, crc32md5OutChan, &wg)

		result := <-crc32OutChan + "~" + <-crc32md5OutChan
		out <- result
	}
}

func MultiHash(in, out chan interface{}) {
	slice := make([]string, 6)
	var wg sync.WaitGroup
	for val := range in {
		stringifiedData := fmt.Sprintf("%v", val)
		wg.Add(6)
		for i := 0; i < 6; i++ {
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
}

func CombineResults(in, out chan interface{}) {
	var slice []string
	for val := range in {
		slice = append(slice, fmt.Sprintf("%v", val))
	}
	sort.Strings(slice)
	str := func(slice []string) string {
		str := ""
		for i := range slice {
			if i == 0 {
				str = slice[i]
			} else {
				str += "_" + slice[i]
			}
		}
		return str
	}(slice)
	out <- str
}
