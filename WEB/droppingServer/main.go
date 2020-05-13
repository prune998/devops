package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
)

func main() {
	http.HandleFunc("/", defaultHandler)
	http.HandleFunc("/hj", hjHandler)
	http.HandleFunc("/world", helloWorldHandler)
	http.HandleFunc("/slow", slowHandler)

	fmt.Println("Listening on port 8443 for TLS")
	go http.ListenAndServeTLS(":8443", "server.crt", "server.key", nil)

	fmt.Println("Listening on port 8080")
	http.ListenAndServe(":8080", nil)

}

// defaultHandler just return OK
func defaultHandler(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "OK")
}

// hjHandler hihack the connexion and send data before the server can return it's HTTP headers
func hjHandler(w http.ResponseWriter, r *http.Request) {

	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "webserver doesn't support hijacking", http.StatusInternalServerError)
		return
	}

	conn, bufrw, err := hj.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Don't forget to close the connection:
	defer conn.Close()
	bufrw.WriteString("connexion hijacked by the server")
	bufrw.Flush()

	io.WriteString(w, "Hello world!")
}

// helloWorldHandler returns a "Hello World" message
func helloWorldHandler(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "Hello world!")
}

// slowHandler return a "Hello World" message after a delay, defaults to 20s
func slowHandler(w http.ResponseWriter, r *http.Request) {

	delay, ok := r.URL.Query()["delay"]

	if !ok || len(delay[0]) < 1 {
		log.Println("Url Param 'delay' is missing from the /slow call. Use /slow?delay=20")
		return
	}

	// convert string delay to int
	sleepDelay, err := strconv.Atoi(delay[0])
	if err != nil {
		log.Println("delay parameter not set, using 20s")
		sleepDelay = 20
	}
	time.Sleep(time.Duration(sleepDelay) * time.Second)
	io.WriteString(w, "Hello world!")
}
