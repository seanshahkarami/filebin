package main

import (
	"flag"
	"log"
	"net/http"
	"time"
)

var serverAddr string
var serverRoot string

func init() {
	flag.StringVar(&serverAddr, "addr", ":8000", "address to listen on")
	flag.StringVar(&serverRoot, "root", "data", "data storage directory")
}

func main() {
	flag.Parse()

	server := http.Server{
		Addr:         serverAddr,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	http.Handle("/data/", http.StripPrefix("/data/", DataServer(serverRoot)))
	log.Printf("listening on %s", serverAddr)
	log.Fatal(server.ListenAndServe())
}
