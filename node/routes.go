package node

import (
	"fmt"
	"kryptcoin/database"
	"net/http"
	"strconv"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

type BalancesResponse struct {
	Hash     database.Hash             `json:"block_hash"`
	Balances map[database.Account]uint `json:"balances"`
}

type TxnAddReq struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Value uint   `json:"value"`
	Data  string `json:"data"`
}

type TxnAddRes struct {
	// Return confirmation not block hash because
	// the mining takes sometimes several minutes
	// and the TXNs should be distributed to all nodes
	// so everyone has equal chance of mining the block
	Success bool `json:"success"`
}

type StatusRes struct {
	Hash       database.Hash       `json:"block_hash"`
	Height     uint64              `json:"block_height"`
	KnownPeers map[string]PeerNode `json:"known_peers"`

	// Exchange pending TXNs as part of the periodic Sync() interval
	PendingTxns []database.Txn `json:"pending_txns"`
}

type SyncRes struct {
	Blocks []database.Block `json:"blocks"`
}

type AddPeerRes struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

func listBalancesHandler(w http.ResponseWriter, r *http.Request, state *database.State) {
	writeRes(w, BalancesResponse{state.LatestBlockHash(), state.Balances})
}

func txnAddHandler(w http.ResponseWriter, r *http.Request, node *Node) {
	req := TxnAddReq{}
	err := readReq(r, &req)
	if err != nil {
		writeErrorRes(w, err)
		return
	}

	txn := database.NewTxn(
		database.NewAccount(req.From),
		database.NewAccount(req.To),
		req.Value,
		req.Data,
	)

	err = node.AddPendingTxn(txn, node.info)
	if err != nil {
		writeErrorRes(w, err)
		return
	}
	writeRes(w, TxnAddRes{true})
}

func statusHandler(w http.ResponseWriter, r *http.Request, node *Node) {
	res := StatusRes{
		Hash:       node.state.LatestBlockHash(),
		Height:     node.state.LatestBlock().Header.Height,
		KnownPeers: node.knownPeers,
	}
	writeRes(w, res)
}

func addPeerHandler(w http.ResponseWriter, r *http.Request, node *Node) {
	peerIP := r.URL.Query().Get(pathAddPeerQueryKeyIP)
	peerPortRaw := r.URL.Query().Get(pathAddPeerQueryKeyPort)
	minerRaw := r.URL.Query().Get(pathAddPeerQueryKeyMiner)

	peerPort, err := strconv.ParseUint(peerPortRaw, 10, 32)
	if err != nil {
		writeRes(w, AddPeerRes{false, err.Error()})
		return
	}

	peer := NewPeerNode(peerIP, peerPort, false, database.Account(minerRaw), true)
	node.AddPeer(peer)

	fmt.Printf("Peer %s was added into KnownPeers\n", peer.TcpAddress())
	writeRes(w, AddPeerRes{true, ""})
}

func syncHandler(w http.ResponseWriter, r *http.Request, node *Node) {
	reqHash := r.URL.Query().Get(pathSyncQueryKeyFromBlock)

	hash := database.Hash{}
	err := hash.UnmarshalText([]byte(reqHash))
	if err != nil {
		writeErrorRes(w, err)
		return
	}

	blocks, err := database.GetBlocksAfter(hash, node.dataDir)
	if err != nil {
		writeErrorRes(w, err)
		return
	}
	writeRes(w, SyncRes{Blocks: blocks})
}
