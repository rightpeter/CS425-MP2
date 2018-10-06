// Package server package for server
package server

// refer to https://varshneyabhi.wordpress.com/2014/12/23/simple-udp-clientserver-in-golang/

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"sort"
	"strings"
	"time"

	"CS425/CS425-MP2/model"
)

type suspiciousStatus uint8

const (
	suspiciousAlive   suspiciousStatus = 0
	suspiciousSuspect suspiciousStatus = 1
	suspiciousFail    suspiciousStatus = 2
)

type payloadType uint8

const (
	payloadJoin       payloadType = 0
	payloadLeave      payloadType = 1
	payloadSuspicious payloadType = 2
	payloadAlive      payloadType = 3
	payloadFail       payloadType = 4
)

type messageType uint8

const (
	messagePing    messageType = 0
	messageAck     messageType = 1
	messageJoin    messageType = 2
	messageMemList messageType = 3
)

type suspiciousMessage struct {
	Type suspiciousStatus
	Inc  uint8
	TS   time.Time
}

type sortMemList []string

func (s sortMemList) Len() int {
	return len(s)
}
func (s sortMemList) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s sortMemList) Less(i, j int) bool {
	ipi := strings.Split(s[i], "-")[0]
	ipj := strings.Split(s[j], "-")[0]
	return ipi < ipj
}

func shuffleMemList(memList []string) {
	rand.Seed(time.Now().UnixNano())
	for i := len(memList) - 1; i > 0; i-- { // Fisher–Yates shuffle
		j := rand.Intn(i + 1)
		memList[i], memList[j] = memList[j], memList[i]
	}
}

// Server server class
type Server struct {
	ID            string
	config        model.NodeConfig
	pingIter      int
	ServerConn    *net.UDPConn
	memList       map[string]uint8 // { "id-ts": 0 }
	sortedMemList []string         // ["id-ts", ...]
	// { "id-ts": { "type" : 0, "int": 0 } }
	suspiciousCachedMessage map[string]suspiciousMessage
	joinCachedMessage       map[string]uint8     // {"id-ts": #TTL}
	leaveCachedMessage      map[string]uint8     // {"id-ts": #TTL}
	suspectList             map[string]time.Time // {"id-ts": timestamp}
	pingList                []string             // ['id-ts']
	failTimeout             time.Duration
	cachedTimeout           time.Duration
}

// NewServer init a server
func NewServer(jsonFile []byte) *Server {
	server := &Server{}
	server.loadConfigFromJSON(jsonFile)
	server.init()
	return server
}

// GetConfigPath export config file path
func (s *Server) GetConfigPath() string {
	return s.config.LogPath
}

func (s *Server) findIndexInSortedMemList(nodeID string) int {
	for k, v := range s.sortedMemList {
		if v == nodeID {
			return k
		}
	}
	return -1
}

func (s *Server) calculateTimeoutDuration(timeout int) time.Duration {
	return time.Duration(timeout) * time.Millisecond
}

func (s *Server) getIncFromCachedMessages(nodeID string) uint8 {
	return s.suspiciousCachedMessage[nodeID].Inc
}

func (s *Server) loadConfigFromJSON(jsonFile []byte) error {
	return json.Unmarshal(jsonFile, &s.config)
}

func (s *Server) init() {
	s.ID = fmt.Sprintf("%s-%d", s.config.IP, time.Now().Unix())
	s.pingIter = 0
	s.memList = map[string]uint8{s.ID: 0}
	s.generateSortedMemList()
	s.suspiciousCachedMessage = map[string]suspiciousMessage{}
	s.joinCachedMessage = map[string]uint8{}
	s.leaveCachedMessage = map[string]uint8{}
	s.suspectList = map[string]time.Time{}
	s.pingList = []string{}
	s.failTimeout = s.calculateTimeoutDuration(s.config.FailTimeout)
	s.cachedTimeout = s.calculateTimeoutDuration(s.config.DisseminationTimeout)
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

func (s *Server) generateSortedMemList() {
	tmpMemList := []string{}
	for k := range s.memList {
		tmpMemList = append(tmpMemList, k)
	}
	sort.Sort(sortMemList(tmpMemList))
	s.sortedMemList = tmpMemList
}

func (s *Server) generatePingList() error {
	i := s.findIndexInSortedMemList(s.ID)
	if i >= 4 {
		s.pingList = []string{}
		s.pingIter = 0
	} else if i == -1 {
		return errors.New("self is not in s.sortedMemList")
	} else {
		coreNodeList := s.sortedMemList[0:4]
		leafNodeList := []string{}

		for j := 4; j < len(s.sortedMemList); j++ {
			if j%4 == i {
				leafNodeList = append(leafNodeList, s.sortedMemList[i])
			}
		}
		shuffleMemList(coreNodeList)
		shuffleMemList(leafNodeList)
		s.pingList = append(coreNodeList, leafNodeList...)
		s.pingIter = 0
	}
	return nil
}

func (s *Server) newNode(nodeID string, inc uint8) {
	if _, ok := s.memList[nodeID]; !ok {
		s.memList[nodeID] = inc
		s.generateSortedMemList()
		s.generatePingList()
	} else {
		if inc > s.memList[nodeID] {
			s.memList[nodeID] = inc
		}
	}
}

func (s *Server) deleteNode(nodeID string) {
	if _, ok := s.memList[nodeID]; ok {
		delete(s.memList, nodeID)
	}
}

func (s *Server) suspectNode(nodeID string, failTimeout time.Duration, cachedTimeout time.Duration) {
	if _, ok := s.suspectList[nodeID]; !ok {
		s.suspectList[nodeID] = time.Now().Add(failTimeout)
		go s.failNode(nodeID, failTimeout)
		s.pushSuspiciousCachedMessage(suspiciousSuspect, nodeID, s.memList[nodeID], cachedTimeout)
	}

}

func (s *Server) failNode(nodeID string, timeout time.Duration) {
	time.Sleep(timeout)
	if _, ok := s.suspectList[nodeID]; !ok {
		delete(s.suspectList, nodeID)
		if _, ok := s.memList[nodeID]; !ok {
			delete(s.memList, nodeID)
			s.generateSortedMemList()
			s.generatePingList()
			s.pushSuspiciousCachedMessage(suspiciousFail, nodeID, s.getIncFromCachedMessages(nodeID), s.cachedTimeout)
		}
	}
}

func (s *Server) pushSuspiciousCachedMessage(sStatus suspiciousStatus, nodeID string, inc uint8, timeout time.Duration) {
	susMessage := s.suspiciousCachedMessage[nodeID]
	if susMessage.Type == suspiciousFail {
		return
	}

	newTS := time.Now().Add(timeout) // timeout = s.calculateTimeoutDuration(s.config.XXTimeout)
	if sStatus == suspiciousFail && inc > susMessage.Inc {
		s.suspiciousCachedMessage[nodeID] = suspiciousMessage{Type: sStatus, Inc: inc, TS: newTS}
	} else if inc == susMessage.Inc && susMessage.Type == suspiciousAlive && sStatus == suspiciousSuspect {
		s.suspiciousCachedMessage[nodeID] = suspiciousMessage{Type: sStatus, Inc: inc, TS: newTS}
	}
}

func (s *Server) pushJoinCachedMessage(nodeID string, ttl uint8) {
	s.joinCachedMessage[nodeID] = ttl
}

func (s *Server) pushLeaveCachedMessage(nodeID string, ttl uint8) {
	s.leaveCachedMessage[nodeID] = ttl
}

func (s *Server) getCachedMessages() [][]byte {
	// Get cached messages from s.suspiciousCachedMessage, s.joinCachedMessage, s.leaveCachedMessage
	messages := make([][]byte, 0)

	for k, v := range s.joinCachedMessage {
		buf := []byte{byte(payloadJoin)}
		buf = append(buf, byte('_'))
		buf = append(buf, []byte(k)...)
		buf = append(buf, byte('_'))
		buf = append(buf, byte(v))
		messages = append(messages, buf)
	}

	for k, v := range s.leaveCachedMessage {
		buf := []byte{byte(payloadLeave)}
		buf = append(buf, byte('_'))
		buf = append(buf, []byte(k)...)
		buf = append(buf, byte('_'))
		buf = append(buf, byte(v))
		messages = append(messages, buf)
	}

	for k, v := range s.suspiciousCachedMessage {
		if time.Now().Sub(v.TS) > 0 {
			delete(s.suspiciousCachedMessage, k)
		} else {
			buf := []byte{}
			switch v.Type {
			case suspiciousAlive:
				buf = append(buf, byte(payloadAlive))
			case suspiciousSuspect:
				buf = append(buf, byte(payloadSuspicious))
			case suspiciousFail:
				buf = append(buf, byte(payloadFail))
			}
			buf = append(buf, byte('_'))
			buf = append(buf, []byte(k)...)
			buf = append(buf, byte('_'))
			buf = append(buf, byte(v.Inc))
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

	payloads := s.getCachedMessages()
	replyBuf := s.generateReplyBuffer(messagePing, payloads)

	_, err = conn.Write(replyBuf)
	if err != nil {
		ch <- false
		return
	}

	buf := []byte{}
	_, _, err = conn.ReadFrom(buf)
	if err != nil {
		ch <- false
		return
	}
	bufList := bytes.Split(buf, []byte(":"))
	if bufList[0][0] == byte(messageAck) {
		s.DealWithPayloads(bufList[2:])
	}
}

// DealWithJoin will deal with new joins in our network
func (s *Server) DealWithJoin(inpMsg []byte) {
	nodeID := string(inpMsg)

	s.newNode(nodeID, 0)
	s.pushJoinCachedMessage(nodeID, s.config.TTL)
}

// DealWithPayloads deal with all kinds of messages
// payloads: [[0_ip-ts_2], [1_ip-ts_1], [2_ip-ts_234], [3_ip-ts_223]]
func (s *Server) DealWithPayloads(payloads [][]byte) {
	for _, payload := range payloads {
		message := bytes.Split(payload, []byte("_"))
		// message = [[0], []byte("ip-ts"), [2]]
		nodeID := string(message[1])
		switch payloadType(message[0][0]) {
		case payloadJoin:
			s.newNode(nodeID, 0)
			ttl := uint8(message[2][0]) - 1
			s.pushJoinCachedMessage(nodeID, ttl)
		case payloadLeave:
			s.deleteNode(nodeID)
			ttl := uint8(message[2][0]) - 1
			s.pushLeaveCachedMessage(nodeID, ttl)
		case payloadSuspicious:
			inc := uint8(message[2][0])
			if nodeID == s.ID {
				if inc >= s.memList[s.ID] {
					s.memList[s.ID] = inc + 1
					s.pushSuspiciousCachedMessage(suspiciousAlive, nodeID, s.memList[s.ID], s.cachedTimeout)
				}
			} else {
				s.pushSuspiciousCachedMessage(suspiciousSuspect, nodeID, inc, s.cachedTimeout)
			}
		case payloadAlive:
			inc := uint8(message[2][0])
			if _, ok := s.suspectList[nodeID]; ok && s.memList[nodeID] < inc {
				delete(s.suspectList, nodeID)
				s.memList[nodeID] = inc
			}
			s.pushSuspiciousCachedMessage(suspiciousAlive, nodeID, inc, s.cachedTimeout)
		}

	}
}

// DealWithMemList deal with messages contains memList
func (s *Server) DealWithMemList(bufList [][]byte) {
	for _, buf := range bufList {
		message := bytes.Split(buf, []byte("_"))
		nodeID := string(message[0])
		inc := uint8(message[1][0])
		s.newNode(nodeID, inc)
	}
}

// FailureDetection ping loop
func (s *Server) FailureDetection() error {
	for {
		time.Sleep(time.Duration(s.config.PeriodTime) * time.Millisecond)
		log.Printf("s.pingList: %v, s.pingIter: %d\n", s.pingList, s.pingIter)
		if len(s.pingList) == 0 {
			continue
		}
		nodeID := s.pingList[s.pingIter]
		ch := make(chan bool)
		s.Ping(nodeID, ch)

		select {
		case res := <-ch:
			if res {
				s.pingIter++
			} else {
				s.suspectNode(nodeID, s.failTimeout, s.cachedTimeout)
			}
		case <-time.After(time.Duration(s.config.PingTimeout) * time.Millisecond):
			s.suspectNode(nodeID, s.failTimeout, s.cachedTimeout)
		}

		s.pingIter++
		if s.pingIter == 0 {
			s.generatePingList()
		}
	}
}

// buf: 0:ip-ts:0_ip-ts_2:1_ip-ts_1:2_ip-ts_234:3_ip-ts_223
func (s *Server) generateReplyBuffer(mType messageType, payloads [][]byte) []byte {
	replyBuf := []byte{byte(mType)}              // messagePing
	replyBuf = append(replyBuf, ':')             // messagePing:
	replyBuf = append(replyBuf, []byte(s.ID)...) // messagePing:ip-ts
	for _, payload := range payloads {
		//payload: 0_ip-ts_342
		replyBuf = append(replyBuf, ':')
		replyBuf = append(replyBuf, payload...)
	}
	return replyBuf
}

// buf: 3:ip-ts:ip-ts:ip-ts
func (s *Server) generateJoinReplyBuffer() []byte {
	replyBuf := []byte{byte(messageMemList)} // messageMemList

	for _, nodeID := range s.sortedMemList {
		replyBuf = append(replyBuf, ':')
		replyBuf = append(replyBuf, []byte(nodeID)...)
		replyBuf = append(replyBuf, '_')
		replyBuf = append(replyBuf, byte(s.memList[nodeID]))
	}
	return replyBuf
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
		if err != nil {
			fmt.Println("Error: ", err)
		}

		// buf: 0:ip-ts:0_ip-ts_2:1_ip-ts_1:2_ip-ts_234:3_ip-ts_223
		fmt.Println("Received ", string(buf[0:n]), " from ", addr)

		bufList := bytes.Split(buf, []byte(":"))
		// bufList[0]: [messageType]
		// bufList[1]: ip-ts
		// bufList[2:]: payload messages
		switch messageType(bufList[0][0]) {
		case messageAck:
			s.DealWithPayloads(bufList[2:])
		case messagePing:
			s.DealWithPayloads(bufList[2:])
			payloads := s.getCachedMessages()
			replyBuf := s.generateReplyBuffer(messageAck, payloads)
			s.ServerConn.WriteTo(replyBuf, addr)
		case messageJoin:
			s.DealWithJoin(bufList[1])
			replyBuf := s.generateJoinReplyBuffer()
			s.ServerConn.WriteTo(replyBuf, addr)
		case messageMemList:
			// bufList[0]: [messageMemList]
			// bufList[1:]: [[ip-ts], [ip-ts], ...]
			s.DealWithMemList(bufList[1:])
		}
	}
}
