package node

import (
	"fmt"
	"kryptcoin/database"
	"net/http"
	"strconv"
	"time"
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
	Hash database.Hash `json:"block_hash"`
}

type StatusRes struct {
	Hash       database.Hash       `json:"block_hash"`
	Height     uint64              `json:"block_height"`
	KnownPeers map[string]PeerNode `json:"known_peers"`
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

func txnAddHandler(w http.ResponseWriter, r *http.Request, state *database.State) {
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

	block := database.NewBlock(
		state.NextBlockHeight(),
		state.LatestBlockHash(),
		uint64(time.Now().Unix()),
		[]database.Txn{txn},
	)
	hash, err := state.AddBlock(block)
	if err != nil {
		writeErrorRes(w, err)
		return
	}
	writeRes(w, TxnAddRes{hash})
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

	peerPort, err := strconv.ParseUint(peerPortRaw, 10, 32)
	if err != nil {
		writeRes(w, AddPeerRes{false, err.Error()})
		return
	}

	peer := NewPeerNode(peerIP, peerPort, false, true)
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
