package common

import (
	"encoding/binary"
	"golang.org/x/exp/rand"
	"math"
	"net"
	"reflect"
	"strconv"
	"strings"
)

func IsStructEmpty(s interface{}) bool {
	// 判断是否是零值
	return reflect.DeepEqual(s, reflect.Zero(reflect.TypeOf(s)).Interface())
}

func GenerateNodeInfo() map[string]string {
	cnum := rand.Intn(100)
	cpu := "%" + strconv.Itoa(cnum)

	mnum := rand.Intn(100)
	mem := strconv.Itoa(mnum) + "MB"

	bnum := rand.Intn(100)
	ba := "%" + strconv.Itoa(bnum)

	return map[string]string{
		"Cpu":     cpu,
		"Mem":     mem,
		"Battery": ba,
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Utility function to check if a slice contains a value
func Contains(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

func Ip2int(ip net.IP) uint32 {
	return binary.BigEndian.Uint32(ip.To4())
}

func ListToString(list []string) string {
	return strings.Join(list, "_")
}

func StringToList(s string) []string {
	// 处理空字符串情况
	if s == "" {
		return []string{}
	}
	return strings.Split(s, "_")
}

func RoundFloat(val float64, precision int) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}
