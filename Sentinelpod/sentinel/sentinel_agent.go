package sentinel

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/meixiezichuan/broadcast-gossip/common"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

type NodeList []string
type Path []string

type HostMsg struct {
	Msg       common.NodeMessage
	SendPaths []Path
}

type NodeInfo struct {
	Cpu     string
	Battery string
	Mem     string
}

type Sentinel struct {
	BroadcastAddr string
	ListenAddr    string
	NodeId        string
	Revision      int
	NodeBuf       sync.Map
	Msgs          sync.Map
	Graph         *common.Graph
	MsgCnt        int
	BroadcastList []string

	// 固定三维坐标：x,y 固定为50,50；z 由 StatefulSet 下发
	X, Y, Z float64
}

var centerX, centerY float64 = 50.0, 50.0
var sphereRadius = 50.0

var TimeOutRev = 2 * time.Second

// InitSentinel 初始化哨兵，不使用随机坐标，而是固定 x=50, y=50, z由外部传入
func InitSentinel(nodeId string, port int, z float64) *Sentinel {
	if z > 100 {
		z = 100
	}
	if z < 0 {
		z = 0
	}
	s := Sentinel{
		BroadcastAddr: "255.255.255.255:" + strconv.Itoa(port),
		ListenAddr:    ":" + strconv.Itoa(port),
		NodeId:        nodeId,
		Revision:      0,
		Graph:         common.NewGraph(),
		MsgCnt:        0,
		X:             centerX,
		Y:             centerY,
		Z:             z,
	}
	return &s
}

func (s *Sentinel) SetBroadcastList(l []string) {
	s.BroadcastList = l
}

// 哨兵不主动更新坐标，UpdateCoordinates 保留但不在广播中调用
func (s *Sentinel) UpdateCoordinates(moveDistance float64) {
	// 哨兵位置固定，不执行坐标更新
}

// 生成Gossip消息，附带固定坐标信息
func (s *Sentinel) generateGossipMessage() common.GossipMessage {
	sendMsg := common.GossipMessage{}
	self := common.NodeMessage{s.NodeId, s.Revision, map[string]string{}, 0}
	// 附加固定的坐标信息
	nodeInfo := common.GenerateNodeInfo()
	nodeInfo["x"] = fmt.Sprintf("%.2f", s.X)
	nodeInfo["y"] = fmt.Sprintf("%.2f", s.Y)
	nodeInfo["z"] = fmt.Sprintf("%.2f", s.Z)
	self.Data = nodeInfo

	sendMsg.Self = self
	var sendMsgs []common.SendMessage

	s.UpdateGraph()

	var avaliblePath []string
	s.Msgs.Range(func(key, value interface{}) bool {
		m := value.(HostMsg)
		paths := m.SendPaths
		for _, p := range paths {
			pathKey := common.ListToString(p)
			pn := p[len(p)-1]
			if common.Contains(avaliblePath, pathKey) {
				nodeMsg := m.Msg
				nodeMsg.TTL += 1
				smsg := common.SendMessage{
					PrevNode: pn,
					NodeMsg:  nodeMsg,
				}
				sendMsgs = append(sendMsgs, smsg)
				break
			}

			if s.checkMsgSend(p) {
				avaliblePath = append(avaliblePath, pathKey)
				nodeMsg := m.Msg
				nodeMsg.TTL += 1
				smsg := common.SendMessage{
					PrevNode: pn,
					NodeMsg:  nodeMsg,
				}
				sendMsgs = append(sendMsgs, smsg)
				break
			}
		}
		s.Msgs.Delete(key)
		return true
	})
	//for _, n := range s.Graph.FindNeighbor(s.NodeId) {
	//	smsg := common.SendMessage{
	//		PrevNode: n,
	//		PrevAdj:  s.Graph.FindNeighbor(n),
	//		NodeMsg:  common.NodeMessage{},
	//	}
	//	sendMsgs = append(sendMsgs, smsg)
	//}
	sendMsg.Msgs = sendMsgs
	return sendMsg
}

func (s *Sentinel) Greeting() common.GossipMessage {
	var dMsgs []common.SendMessage
	for _, n := range s.Graph.FindNeighbor(s.NodeId) {
		var prevAdj []string
		for _, nn := range s.Graph.FindNeighbor(n) {
			prevAdj = append(prevAdj, nn)
		}
		sm := common.SendMessage{
			PrevNode: n,
			PrevAdj:  prevAdj,
		}
		dMsgs = append(dMsgs, sm)
	}
	nodeInfo := make(map[string]string)
	nodeInfo["x"] = fmt.Sprintf("%.2f", s.X)
	nodeInfo["y"] = fmt.Sprintf("%.2f", s.Y)
	nodeInfo["z"] = fmt.Sprintf("%.2f", s.Z)
	greeting := common.GossipMessage{
		Self: common.NodeMessage{
			NodeID:   s.NodeId,
			Revision: 0,
			Data:     nodeInfo,
		},
		Msgs: dMsgs,
	}
	return greeting
}

func (s *Sentinel) DoHelloBroadCast(msg common.GossipMessage, ep int) {
	if len(s.BroadcastList) > 0 {
		for _, h := range s.BroadcastList {
			baddr := h + s.ListenAddr
			s.SendMsg(baddr, msg)
		}
	} else {
		baddr := s.BroadcastAddr
		s.SendMsg(baddr, msg)
	}
	//fmt.Println(s.NodeId, "Sent hello msg:", msg)
}

func (s *Sentinel) DoBroadCast(msg common.GossipMessage, ep int) {
	l := 0
	if s.Revision < ep {
		l++
	}
	for _, m := range msg.Msgs {
		if !common.IsStructEmpty(m.NodeMsg) {
			if m.NodeMsg.Revision < ep {
				l++
			}
		}
	}
	s.MsgCnt = s.MsgCnt + l
	if len(s.BroadcastList) > 0 {
		for _, h := range s.BroadcastList {
			baddr := h + s.ListenAddr
			s.SendMsg(baddr, msg)
		}
	} else {
		baddr := s.BroadcastAddr
		s.SendMsg(baddr, msg)
	}
	//fmt.Println(s.NodeId, "Sent msg:", msg)
}

func (s *Sentinel) recordMsg(msg common.GossipMessage) {
	logPath := os.Getenv("LOG_PATH")
	if logPath == "" {
		logPath = "."
	}
	filename := logPath + "/_record/" + s.NodeId
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return
	}
	defer file.Close()
	logContent := fmt.Sprintf("%s %d\n", msg.Self.NodeID, msg.Self.Revision)
	for _, m := range msg.Msgs {
		if m.NodeMsg.NodeID == "" {
			continue
		}
		logContent += fmt.Sprintf("%s %d\n", m.NodeMsg.NodeID, m.NodeMsg.Revision)
	}
	if _, err := file.WriteString(logContent); err != nil {
		fmt.Printf("Error writing to file: %v\n", err)
	}
}

func (s *Sentinel) isMsgRecorded(nodeID string, revision int) bool {

	logPath := os.Getenv("LOG_PATH")
	if logPath == "" {
		logPath = "."
	}
	filename := logPath + "/_record/" + s.NodeId
	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return false
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var recordedNodeID string
		var recordedRevision int
		_, err := fmt.Sscanf(scanner.Text(), "%s %d", &recordedNodeID, &recordedRevision)
		if err != nil {
			continue
		}
		if recordedNodeID == nodeID && recordedRevision == revision {
			return true
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading file: %v\n", err)
	}
	return false
}

func (s *Sentinel) UpdateGraph() {
	s.NodeBuf.Range(func(key, value interface{}) bool {
		n := key.(string)
		t := value.(int64)
		now := time.Now().UnixNano()
		if now-t > TimeOutRev.Nanoseconds() {
			s.Graph.RemoveEdge(s.NodeId, n)
		}
		return true
	})
}

func (s *Sentinel) WriteMsg(msg common.NodeMessage, ep int) {
	if msg.Revision >= ep || msg.Revision == 0 {
		return
	}
	logPath := os.Getenv("LOG_PATH")
	if logPath == "" {
		logPath = "."
	}
	filename := logPath + "/" + s.NodeId
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening or creating file:", err)
		return
	}
	defer file.Close()
	latency := msg.TTL
	msgWritten := msg.NodeID + "_" + strconv.Itoa(msg.Revision) + " " + strconv.Itoa(latency) + "\n"
	_, err = file.WriteString(msgWritten)
	if err != nil {
		fmt.Println("Error writing to file:", err)
	}
}

func (s *Sentinel) SendMsg(baddr string, msg common.GossipMessage) {
	addr, err := net.ResolveUDPAddr("udp", baddr)
	if err != nil {
		fmt.Printf("%s Error resolving address: %v\n", s.NodeId, err)
		return
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		fmt.Printf("%s Error dialing UDP: %v\n", s.NodeId, err)
		return
	}
	bytes, err := json.Marshal(msg)
	defer conn.Close()
	_, err = conn.Write(bytes)
	if err != nil {
		fmt.Printf("%s Error writing UDP: %v\n", s.NodeId, err)
	}
}

func (s *Sentinel) checkMsgSend(p []string) bool {
	if p[0] == s.NodeId {
		return false
	}
	prevNode := p[len(p)-1]
	subGraph := s.Graph.GetSubgraphWithinHops(prevNode, 2)
	allP := append(p, s.NodeId)
	if s.PathExistInMLST(subGraph, allP) {
		return true
	}
	return false
}

func (s *Sentinel) periodicHelloBroadcast(stopCh <-chan bool, distance int, ep int) {
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-stopCh:
			fmt.Println(s.NodeId, "stopping periodic hello broadcast")
			return
		case <-ticker.C:
			// 当广播轮次达到预设值时停止
			if s.Revision >= ep+10 {
				fmt.Println(s.NodeId, "broadcast rounds reached, stopping periodic broadcast")
				return
			}
			s.UpdateGraph()
			msg := s.Greeting()
			s.DoHelloBroadCast(msg, ep)
		}
	}
}

// 周期性广播函数，哨兵启动后不主动广播，而是在首次接收到消息后启动周期广播
func (s *Sentinel) periodicBroadcast(stopCh <-chan bool, ep int, distance int) {
	now := time.Now()
	nextTenMinute := now.Truncate(5 * time.Minute).Add(5 * time.Minute)
	time.Sleep(time.Until(nextTenMinute))
	interval := 10
	timeoutStr, exists := os.LookupEnv("BROADCAST_INTERVAL")
	if exists {
		if seconds, err := strconv.Atoi(timeoutStr); err == nil {
			interval = seconds
		}
	}

	t := rand.Intn(interval * 1000)
	time.Sleep(time.Duration(t) * time.Millisecond)

	ticker := time.NewTicker(time.Duration(interval) * time.Second) // 每10秒广播一次
	defer func() {
		ticker.Stop()
		// 在退出时输出统计信息
		logPath := os.Getenv("LOG_PATH")
		if logPath == "" {
			logPath = "."
		}
		filename := logPath + "/" + s.NodeId
		file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			outStr := fmt.Sprintf("Sent Message Count: %d in %d Epochs", s.MsgCnt, s.Revision)
			file.WriteString(outStr)
			file.Close()
		} else {
			fmt.Printf("Error opening log file: %v\n", err)
		}
	}()
	for {
		select {
		case <-stopCh:
			fmt.Println(s.NodeId, "stopping periodic broadcast")
			return
		case <-ticker.C:
			// 当广播轮次达到预设值时停止
			if s.Revision >= ep+10 {
				fmt.Println(s.NodeId, "broadcast rounds reached, stopping periodic broadcast")
				return
			}
			s.Revision++
			msg := s.generateGossipMessage()
			s.recordMsg(msg)
			s.DoBroadCast(msg, ep)
			fmt.Println(s.NodeId, "Periodic broadcast executed, revision:", s.Revision)
		}
	}
}

// Start 函数：哨兵启动后只启动接收消息，不主动广播
func (s *Sentinel) Start(stopCh chan bool, ep int, distance int) {
	addr, err := net.ResolveUDPAddr("udp", s.ListenAddr)
	//TimeOutRev = ep
	if err != nil {
		log.Fatalf("Failed to resolve UDP address: %v\n", err)
	}
	fmt.Println(s.NodeId, "udpListen:", addr)
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatalf("%s Failed to listen on UDP: %v\n", s.NodeId, err)
	}

	// // —— 在这里设置 socket 缓冲区大小
	// const bufSize = 2 << 30 // 2 GiB
	// if err := conn.SetReadBuffer(bufSize); err != nil {
	//     log.Printf("Warning: SetReadBuffer(%d) failed: %v\n", bufSize, err)
	// }
	// if err := conn.SetWriteBuffer(bufSize); err != nil {
	//     log.Printf("Warning: SetWriteBuffer(%d) failed: %v\n", bufSize, err)
	// }

	timeoutStr, exists := os.LookupEnv("TIME_OUT_INTERVAL")
	if exists {
		if seconds, err := strconv.Atoi(timeoutStr); err == nil {
			TimeOutRev = time.Duration(seconds) * time.Second
		}
	}

	defer conn.Close()
	go s.periodicHelloBroadcast(stopCh, distance, ep)
	go s.periodicBroadcast(stopCh, distance, ep)
	s.ReceiveMsg(conn, stopCh, distance, ep)
}
