package main

import (
	"CS425/CS425-MP2/server"
	"flag"
	"io/ioutil"
	"log"
	"os"
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

	f, err := os.OpenFile(s.GetConfigPath(), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()

	log.SetOutput(f)
	log.Printf("Starting server on IP: %s and port: %d", *IP, *port)

	go s.FailureDetection()
	s.ServerLoop()
}
