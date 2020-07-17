package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

func main() {
	r := mux.NewRouter()

	r.HandleFunc("/", defaultHandler)
	r.HandleFunc("/hj", hjHandler)
	r.HandleFunc("/dup", dupHeaderHandler)
	r.HandleFunc("/dup2", dupHeaderHandler2)
	r.HandleFunc("/world", helloWorldHandler)
	r.HandleFunc("/slow", slowHandler)
	r.HandleFunc("/status/{code}", statusHandler)

	fmt.Println("Listening on port 8443 for TLS")
	go http.ListenAndServeTLS(":8443", "server.crt", "server.key", r)

	fmt.Println("Listening on port 8080")
	http.ListenAndServe(":8080", r)

}

// defaultHandler just return OK
func defaultHandler(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "OK")
}

// dupHeaderHandler hihack the connexion and send a duplicated 'Transfer-Encoding: chunked' header
func dupHeaderHandler(w http.ResponseWriter, r *http.Request) {

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
	bufrw.WriteString("HTTP/1.1 200 OK\r\nx-content-type-options: nosniff\r\nTransfer-Encoding: chunked\r\nx-content-type-options: nosniff\r\nContent-Type: text/plain; charset=utf-8\r\nTransfer-Encoding: chunked\r\n\r\n2\r\nOK\r\n0\r\n")
	bufrw.Flush()
}

// dupHeaderHandler hihack the connexion and send a duplicated header
func dupHeaderHandler2(w http.ResponseWriter, r *http.Request) {

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
	bufrw.WriteString("HTTP/1.1 200 OK\r\nContent-Length: 2\r\nx-content-type-options: nosniff\r\nx-content-type-options: nosniff\r\nContent-Type: text/plain; charset=utf-8\r\n\r\nOK")
	bufrw.Flush()
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

// slowHandler return a "Hello World" message after a delay, defaults to 20s
func statusHandler(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)

	i, err := strconv.Atoi(vars["code"])
	if err != nil {
		http.Error(w, "error code not supported", http.StatusInternalServerError)
		return
	}

	// sleep if needed
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

	w.WriteHeader(i)
	fmt.Fprintf(w, "Error: %d\n", i)
}
