package discover

import (
	"time"
	"log"
	"encoding/json"
	"net/http"
	"io/ioutil"
	"strings"
	"bytes"
	"fmt"
	"errors"

	"../common"

	"github.com/coreos/etcd/client"
	"github.com/coreos/etcd/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/gorilla/mux"
)

// node report status
// can't be modified
const (
	CREATE int = iota
	UP
	IMMATURE
	DOWN
	DELETE
)

// for etcd monitoring
// servNodes: node info, tenantId: tenant info,
// peerAddr: url of peer, keysAPI: etcd controller
type Monitor struct {
	servNodes	map[string]*Node
	tenantId	string
	peerAddr	string
	keysAPI		client.KeysAPI
}

// node info get from etcd
type Node struct {
	ip 			string
	service		string
	rptStatus	int
	IsReported	bool
}

// struct in json
type JMonitorCfg struct {
	Port		string	`json:"port"`
	TenantId	string	`json:"tenant-id"`
}

type JTenantInfo struct {
	TenantId	string	`json:"tenant-id"`
}

type JNode struct {
	IP		string	`json:"ip"`
	Status	string	`json:"status"`
}

type JNodesInfo struct {
	ServName	string	`json:"serv-name"`
	NodeList	[]JNode	`json:"nodes"`
}

type JServNodesInfo struct {
	TenantId	string			`json:"tenant-id"`
	Services	[]JNodesInfo	`json:"services"`
}

//
func NewMonitor(endpoints []string) *Monitor {
	cfg := client.Config{
		Endpoints:					endpoints,
		Transport:					client.DefaultTransport,
		HeaderTimeoutPerRequest:	time.Second,
	}

	etcdClient, err := client.New(cfg)
	if err != nil {
		log.Println("Failed to connect to etcd: ", err)
		return nil
	}

	monitor := &Monitor {
		servNodes:		make(map[string]*Node),
		keysAPI:		client.NewKeysAPI(etcdClient),
	}

	return monitor
}

func (m *Monitor) AddNode(key string, info *common.NodeInfo) {
	node := &Node{
		ip:			info.IP,
		service:	info.Service,
		rptStatus:	IMMATURE,
		IsReported:	false,
	}
	if info.Status == "UP" {
		node.rptStatus = CREATE
	}
	m.servNodes[key] = node
}

func (m *Monitor) UpdateNode(key string, info *common.NodeInfo) {
	node := m.servNodes[key]
	if info.Status == "UP" && node.rptStatus >= IMMATURE {
		if node.rptStatus == IMMATURE {
			node.rptStatus = CREATE
		}else {
			node.rptStatus = UP
		}
		node.IsReported = false
	}
	if info.Status == "DOWN" && node.rptStatus <= UP {
		node.rptStatus = DOWN
		node.IsReported = false
	}
}

func (m *Monitor) DeleteNode(key string) {
	if node, ok := m.servNodes[key]; ok {
		if node.rptStatus <= UP {
			node.IsReported = false
		}
		node.rptStatus = DELETE
	}
}

func (m *Monitor) NodeExpire(key string) {
	if node, ok := m.servNodes[key]; ok {
		if node.rptStatus == UP {
			node.rptStatus = DOWN
		}else if node.rptStatus == CREATE {
			if node.IsReported {
				node.rptStatus = DOWN
			}else {
				node.rptStatus = IMMATURE
			}
		}else {
			return
		}
		node.IsReported = false
	}
}

func (m *Monitor) assembleRptNode(n *Node) (*JServNodesInfo) {
	stat := NodeStatusConvert(n)
	node := JNode {
		IP:		n.ip,
		Status:	stat,
	}

	serv := JNodesInfo {
		ServName:	n.service,
	}
	serv.NodeList = append(serv.NodeList, node)

	servNodes := new(JServNodesInfo)
	servNodes.TenantId = m.tenantId
	servNodes.Services = append(servNodes.Services, serv)

	return servNodes
}

func (m *Monitor) reportNode(n *Node) error {
    client := &http.Client{}
	servNodes := m.assembleRptNode(n)
	j, _ := json.Marshal(*servNodes)
    fmt.Println(*servNodes)
	request, _ := http.NewRequest("POST", m.peerAddr, bytes.NewBuffer(j))
	request.Header.Set("Content-Type", "application/json")
	res, err := client.Do(request)
	if err != nil {
		log.Print("Error in posting node info: ", err)
		return errors.New("Failed to connect to server!")
	}
	defer res.Body.Close()
	return nil
}

func (m *Monitor) ReportStatus(key string) {
	if node, ok := m.servNodes[key]; ok {
		if node.IsReported || m.peerAddr == "" {
			return
		}
		err := m.reportNode(node)
		if err == nil {
			node.IsReported = true
		}
	}
}


func (m *Monitor) WatchNodes() {
	api := m.keysAPI
	watcher := api.Watcher("/service", &client.WatcherOptions{
		Recursive:	true,
	})
	for {
		res, err := watcher.Next(context.Background())
		if err != nil {
			log.Println("Error watch nodes: ", err)
			break
		}
		if res.Action == "expire" {
			m.NodeExpire(res.Node.Key)
		}else if res.Action == "set" || res.Action == "update"{
			info := &common.NodeInfo{}
			err := json.Unmarshal([]byte(res.Node.Value), info)
			if err != nil {
				log.Print(err)
			}
			if _, ok := m.servNodes[res.Node.Key]; ok {
				m.UpdateNode(res.Node.Key, info)
			}else {
				m.AddNode(res.Node.Key, info)
			}
		}else if res.Action == "delete" {
			m.DeleteNode(res.Node.Key)
		}else {
			log.Println("Unreachable!")
		}
		m.ReportStatus(res.Node.Key)
	}
}


func (m *Monitor) PostMonitorCfgHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var t JMonitorCfg
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
	}else {
		json.Unmarshal(b, &t)
		m.tenantId = t.TenantId
		m.peerAddr = "http://" + strings.Split(r.RemoteAddr, ":")[0] + ":" + t.Port + "/api/v1.0/monitor"
		fmt.Println(m.peerAddr)
		w.Write(b)
		w.WriteHeader(http.StatusOK)
	}
}

func NodeStatusConvert(n *Node) string {
	if n.rptStatus <= UP {
		return "UP"
	}else {
		return "DOWN"
	}
}


func (m *Monitor) GetNodesByNames(names []string)(*JServNodesInfo) {
	res := new(JServNodesInfo)
	res.TenantId = m.tenantId
	for _, n := range names {
		service := JNodesInfo{}
		service.ServName = n
		for _, v := range m.servNodes {
			if v.service == n {
				stat := NodeStatusConvert(v)
				var node = JNode {
						 IP:		v.ip,
						 Status:	stat,
				}
				service.NodeList = append(service.NodeList, node)
			}
		}
		res.Services = append(res.Services, service)
	}
	return res
}

func (m *Monitor) GetServInfoHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	servName := vars["serv_name"]
	servNodes := m.GetNodesByNames([]string{servName})
	j, _ := json.Marshal(*servNodes)
	w.Write(j)
	w.WriteHeader(http.StatusOK)
}
