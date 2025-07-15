package sentinel

import (
	"encoding/json"
	"fmt"
	"github.com/meixiezichuan/broadcast-gossip/common"
	"log"
	"math"
	"time"

	//"time"
	"net"
)

// 接收消息函数，哨兵启动后进入接收状态，首次收到消息后启动周期广播（不主动广播）
func (s *Sentinel) ReceiveMsg(conn *net.UDPConn, stopCh <-chan bool, distance int, ep int) {
	//fmt.Println(s.NodeId, "receive msg")
	buf := make([]byte, 6553500)
	for {
		select {
		case <-stopCh:
			fmt.Println(s.NodeId, "Received stop signal, stopping message receive loop")
			return
		default:
			n, _, err := conn.ReadFromUDP(buf)
			if err != nil {
				continue
			}
			//fmt.Println(s.NodeId, "received msg n:", n)
			var msg common.GossipMessage
			if err := json.Unmarshal(buf[:n], &msg); err != nil {
				log.Printf("Failed to unmarshal message: %v\n", err)
				continue
			}
			s.HandleMsg(msg, distance, ep)
		}
	}
}

func (s *Sentinel) HandleMsg(msg common.GossipMessage, distance int, ep int) {
	dmsg := msg.Self
	if dmsg.NodeID == s.NodeId {
		return
	}
	// 解析发送方坐标信息
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
	// 计算欧几里得距离
	dx := s.X - senderX
	dy := s.Y - senderY
	dz := s.Z - senderZ
	dist := math.Sqrt(dx*dx + dy*dy + dz*dz)
	// 如果距离超过设定阈值，则丢弃该消息
	if dist > float64(distance) {
		fmt.Printf("%s: Discarding message from %s, distance %.2f exceeds threshold %d\n",
			s.NodeId, dmsg.NodeID, dist, distance)
		return
	}
	fmt.Println(s.NodeId, "handle msg: ", msg)
	// 记录日志、更新图与状态
	s.WriteMsg(dmsg, ep)
	s.Graph.AddEdge(s.NodeId, dmsg.NodeID)
	now := time.Now().UnixNano()
	s.NodeBuf.Store(dmsg.NodeID, now)
	//rev, exist := func() (int, bool) {
	//	value, ok := s.NodeBuf.Load(dmsg.NodeID)
	//	if ok {
	//		return value.(int), true
	//	}
	//	return 0, false
	//}()
	//if exist {
	//	if rev < dmsg.Revision {
	//		s.NodeBuf.Store(dmsg.NodeID, dmsg.Revision)
	//	}
	//} else {
	//	s.NodeBuf.Store(dmsg.NodeID, dmsg.Revision)
	//}
	path := Path{dmsg.NodeID}
	s.UpdateMsgs(dmsg, path, ep)
	// 处理附加消息
	for _, m := range msg.Msgs {
		if !common.IsStructEmpty(m.NodeMsg) {
			s.WriteMsg(m.NodeMsg, ep)
			path = Path{m.PrevNode, dmsg.NodeID}
			if m.NodeMsg.NodeID != s.NodeId {
				s.UpdateMsgs(m.NodeMsg, path, ep)
			}
		} else {
			s.Graph.AddEdge(dmsg.NodeID, m.PrevNode)
			for _, pn := range m.PrevAdj {
				s.Graph.AddEdge(m.PrevNode, pn)
			}
		}
	}
}

func (s *Sentinel) PathExistInMLST(g *common.Graph, p Path) bool {
	preNode := p[0]
	mlst, _ := g.MLST10(preNode)
	if mlst.IsLeaf(s.NodeId) {
		return false
	}
	return mlst.PathExists(p)
}

func (s *Sentinel) UpdateMsgs(msg common.NodeMessage, path Path, ep int) {
	if msg.Revision >= ep || msg.Revision == 0 {
		return
	}
	if s.isMsgRecorded(msg.NodeID, msg.Revision) {
		return
	}
	key := fmt.Sprintf("%s_%d", msg.NodeID, msg.Revision)
	value, exist := s.Msgs.Load(key)
	Hm := HostMsg{Msg: msg}
	if exist {
		existingHostMsg := value.(HostMsg)
		sendpath := append(existingHostMsg.SendPaths, path)
		Hm.SendPaths = sendpath
	} else {
		Hm.SendPaths = []Path{path}
	}
	s.Msgs.Store(key, Hm)
}
