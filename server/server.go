// Package server package for server
package server

// refer to https://varshneyabhi.wordpress.com/2014/12/23/simple-udp-clientserver-in-golang/

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"CS425/CS425-MP2/model"
)

type suspeciousStatus int
type payloadType int
type messageType int

const (
	alive             suspeciousStatus = 0
	suspect           suspeciousStatus = 1
	payloadJoin       payloadType      = 0
	payloadLeave      payloadType      = 1
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
func NewServer(jsonFile []byte) *Server {
	server := &Server{}
	server.loadConfigFromJSON(jsonFile)
	return server
}

func (s *Server) loadConfigFromJSON(jsonFile []byte) error {
	return json.Unmarshal(jsonFile, &s.config)
}

// GetIP getip for server
func (s *Server) GetIP() string {
	return s.config.IP
}

// SetIP setip for server
func (s *Server) SetIP(IP string) {
	s.config.IP = IP
}

// GetPort getport of server
func (s *Server) GetPort() int {
	return s.config.Port
}

// SetPort setport for server
func (s *Server) SetPort(port int) {
	s.config.Port = port
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
	return nil
}

func (s *Server) generatePingList() {
	log.Fatalln("generatePingList TODO!!!")
}

func (s *Server) checkAvailability() {
	log.Fatalln("checkAvailability TODO!!!")
}

func (s *Server) suspectNode(nodeID string) {
	nodeInfo := s.memList[nodeID]
	nodeInfo.Inc++
	s.memList[nodeID] = nodeInfo

	_, ok := s.suspectList[nodeID]
	if !ok {
		s.suspectList[nodeID] = time.Now()
	}
	// Lu kindly set the TTL value correctly
	ttl := 3
	message := string(suspect) + "_" + nodeID + "_" + string(ttl)
	s.pushCachedMessage(payloadSuspicious, nodeID, []byte(message), s.config.DisseminationTimeout)

}

func (s *Server) pushCachedMessage(mType payloadType, nodeID string, message []byte, timeout time.Duration) {
	ipTS := string(message[1])

	switch mType {
	case payloadJoin:
		s.joinCachedMessage[ipTS] = int(message[2])
	case payloadLeave:
		s.leaveCachedMessage[ipTS] = int(message[2])
	case payloadSuspicious:
		s.suspiciousCachedMessage[ipTS] = suspiciousMessage{Type: suspect, Inc: int(message[2])}
	}
}

func (s *Server) getCachedMessages() [][]byte {
	// Get cached messages from s.suspiciousCachedMessage, s.joinCachedMessage, s.leaveCachedMessage
	messages := make([][]byte, 0)

	for k, v := range s.memList {
		v, ok := s.joinCachedMessage[k]
		buf := make([]byte, 0)
		if ok {
			buf = append(buf, byte(messageJoin))
			buf = append(buf, byte('_'))
			buf = append(buf, []byte(k)...)
			buf = append(buf, byte('_'))
			buf = append(buf, byte(v))
			messages = append(messages, buf)
		}
		v, ok = s.leaveCachedMessage[k]
		if ok {
			buf = append(buf, byte(messageJoin))
			buf = append(buf, byte('_'))
			buf = append(buf, []byte(k)...)
			buf = append(buf, byte('_'))
			buf = append(buf, byte(v))
			messages = append(messages, buf)
		}
		val, ok := s.suspiciousCachedMessage[k]
		if ok {
			buf = append(buf, byte(messageJoin))
			buf = append(buf, byte('_'))
			buf = append(buf, []byte(k)...)
			buf = append(buf, byte('_'))
			buf = append(buf, byte(val.Inc))
			messages = append(messages, buf)
		}
	}

	return messages
}

// Ping ping/ack error detection
func (s *Server) Ping(nodeID string, ch chan bool) {
	pingAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", strings.Split(nodeID, "-")[0], s.config.Port))
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
	messages := s.getCachedMessages()
	buf.WriteString(fmt.Sprintf("%d:%s", messagePing, s.config.ID))
	for _, message := range messages {
		buf.WriteString(fmt.Sprintf(":%s", message))
	}

	_, err = conn.Write(buf.Bytes())
	if err != nil {
		ch <- false
		return
	}

	n, addr, err := conn.ReadFrom(buf.Bytes())
	if err != nil {
		ch <- false
		return
	}
	s.DealWithMessage(n, addr, buf.Bytes())
}

// DealWithJoin will deal with new joins in our network
func (s *Server) DealWithJoin(inpMsg []byte) error {
	ipTS := bytes.Split(inpMsg, []byte(":"))[1]

	ni := nodeInfo{
		IP:         string(bytes.Split(ipTS, []byte("_"))[0]),
		Port:       s.config.Port,
		Inc:        0,
		Suspecious: alive,
	}
	s.memList[string(ipTS)] = ni

	s.pushCachedMessage(payloadJoin, string(ipTS), []byte(inpMsg), time.Millisecond*time.Duration(s.config.TTL))
	return nil
}

// DealWithMessage deal with all kinds of messages
// buf: 0:ip-ts:0_ip-ts_2:1_ip-ts_1:2_ip-ts_234:3_ip-ts_223
func (s *Server) DealWithMessage(n int, addr net.Addr, buf []byte) {
	fmt.Println("Received ", string(buf[0:n]), " from ", addr)

	bufList := bytes.Split(buf, []byte(":"))
	switch bufList[0] {
	case messageAck:

	case messagePing:
	case messageJoin:
		s.DealWithJoin(buf[2:])
	}

	inpMsg := string(buf[0:n])

	s.ServerConn.WriteTo(buf, addr)
}

// FailureDetection ping loop
func (s *Server) FailureDetection() error {
	for {
		s.checkAvailability()
		nodeID := s.pingList[s.pingIter]
		ch := make(chan bool)
		s.Ping(nodeID, ch)

		select {
		case res := <-ch:
			if res {
				s.pingIter++
			} else {
				s.suspectNode(nodeID)
			}
		case <-time.After(time.Duration(s.pingTimeout) * time.Millisecond):
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
