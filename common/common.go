package common

import (
	"net"
	"log"

	"github.com/robfig/config"
)

// node info register into etcd
type NodeInfo struct {
	IP			string	`json:"ip"`
	Service		string	`json:"service"`
	Status		string	`json:"status"`
}

// Query ip info by interface name, e.g. eth0
// ip in form of xx.xx.xx.xx
func GetIPv4ByIFName(name string) (ip string) {
	intf, err := net.InterfaceByName(name)
	if err != nil {
		log.Fatal("Error in getting interface by name: ", err)
	}

	addrs, err := intf.Addrs()
	if err != nil {
		log.Fatal("Error in getting addrs of interface: ", err)
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok {
			if ipnet.IP.To4() != nil {
				ip = ipnet.IP.String()
			}
		}
	}
	return ip
}

// Get sections defined in config file, except for "DEFAULT"
func GetServFromConf(fn string) []string {
	cfg, err := config.ReadDefault(fn)
	if err != nil {
		log.Fatal("Error in get configuration: ", err)
	}
	return cfg.Sections()[1:]
}



