package main

import (
	"byte"
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

type suspeciousStatus int
type payloadType int
type messageType int

const (
	alive             suspeciousStatus = 0
	suspect           suspeciousStatus = 1
	payloasJoin       payloadType      = 0
	payloasLeave      payloadType      = 1
	payloadSuspicious payloadType      = 2
	payloadAlive      payloadType      = 3
	messagePing       messageType      = 0
	messageAck        messageType      = 1
	messageJoin       messageType      = 2
)

type nodeInfo struct {
	IP         string
	Port       int
	Inc        int
	Suspecious suspeciousStatus
}

type suspiciousMessage struct {
	Type suspeciousStatus
	Inc  int
}

// Server server class
type Server struct {
	config      model.NodeConfig
	pingIter    int
	pingTimeout int //Millisecond
	ServerConn  *net.UDPConn
	memList     map[string]nodeInfo // { "id-ts": {"ip": "192.168.1.1", "port": 8081, "inc": 0, suspicious: 0} }
	// { "id-ts": { "type" : 0, "int": 0 } }
	suspiciousCachedMessage map[string]suspiciousMessage
	joinCachedMessage       map[string]int       // {"id-ts": #TTL}
	leaveCachedMessage      map[string]int       // {"id-ts": #TTL}
	suspectList             map[string]time.Time // {"id-ts": timestamp}
	pingList                []string             // ['id-ts']
}

// NewServer init a server
func NewServer() *Server {
	return &Server{}
}

func (s *Server) loadConfigFromJSON(jsonFile []byte) error {
	return json.Unmarshal(jsonFile, &s.config)
}

func (s *Server) getIP() string {
	return s.config.Current.IP
}

func (s *Server) setIP(IP string) {
	s.config.Current.IP = IP
}

func (s *Server) getPort() int {
	return s.config.Current.Port
}

func (s *Server) setPort(port int) {
	s.config.Current.Port = port
}

func (s *Server) getIP() string {
	return s.config.Current.IP
}

// ListenUDP Server listen to udp
func (s *Server) ListenUDP() error {
	/* Lets prepare a address at any address at port s.config.Port*/
	serverAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", s.config.Port))
	if err != nil {
		return err
	}

	/* Now listen at selected port */
	s.ServerConn, err = net.ListenUDP("udp", serverAddr)
	if err != nil {
		return err
	}
}

func (s *Server) generatePingList() {
	log.Fatalln("generatePingList TODO!!!")
}

func (s *Server) checkAvailability() {
	log.Fatalln("checkAvailability TODO!!!")
}

func (s *Server) suspectNode(nodeID string) {
	log.Fatalln("suspectNode TODO!!!")
}

func (s *Server) pushCachedMessage(mType payloadType, nodeID string, message []byte, timeout time.Time) {
	log.Fatalln("pushCachedMessage TODO!!!")
}

func (s *Server) getCachedMessage() {
	// Get cached messages from s.suspiciousCachedMessage, s.joinCachedMessage, s.leaveCachedMessage
	log.Fatalln("getCachedMessage TODO!!!")
}

// Ping ping/ack error detection
func (s *Server) Ping(nodeID string, ch chan bool) {
	pingAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("$s:$d", strings.Split(nodeID, '-')[0], s.config.Port))
	if err != nil {
		ch <- false
		return
	}

	conn, err := net.DialUDP("udp", nil, pingAddr)
	if err != nil {
		ch <- false
		return
	}

	defer conn.Close()

	var buf bytes.Buffer
	messages := s.getCachedMessage()
	buf.WriteString(fmt.Sprintf("%d:%s", messagePing, s.config.ID))
	for _, message := range messages {
		buf.WriteString(fmt.Sprintf(":%s", message))
	}

	_, err := conn.Write(buf)
	if err != nil {
		ch <- false
		return
	}

	n, addr, err := conn.ReadFrom(buf)
	if err != nil {
		ch <- false
		return
	}
	s.DealWithMessage(n, addr, buf)
}

// DealWithMessage deal with all kinds of messages
// buf: 0:ip-ts:0_ip-ts_2:1_ip-ts_1:2_ip-ts_234:3_ip-ts_223
func (s *Server) DealWithMessage(n int, addr *net.UDPAddr, buf []byte) {
	fmt.Println("Received ", string(buf[0:n]), " from ", addr)
	s.ServerConn.WriteTo(buf, addr)
}

// FailureDetection ping loop
func (s *Server) FailureDetection() error {
	for {
		s.checkAvailability()
		nodeID := s.pingList[s.pingIter]
		ch := make(chan string)
		s.Ping(nodeID, ch)

		select {
		case res <- ch:
			if res {
				pingIter++
			} else {
				s.suspectNode(nodeID)
			}
		case <-time.After(s.pingTimeout * time.Millisecond):
			s.suspectNode(nodeID)
		}

		s.pingIter++
		if s.pingIter == 0 {
			s.generatePingList()
		}
	}
}

// ServerLoop main server loop: listen on s.config.port for incoming udp package
func (s *Server) ServerLoop() {
	err := s.ListenUDP()
	if err != nil {
		log.Fatalf("ListenUDP Fail: %v\n", err)
	}
	defer s.ServerConn.Close()

	buf := make([]byte, 1024)
	for {
		n, addr, err := s.ServerConn.ReadFromUDP(buf)
		fmt.Println("Received ", string(buf[0:n]), " from ", addr)

		if err != nil {
			fmt.Println("Error: ", err)
		}
		go s.DealWithMessage(n, addr, buf)
	}
}
