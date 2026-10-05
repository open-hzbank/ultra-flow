package db

import (
	"net"
	"os"
	"sync"
)

var (
	hostIp   string
	hostName string
	hostInfo string
	once     sync.Once
)

func initHosts() {
	once.Do(func() {
		hostName = initLocalHostName()
		hostIp = initLocalHostIp()
		hostInfo = hostName + "@" + hostIp
	})
}

// GetLocalHostIp 获取本机 IP
func GetLocalHostIp() string {
	initHosts()
	return hostIp
}

// GetLocalHostInfo 获取本机信息 (hostname@ip)
func GetLocalHostInfo() string {
	initHosts()
	return hostInfo
}

func initLocalHostName() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

func initLocalHostIp() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() && !ipNet.IP.IsMulticast() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String()
			}
		}
	}
	return "127.0.0.1"
}
