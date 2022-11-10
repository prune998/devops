package main

import (
	"net/http"
	"sync"
)

func main() {
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)

		go func() {
			http.Get(`https://httpstat.us/200?sleep=100000`)

			wg.Done()
		}()
	}

	wg.Wait()
}
