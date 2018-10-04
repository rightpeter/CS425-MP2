// refer to https://varshneyabhi.wordpress.com/2014/12/23/simple-udp-clientserver-in-golang/
package main

import (
	"log"
	"net"
	"server"
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
	s := newServer()
	s.loadConfigFromJSON(configFile)

	s.setIP(*IP)
	s.setPort(*port)
	s.setFilePath(server.getFilePath())

	fmt.Printf("Starting server on IP: %s and port: %d", *IP, *port)

	go server.FailureDetection()
	go server.ServerLoop()
}
