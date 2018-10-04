package main

import (
	"CS425/CS425-MP2/server"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
)

// This function will register and initiate server
func main() {
	// parse argument
	configFilePath := flag.String("c", "./config.json", "Config file path")
	port := flag.Int("p", 8080, "Port number")
	IP := flag.String("ip", "0.0.0.0", "IP address")

	flag.Parse()

	// load config file
	configFile, e := ioutil.ReadFile(*configFilePath)
	if e != nil {
		log.Fatalf("File error: %v\n", e)
	}

	// Class for server
	s := server.NewServer(configFile)

	s.SetIP(*IP)
	s.SetPort(*port)

	fmt.Printf("Starting server on IP: %s and port: %d", *IP, *port)

	go s.FailureDetection()
	go s.ServerLoop()
}
