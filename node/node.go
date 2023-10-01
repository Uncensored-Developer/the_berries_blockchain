package node

import (
	"context"
	"fmt"
	"kryptcoin/database"
	"net/http"
)

const (
	DefaultHTTPPort = 8081
	DefaultIP       = "127.0.0.1"

	pathNodeStatus = "/node/status"
	pathNodeSync   = "/node/sync"

	pathSyncQueryKeyFromBlock = "fromBlock"

	pathAddPeer             = "/node/peer"
	pathAddPeerQueryKeyIP   = "ip"
	pathAddPeerQueryKeyPort = "port"
)

type PeerNode struct {
	IP          string `json:"ip"`
	Port        uint64 `json:"port"`
	IsBootstrap bool   `json:"is_bootstrap"`
	connected   bool   // when node already established connection
}

type Node struct {
	dataDir string
	ip      string
	port    uint64

	state      *database.State // To inject the State into the HTTP handlers
	knownPeers map[string]PeerNode
}

func (pn PeerNode) TcpAddress() string {
	return fmt.Sprintf("%s:%d", pn.IP, pn.Port)
}

func (n *Node) AddPeer(peer PeerNode) {
	n.knownPeers[peer.TcpAddress()] = peer
}

func (n *Node) RemovePeer(peer PeerNode) {
	delete(n.knownPeers, peer.TcpAddress())
}

func (n *Node) IsKnownPeer(peer PeerNode) bool {
	if peer.IP == n.ip && peer.Port == n.port {
		return true
	}
	_, present := n.knownPeers[peer.TcpAddress()]
	return present
}

func NewNode(dataDir string, ip string, port uint64, bootstrap PeerNode) *Node {
	// Initialize a new map with only one known peer,
	// the bootstrap node
	knownPeers := make(map[string]PeerNode)
	knownPeers[bootstrap.TcpAddress()] = bootstrap

	return &Node{
		dataDir:    dataDir,
		ip:         ip,
		port:       port,
		knownPeers: knownPeers,
	}
}

func NewPeerNode(ip string, port uint64, isBootstrap bool, connected bool) PeerNode {
	return PeerNode{ip, port, isBootstrap, connected}
}

func (n *Node) Run() error {
	ctx := context.Background()
	fmt.Printf("Listening on: %s:%d", n.ip, n.port)

	state, err := database.NewStateFromDisk(n.dataDir)
	if err != nil {
		return err
	}
	defer state.Close()

	n.state = state

	go n.sync(ctx)

	http.HandleFunc("/balances/list", func(w http.ResponseWriter, req *http.Request) {
		listBalancesHandler(w, req, state)
	})

	http.HandleFunc("/txn/add", func(w http.ResponseWriter, req *http.Request) {
		txnAddHandler(w, req, state)
	})

	http.HandleFunc(pathNodeStatus, func(w http.ResponseWriter, req *http.Request) {
		statusHandler(w, req, n)
	})

	http.HandleFunc(pathAddPeer, func(w http.ResponseWriter, req *http.Request) {
		addPeerHandler(w, req, n)
	})

	http.HandleFunc(pathNodeSync, func(w http.ResponseWriter, req *http.Request) {
		syncHandler(w, req, n)
	})

	err = http.ListenAndServe(fmt.Sprintf(":%d", n.port), nil)
	if err != nil {
		return err
	}
	return nil
}
