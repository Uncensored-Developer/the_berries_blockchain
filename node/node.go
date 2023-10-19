package node

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"kryptcoin/database"
	"log"
	"net/http"
	"time"
)

const (
	DefaultHTTPPort     = 8081
	DefaultIP           = "127.0.0.1"
	DefaultMiner        = "0x0000000000000000000000000000000000000000"
	DefaultBootstrapAcc = "0x0418A658C5874D2Fe181145B685d2e73D761865D"
	DefaultBootstrapIp  = "127.0.0.1"

	pathNodeStatus = "/node/status"
	pathNodeSync   = "/node/sync"

	pathSyncQueryKeyFromBlock = "fromBlock"

	pathAddPeer              = "/node/peer"
	pathAddPeerQueryKeyIP    = "ip"
	pathAddPeerQueryKeyPort  = "port"
	pathAddPeerQueryKeyMiner = "miner"

	miningIntervalSeconds = 10
)

type PeerNode struct {
	IP          string         `json:"ip"`
	Port        uint64         `json:"port"`
	Account     common.Address `json:"account"`
	IsBootstrap bool           `json:"is_bootstrap"`
	connected   bool           // when node already established connection
}

type Node struct {
	dataDir string
	info    PeerNode

	state        *database.State // Main blockchain state after mined Txns have been applied
	pendingState *database.State // temporary pending state to validate new incoming Txns, resets after block is mined

	knownPeers      map[string]PeerNode
	pendingTxns     map[string]database.SignedTxn
	archivedTxns    map[string]database.SignedTxn
	newPendingTxns  chan database.SignedTxn
	newSyncedBlocks chan database.Block
	isMining        bool
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
	if peer.IP == n.info.IP && peer.Port == n.info.Port {
		return true
	}
	_, present := n.knownPeers[peer.TcpAddress()]
	return present
}

func NewNode(dataDir string, ip string, port uint64, bootstrap PeerNode, acct common.Address) *Node {
	// Initialize a new map with only one known peer,
	// the bootstrap node
	knownPeers := make(map[string]PeerNode)
	knownPeers[bootstrap.TcpAddress()] = bootstrap

	return &Node{
		dataDir:         dataDir,
		info:            NewPeerNode(ip, port, false, acct, true),
		knownPeers:      knownPeers,
		pendingTxns:     make(map[string]database.SignedTxn),
		archivedTxns:    make(map[string]database.SignedTxn),
		newSyncedBlocks: make(chan database.Block),
		newPendingTxns:  make(chan database.SignedTxn, 10000),
		isMining:        false,
	}
}

func NewPeerNode(ip string, port uint64, isBootstrap bool, acct common.Address, connected bool) PeerNode {
	return PeerNode{ip, port, acct, isBootstrap, connected}
}

func (n *Node) Run(ctx context.Context) error {
	fmt.Printf("Listening on: %s:%d\n", n.info.IP, n.info.Port)

	state, err := database.NewStateFromDisk(n.dataDir)
	if err != nil {
		return err
	}
	defer state.Close()

	n.state = state
	pendingState := state.Copy()
	n.pendingState = &pendingState

	fmt.Println("Blockchain state:")
	fmt.Printf("	- height: %d\n", n.state.LatestBlock().Header.Height)
	fmt.Printf("	- hash: %s\n", n.state.LatestBlockHash().Hex())

	go n.sync(ctx)
	go n.mine(ctx)

	handler := http.NewServeMux()

	handler.HandleFunc("/balances/list", func(w http.ResponseWriter, req *http.Request) {
		listBalancesHandler(w, req, state)
	})

	handler.HandleFunc("/txn/add", func(w http.ResponseWriter, req *http.Request) {
		txnAddHandler(w, req, n)
	})

	handler.HandleFunc(pathNodeStatus, func(w http.ResponseWriter, req *http.Request) {
		statusHandler(w, req, n)
	})

	handler.HandleFunc(pathAddPeer, func(w http.ResponseWriter, req *http.Request) {
		addPeerHandler(w, req, n)
	})

	handler.HandleFunc(pathNodeSync, func(w http.ResponseWriter, req *http.Request) {
		syncHandler(w, req, n)
	})

	handler.HandleFunc("/block/", func(w http.ResponseWriter, req *http.Request) {
		getBlockByHashOrHeight(w, req, n)
	})

	server := &http.Server{Addr: fmt.Sprintf(":%d", n.info.Port), Handler: handler}

	go func() {
		<-ctx.Done()
		_ = server.Close()
	}()

	err = server.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (n *Node) mine(ctx context.Context) error {
	log.Println("-> Node mine.")
	var miningCtx context.Context
	var stopCurrentMining context.CancelFunc

	ticker := time.NewTicker(time.Second * miningIntervalSeconds)

	for {
		select {
		case <-ticker.C:
			go func() {
				// Wait for new TXNs then start mining
				if len(n.pendingTxns) > 0 && !n.isMining {
					n.isMining = true

					miningCtx, stopCurrentMining = context.WithCancel(ctx)
					err := n.minePendingTxns(miningCtx)
					if err != nil {
						log.Printf("ERROR: %s\n", err)
					}

					n.isMining = false
				}
			}()
		case block, _ := <-n.newSyncedBlocks: // If another node was faster, stop mining
			if n.isMining {
				blockHash, _ := block.Hash()
				log.Printf("\nPeer mined next Block %s faster:\n", blockHash.Hex())

				n.removeMinedPendingTxns(block)
				stopCurrentMining()
			}
		case <-ctx.Done():
			ticker.Stop()
			return nil
		}
	}
}

func (n *Node) minePendingTxns(ctx context.Context) error {
	blockToMine := NewPendingBlock(
		n.state.LatestBlockHash(),
		n.state.NextBlockHeight(),
		n.info.Account, // Potential block miner
		n.getPendingTxnsAsArray(),
	)

	minedBlock, err := Mine(ctx, blockToMine)
	if err != nil {
		return err
	}
	n.removeMinedPendingTxns(minedBlock)

	err = n.addBlock(minedBlock)
	if err != nil {
		return err
	}

	return nil
}

func (n *Node) removeMinedPendingTxns(block database.Block) {
	if len(block.Txns) > 0 && len(n.pendingTxns) > 0 {
		log.Printf("Updating in-memory pending TXNs pool")
	}

	for _, txn := range block.Txns {
		txnHash, _ := txn.Hash()
		if _, exists := n.pendingTxns[txnHash.Hex()]; exists {
			log.Printf("\t archiving mined TXN: %s\n", txnHash.Hex())

			n.archivedTxns[txnHash.Hex()] = txn
			delete(n.pendingTxns, txnHash.Hex())
		}
	}
}

func (n *Node) getPendingTxnsAsArray() []database.SignedTxn {
	txns := make([]database.SignedTxn, len(n.pendingTxns))
	i := 0
	for _, txn := range n.pendingTxns {
		txns[i] = txn
		i++
	}
	return txns
}

func (n *Node) AddPendingTxn(txn database.SignedTxn, peer PeerNode) error {
	txnHash, err := txn.Hash()
	if err != nil {
		return err
	}

	txnJson, err := json.Marshal(txn)
	if err != nil {
		return err
	}

	err = database.ApplyTxn(txn, n.pendingState)
	if err != nil {
		return err
	}

	_, isPending := n.pendingTxns[txnHash.Hex()]
	_, isArchived := n.archivedTxns[txnHash.Hex()]

	if !isPending && !isArchived {
		n.pendingTxns[txnHash.Hex()] = txn
		n.newPendingTxns <- txn
		log.Printf("Added pending TXN %s from Peer %s\n", txnJson, peer.TcpAddress())
	}
	return nil
}

func (n *Node) syncPendingTxns(peer PeerNode, txns []database.SignedTxn) error {
	for _, txn := range txns {
		err := n.AddPendingTxn(txn, peer)
		if err != nil {
			return err
		}
	}
	return nil
}

func (n *Node) addBlock(block database.Block) error {
	_, err := n.state.AddBlock(block)
	if err != nil {
		return err
	}

	// Reset pending state
	pendingState := n.state.Copy()
	n.pendingState = &pendingState

	return nil
}
