package main

import (
	"fmt"
	"strconv"
	"sync"
	"time"
)

func ExecutePipeline(jobs ...job) {

}

func runDataSignerCrc32(in <-chan interface{}, out chan<- interface{}) {
	select {
	case data := <-in:
		out <- DataSignerCrc32(data.(string))
	}
}

func SingleHash(in, out chan interface{}) {
	select {
	case data := <-in:
		crc32InChan := make(chan interface{}, 1)
		crc32OutChan := make(chan interface{}, 1)
		go runDataSignerCrc32(crc32InChan, crc32OutChan)
		crc32InChan <- data.(string)

		go runDataSignerCrc32(crc32InChan, crc32OutChan)
		md5 := DataSignerMd5(data.(string))
		crc32InChan <- md5

		result := (<-crc32OutChan).(string) + "~" + (<-crc32OutChan).(string)
		out <- result
		close(out)
	}
}

func Test() {
	start := time.Now()

	singleHashIn := make(chan interface{}, 1)
	singleHashOut := make(chan interface{}, 1)

	go SingleHash(singleHashIn, singleHashOut)
	singleHashIn <- "0" // TODO
	singleHash := (<-singleHashOut).(string)

	multiHashIn := make(chan interface{}, 6)
	multiHashOut := make(chan interface{}, 6)
	slice := make([]string, 0, 6)

	go func() {
		for i := 0; i < 6; i++ {
			multiHashIn <- strconv.Itoa(i) + singleHash
		}
		close(multiHashIn)
	}()

	go MultiHash(multiHashIn, multiHashOut)

	for val := range multiHashOut {
		slice = append(slice, val.(string))
	}

	fmt.Println(slice)
	fmt.Println(time.Since(start))
}

func MultiHash(in, out chan interface{}) {
	var wg sync.WaitGroup

	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runDataSignerCrc32(in, out)
		}()
	}

	wg.Wait()
	close(out)
}

func CombineResults(in, out chan interface{}) {

}
