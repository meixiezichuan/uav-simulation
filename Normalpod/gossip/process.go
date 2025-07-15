package gossip

import (
	"encoding/json"
	"fmt"
	"github.com/meixiezichuan/broadcast-gossip/common"
	"log"
	"math"
	"net"
	"time"
)

func (a *Agent) ReceiveMsg(conn *net.UDPConn, stopCh <-chan bool, distance int, ep int) {
	//fmt.Println(a.NodeId, " receive msg ")
	buf := make([]byte, 6553500)
	for {
		select {
		case <-stopCh:
			fmt.Println(a.NodeId, "Received stop signal, stopping goroutine")
			return
		default:
			n, _, err := conn.ReadFromUDP(buf)
			if err != nil {
				//log.Printf("%s Failed to read UDP message: %v", a.NodeId, err)
				continue
			}
			//fmt.Println(a.NodeId, " receive msg n: ", n)
			var msg common.GossipMessage
			if err := json.Unmarshal(buf[:n], &msg); err != nil {
				log.Printf("Failed to unmarshal message: %v", err)
				continue
			}
			a.HandleMsg(msg, distance, ep)
		}
	}
}

func (a *Agent) HandleMsg(msg common.GossipMessage, distance int, ep int) {
	//1. first get network topo
	// get direct node msg
	dmsg := msg.Self
	if dmsg.NodeID == a.NodeId {
		return
	}

	// parentIP := common.Ip2int(net.ParseIP(dmsg.NodeID))
	// localIP := common.Ip2int(net.ParseIP(a.NodeId))
	// mdis := int64(parentIP) - int64(localIP)
	//fmt.Printf("localIP: %s, %d , parentIP: %s, %d， distance: %d \n", a.NodeId, localIP, dmsg.NodeID, parentIP, mdis)
	// if mdis < int64(-1*distance) || mdis > int64(distance) {
	// 	return
	// }
	//lastDot := strings.LastIndex(dmsg.NodeID, ".")
	//lastPart := dmsg.NodeID[lastDot+1:]
	//receiveNum, _ := strconv.Atoi(lastPart)
	//
	//lastDot = strings.LastIndex(a.NodeId, ".")
	//lastPart = a.NodeId[lastDot+1:]
	//num, _ := strconv.Atoi(lastPart)
	//
	//distance1 := (receiveNum - num + 255) % 255
	//distance2 := (num - receiveNum + 255) % 255
	//if distance1 > distance && distance2 > distance {
	//	return
	//}

	//解析坐标信息
	var senderX, senderY, senderZ float64
	if dmsg.Data != nil {
		if xStr, ok := dmsg.Data["x"]; ok {
			fmt.Sscanf(xStr, "%f", &senderX)
		}
		if yStr, ok := dmsg.Data["y"]; ok {
			fmt.Sscanf(yStr, "%f", &senderY)
		}
		if zStr, ok := dmsg.Data["z"]; ok {
			fmt.Sscanf(zStr, "%f", &senderZ)
		}
	}

	//计算发送方与本地的欧几里得距离
	dx := a.X - senderX
	dy := a.Y - senderY
	dz := a.Z - senderZ
	dist := math.Sqrt(dx*dx + dy*dy + dz*dz)

	//如果距离超过设定阈值，则丢弃消息
	if dist > float64(distance) {
		//fmt.Printf("%s: Discarding message from %s, distance %.2f exceeds threshold %d\n",
		//	a.NodeId, dmsg.NodeID, dist, distance)
		return
	}

	//fmt.Println(a.NodeId, "handle ", msg)
	// 加入一跳桶
	a.WriteMsg(dmsg, ep)
	a.Graph.AddEdge(a.NodeId, dmsg.NodeID)

	now := time.Now().UnixNano()
	a.NodeBuf.Store(dmsg.NodeID, now)

	//rev, exist := func() (int, bool) {
	//	value, ok := a.NodeBuf.Load(dmsg.NodeID)
	//	if ok {
	//		return value.(int), true
	//	}
	//	return 0, false
	//}()

	//if exist {
	//	if rev < dmsg.Revision {
	//		a.NodeBuf.Store(dmsg.NodeID, dmsg.Revision)
	//	}
	//} else {
	//	// first time received dmsg
	//	a.NodeBuf.Store(dmsg.NodeID, dmsg.Revision)
	//}

	// add msg
	path := Path{dmsg.NodeID}
	a.UpdateMsgs(dmsg, path, ep)

	// handle other msg
	fmt.Println("receive msg:", msg)
	for _, m := range msg.Msgs {
		// handle msg
		if !common.IsStructEmpty(m.NodeMsg) {
			a.WriteMsg(m.NodeMsg, ep)
			path = Path{m.PrevNode, dmsg.NodeID}
			if m.NodeMsg.NodeID != a.NodeId {
				a.UpdateMsgs(m.NodeMsg, path, ep)
			}
		}
		// add PrevAdj
		a.Graph.AddEdge(dmsg.NodeID, m.PrevNode)
		for _, pn := range m.PrevAdj {
			a.Graph.AddEdge(m.PrevNode, pn)
		}
	}
}

// 处理接收到的Gossip消息
func (a *Agent) PathExistInMLST(g *common.Graph, p Path) bool {

	preNode := p[0]
	mlst, _ := g.MLST10(preNode)
	//fmt.Println(a.NodeId, " root: ", preNode, " path: ", p, " mlst: ")
	//mlst.Display()
	// if node is leaf, return false
	if mlst.IsLeaf(a.NodeId) {
		return false
	}

	if mlst.PathExists(p) {
		return true
	}
	return false
}

func (a *Agent) UpdateMsgs(msg common.NodeMessage, path Path, ep int) {
	if msg.Revision >= ep || msg.Revision == 0 {
		return
	}
	if a.isMsgRecorded(msg.NodeID, msg.Revision) {
		return
	}
	key := fmt.Sprintf("%s_%d", msg.NodeID, msg.Revision)
	value, exist := a.Msgs.Load(key)
	Hm := HostMsg{
		Msg: msg,
	}

	if exist {
		existingHostMsg := value.(HostMsg)
		sendpath := append(existingHostMsg.SendPaths, path)
		Hm.SendPaths = sendpath
	} else {
		Hm.SendPaths = []Path{path}
	}

	a.Msgs.Store(key, Hm)
}
