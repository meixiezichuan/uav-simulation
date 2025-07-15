package gossip

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/meixiezichuan/broadcast-gossip/common"
	"log"
	"math"
	"math/rand"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

// 定义类型
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

type Agent struct {
	BroadcastAddr string
	ListenAddr    string
	NodeId        string
	Revision      int
	NodeBuf       sync.Map
	Msgs          sync.Map
	Graph         *common.Graph
	MsgCnt        int
	BroadcastList []string
	Markov        *common.GaussMarkov3D

	// 三维坐标
	X, Y, Z float64
}

// 水平中心坐标
var centerX, centerY float64 = 50.0, 50.0

// 垂直区间固定为 [0, 100]
var zMin, zMax float64 = 0.0, 100.0

var TimeOutRev = 2 * time.Second

// randomCoordinatesInRange 根据给定 r 生成普通 Pod 的初始坐标，要求：
// x ∈ [50 - (√3/2 * r), 50 + (√3/2 * r)]
// y 同上；
// z ∈ [ 0, 100 ]
func randomCoordinatesInRange(r float64) (float64, float64, float64) {
	delta := (math.Sqrt(3) / 2) * r
	x := centerX - delta + rand.Float64()*(2*delta)
	y := centerY - delta + rand.Float64()*(2*delta)
	z := rand.Float64() * 100
	x = common.RoundFloat(x, 2)
	y = common.RoundFloat(y, 2)
	z = common.RoundFloat(z, 2)
	return x, y, z
}

// InitAgent 修改为接收 r 参数（这里 r 就是传入的 distance 用于限定坐标范围）
func InitAgent(nodeId string, port int, r float64) *Agent {
	agent := Agent{
		BroadcastAddr: "255.255.255.255:" + strconv.Itoa(port),
		ListenAddr:    ":" + strconv.Itoa(port),
		NodeId:        nodeId,
		Revision:      0,
		Graph:         common.NewGraph(),
		MsgCnt:        0,
	}
	// 使用新的函数生成初始坐标
	agent.X, agent.Y, agent.Z = randomCoordinatesInRange(r)
	agent.InitMarkov(r)

	return &agent
}

func (a *Agent) InitMarkov(r float64) {
	markov := common.Markov(r / 100.0)
	Delta := (math.Sqrt(3) / 2) * r
	markov.SetBundary(centerX-Delta, centerX+Delta, centerY-Delta, centerY+Delta, 0, 100)
	markov.Position = [3]float64{a.X, a.Y, a.Z}
	a.Markov = markov
}

func (a *Agent) SetBroadcastList(l []string) {
	a.BroadcastList = l
}

// UpdateCoordinates 修改为接收两个参数：step（步长）和 r（用于限定范围）
// 生成随机微调，并确保更新后坐标仍在规定范围内
func (a *Agent) UpdateCoordinates(step float64, r float64) {
	//生成随机微调
	//deltaX := (rand.Float64()*2 - 1) * step
	//deltaY := (rand.Float64()*2 - 1) * step
	//deltaZ := (rand.Float64()*2 - 1) * step
	//
	//newX := a.X + deltaX
	//newY := a.Y + deltaY
	//newZ := a.Z + deltaZ
	//
	//// 限制 x,y 在 [centerX - (√3/2 * r), centerX + (√3/2 * r)]
	//horizontalDelta := (math.Sqrt(3) / 2) * r
	//if newX < centerX-horizontalDelta {
	//	newX = centerX - horizontalDelta
	//} else if newX > centerX+horizontalDelta {
	//	newX = centerX + horizontalDelta
	//}
	//if newY < centerY-horizontalDelta {
	//	newY = centerY - horizontalDelta
	//} else if newY > centerY+horizontalDelta {
	//	newY = centerY + horizontalDelta
	//}
	//
	//// 限制 z 在 [0, 100]
	//minZ := 0.0
	//maxZ := 100.0
	//if newZ < minZ {
	//	newZ = minZ
	//} else if newZ > maxZ {
	//	newZ = maxZ
	//}
	pos := a.Markov.Update()
	newX := common.RoundFloat(pos[0], 2)
	newY := common.RoundFloat(pos[1], 2)
	newZ := common.RoundFloat(pos[2], 2)
	a.X, a.Y, a.Z = newX, newY, newZ
}

func (a *Agent) generateGossipMessage() common.GossipMessage {
	a.UpdateGraph()
	sendMsg := common.GossipMessage{}
	self := common.NodeMessage{a.NodeId, a.Revision, map[string]string{}, 0}
	nodeInfo := common.GenerateNodeInfo()
	nodeInfo["x"] = fmt.Sprintf("%.2f", a.X)
	nodeInfo["y"] = fmt.Sprintf("%.2f", a.Y)
	nodeInfo["z"] = fmt.Sprintf("%.2f", a.Z)
	self.Data = nodeInfo
	sendMsg.Self = self
	var sendMsgs []common.SendMessage
	// a.Msgs.Range(func(key, value interface{}) bool {
	//     m := value.(HostMsg)
	//     sendMsgs = append(sendMsgs, common.SendMessage{
	//         PrevNode: m.Msg.NodeID,
	//         NodeMsg:  m.Msg,
	//     })
	//     a.Msgs.Delete(key)
	//     return true
	// })
	var avaliblePath []string
	a.Msgs.Range(func(key, value interface{}) bool {
		m := value.(HostMsg)
		paths := m.SendPaths

		for _, p := range paths {
			pn := p[len(p)-1]
			pathKey := common.ListToString(p)
			if common.Contains(avaliblePath, pathKey) {
				nodeMsg := m.Msg
				nodeMsg.TTL = nodeMsg.TTL + 1
				smsg := common.SendMessage{
					PrevNode: pn,
					NodeMsg:  nodeMsg,
				}
				sendMsgs = append(sendMsgs, smsg)
				break
			}

			if a.checkMsgSend(p) {
				avaliblePath = append(avaliblePath, pathKey)
				nodeMsg := m.Msg
				nodeMsg.TTL += 1
				s := common.SendMessage{
					PrevNode: pn,
					NodeMsg:  nodeMsg,
				}
				sendMsgs = append(sendMsgs, s)
				break
			}
		}
		a.Msgs.Delete(key)
		return true
	})
	//for _, n := range a.Graph.FindNeighbor(a.NodeId) {
	//	s := common.SendMessage{
	//		PrevNode: n,
	//		PrevAdj:  a.Graph.FindNeighbor(n),
	//	}
	//	sendMsgs = append(sendMsgs, s)
	//}
	sendMsg.Msgs = sendMsgs
	return sendMsg
}

func (a *Agent) Start(stopCh chan bool, ep int, distance int) {
	addr, err := net.ResolveUDPAddr("udp", a.ListenAddr)
	//TimeOutRev = ep
	if err != nil {
		log.Fatalf("Failed to resolve UDP address: %v", err)
	}
	fmt.Println(a.NodeId, " udpListen: ", addr)
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatalf("%s Failed to listen on UDP: %v", a.NodeId, err)
	}

	// —— 在这里设置 socket 缓冲区大小
	// const bufSize = 2 << 30 // 2 GiB
	// if err := conn.SetReadBuffer(bufSize); err != nil {
	//     log.Printf("Warning: SetReadBuffer(%d) failed: %v\n", bufSize, err)
	// }
	// if err := conn.SetWriteBuffer(bufSize); err != nil {
	//     log.Printf("Warning: SetWriteBuffer(%d) failed: %v\n", bufSize, err)
	// }

	defer func() {
		conn.Close()
	}()

	timeoutStr, exists := os.LookupEnv("TIME_OUT_INTERVAL")
	if exists {
		if seconds, err := strconv.Atoi(timeoutStr); err == nil {
			TimeOutRev = time.Duration(seconds) * time.Second
		}
	}

	go a.HelloBroadCast(stopCh, ep, float64(distance))
	go a.BroadCast(stopCh, ep, float64(distance))
	// 使用 distance 这个参数，其类型为 int
	a.ReceiveMsg(conn, stopCh, distance, ep)
}

func (a *Agent) Greeting() common.GossipMessage {
	var dMsgs []common.SendMessage
	for _, n := range a.Graph.FindNeighbor(a.NodeId) {
		sm := common.SendMessage{
			PrevNode: n,
			PrevAdj:  a.Graph.FindNeighbor(n),
		}
		dMsgs = append(dMsgs, sm)
	}
	nodeInfo := make(map[string]string)
	nodeInfo["x"] = fmt.Sprintf("%.2f", a.X)
	nodeInfo["y"] = fmt.Sprintf("%.2f", a.Y)
	nodeInfo["z"] = fmt.Sprintf("%.2f", a.Z)
	greeting := common.GossipMessage{
		Self: common.NodeMessage{
			NodeID:   a.NodeId,
			Revision: 0,
			Data:     nodeInfo,
		},
		Msgs: dMsgs,
	}
	return greeting
}

func (a *Agent) DoHelloBroadCast(msg common.GossipMessage, ep int) {
	if len(a.BroadcastList) > 0 {
		for _, h := range a.BroadcastList {
			baddr := h + a.ListenAddr
			a.SendMsg(baddr, msg)
		}
	} else {
		baddr := a.BroadcastAddr
		a.SendMsg(baddr, msg)
	}
}

func (a *Agent) DoBroadCast(msg common.GossipMessage, ep int) {
	l := 0
	if a.Revision < ep {
		l++
	}
	for _, m := range msg.Msgs {
		if !common.IsStructEmpty(m.NodeMsg) {
			if m.NodeMsg.Revision < ep {
				l++
			}
		}
	}
	a.MsgCnt = a.MsgCnt + l
	if len(a.BroadcastList) > 0 {
		for _, h := range a.BroadcastList {
			baddr := h + a.ListenAddr
			a.SendMsg(baddr, msg)
		}
	} else {
		baddr := a.BroadcastAddr
		a.SendMsg(baddr, msg)
	}
	fmt.Println(a.NodeId, "Send msg:", msg)
}

func (a *Agent) BroadCast(stopCh chan bool, ep int, r float64) {
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
	fmt.Println(a.NodeId, "BroadCast")

	defer func() {
		fmt.Println(a.NodeId, "Sent Message Count:", a.MsgCnt, "in", a.Revision, "epochs")
		logPath := os.Getenv("LOG_PATH")
		if logPath == "" {
			logPath = "."
		}
		filename := logPath + "/" + a.NodeId
		file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			outStr := fmt.Sprintf("Sent Message Count: %d in %d Epochs", a.MsgCnt, a.Revision)
			file.WriteString(outStr)
		}
	}()
	for {
		select {
		case <-stopCh:
			fmt.Println("Received stop signal, stopping goroutine")
			return
		default:
			if a.Revision == ep+10 {
				fmt.Println("********", a.NodeId, "ran", ep, "epoch finished. ********")
				return
			}

			a.Revision++
			msg := a.generateGossipMessage()
			a.recordMsg(msg)
			a.DoBroadCast(msg, ep)
			time.Sleep(time.Duration(interval) * time.Second)
		}
	}
}

func (a *Agent) HelloBroadCast(stopCh chan bool, ep int, r float64) {
	for {
		select {
		case <-stopCh:
			fmt.Println("Received stop signal, stopping goroutine")
			return
		default:
			if a.Revision == ep+10 {
				fmt.Println("********", a.NodeId, "ran", ep, "epoch finished. ********")
				return
			}
			a.UpdateGraph()
			a.UpdateCoordinates(r/100.0, r)
			msg := a.Greeting()
			a.DoHelloBroadCast(msg, ep)
			time.Sleep(1 * time.Second)
		}
	}
}

func (a *Agent) recordMsg(msg common.GossipMessage) {
	file, err := os.OpenFile("./gossip_logs.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
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

func (a *Agent) isMsgRecorded(nodeID string, revision int) bool {
	file, err := os.Open("./gossip_logs.txt")
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

func (a *Agent) UpdateGraph() {
	a.NodeBuf.Range(func(key, value interface{}) bool {
		n := key.(string)
		t := value.(int64)
		now := time.Now().UnixNano()
		if now-t > TimeOutRev.Nanoseconds() {
			a.Graph.RemoveEdge(a.NodeId, n)
		}
		return true
	})
}

func (a *Agent) WriteMsg(msg common.NodeMessage, ep int) {
	if msg.Revision >= ep || msg.Revision == 0 {
		return
	}
	logPath := os.Getenv("LOG_PATH")
	if logPath == "" {
		logPath = "."
	}
	filename := logPath + "/" + a.NodeId
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

func (a *Agent) SendMsg(baddr string, msg common.GossipMessage) {
	addr, err := net.ResolveUDPAddr("udp", baddr)
	if err != nil {
		fmt.Printf("%s Error resolving address: %v\n", a.NodeId, err)
		return
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		fmt.Printf("%s Error dialing UDP: %v\n", a.NodeId, err)
		return
	}
	bytes, err := json.Marshal(msg)
	defer conn.Close()
	_, err = conn.Write(bytes)
	if err != nil {
		fmt.Printf("%s Error write UDP: %v\n", a.NodeId, err)
	}
}

func (a *Agent) checkMsgSend(p []string) bool {
	if p[0] == a.NodeId {
		return false
	}
	prevNode := p[len(p)-1]
	subGraph := a.Graph.GetSubgraphWithinHops(prevNode, 2)
	allP := append(p, a.NodeId)
	if a.PathExistInMLST(subGraph, allP) {
		return true
	}
	return false
}
