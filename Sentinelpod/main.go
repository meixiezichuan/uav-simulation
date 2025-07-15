package main

import (
    "fmt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/meixiezichuan/broadcast-gossip/sentinel"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"strconv"
	//"sync"
    "strings"
	"syscall"
)

// getLocalIP 为后备方案，返回一个默认节点标识
func getLocalIP() string {
    i := rand.Intn(100)
    ip := "sentinel-node" + strconv.Itoa(i)
    addrs, err := net.InterfaceAddrs()
    if err != nil {
        fmt.Println("Get Interface Error:", err)
        return ip
    }
    for _, addr := range addrs {
        if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
            if ipNet.IP.To4() != nil {
                ip = ipNet.IP.String()
                fmt.Println("Local IP address:", ip)
                return ip
            }
        }
    }
    return ip
}

// runSentinel 初始化并启动单个哨兵实例
func runSentinel(node string, port int, distance int, ep int) {
    // 创建信号通道以监听退出信号
    sigs := make(chan os.Signal, 1)
    signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
    done := make(chan bool, 1)
    go func() {
        sig := <-sigs
        fmt.Println("Get sig", sig, "Exiting …")
        done <- true
    }()

    // 从 POD_NAME 中解析序号，用于计算 z 坐标
    podName := os.Getenv("POD_NAME")
    z := 0.0
    if podName != "" {
        parts := strings.Split(podName, "-")
        if len(parts) > 1 {
            if ordinal, err := strconv.Atoi(parts[len(parts)-1]); err == nil {
                // 使用传入的 distance 作为步长，计算 z 坐标
                z = float64(ordinal) * float64(distance)
            }
        }
    }
    // 如果没有从 POD_NAME 解析到，则默认 z 为 0

    // 初始化哨兵实例，固定 x,y 为 50,50；z为上述计算值
    sentinelAgent := sentinel.InitSentinel(node, port, z)
    fmt.Printf("%s Started Sentinel with coordinate: (%.2f, %.2f, %.2f)\n",
        sentinelAgent.NodeId, sentinelAgent.X, sentinelAgent.Y, sentinelAgent.Z)

    // 启动哨兵实例（接收消息，首次收到消息后启动周期广播）
    sentinelAgent.Start(done, ep, distance)
    fmt.Println("All sentinel tasks completed.")
}

func main() {
    // 默认广播轮次 ep 和 distance（通信步长、最大跳距）参数
    ep := 100
    distance := 10
    if len(os.Args) > 1 {
        if val, err := strconv.Atoi(os.Args[1]); err == nil {
            ep = val
        }
    }
    if len(os.Args) > 2 {
        if val, err := strconv.Atoi(os.Args[2]); err == nil {
            distance = val
        }
    }

    // 获取节点标识：优先从 HOSTNAME 环境变量获取
    node, exist := os.LookupEnv("HOSTNAME")
    if !exist || node == "" {
        node = getLocalIP()
    }

    // 读取端口号（默认为9898），可通过环境变量 BroadcastPort 传入
    port := 9898
    if strPort, exist := os.LookupEnv("BroadcastPort"); exist {
        if p, err := strconv.Atoi(strPort); err == nil {
            port = p
        }
    }

    // 运行哨兵
    runSentinel(node, port, distance, ep)
}
