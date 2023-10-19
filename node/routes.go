package node

import (
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"kryptcoin/database"
	"kryptcoin/wallet"
	"net/http"
	"strconv"
	"strings"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

type BalancesResponse struct {
	Hash     database.Hash           `json:"block_hash"`
	Balances map[common.Address]uint `json:"balances"`
}

type TxnAddReq struct {
	From string `json:"from"`
	To   string `json:"to"`

	Gas      uint `json:"gas"`
	GasPrice uint `json:"gasPrice"`

	Password string `json:"password"`
	Value    uint   `json:"value"`
	Data     string `json:"data"`
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
	PendingTxns []database.SignedTxn `json:"pending_txns"`
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
	if req.From == "" || req.Password == "" {
		writeErrorRes(w, fmt.Errorf("'from' and 'password' fields are empty"))
		return
	}

	fromAcct := database.NewAccount(req.From)
	nonce := node.state.GetNextAccountNonce(fromAcct)

	txn := database.NewDefaultTxn(
		fromAcct,
		database.NewAccount(req.To),
		req.Value,
		nonce,
		req.Data,
	)
	// Decrypt private key stored in keystore file and sign the txn
	signedTxn, err := wallet.SignWithKeystoreAccount(
		txn,
		fromAcct,
		req.Password,
		wallet.GetKeystoreDirPath(node.dataDir),
	)
	if err != nil {
		writeErrorRes(w, err)
		return
	}

	err = node.AddPendingTxn(signedTxn, node.info)
	if err != nil {
		writeErrorRes(w, err)
		return
	}
	writeRes(w, TxnAddRes{true})
}

func statusHandler(w http.ResponseWriter, r *http.Request, node *Node) {
	res := StatusRes{
		Hash:        node.state.LatestBlockHash(),
		Height:      node.state.LatestBlock().Header.Height,
		KnownPeers:  node.knownPeers,
		PendingTxns: node.getPendingTxnsAsArray(),
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

	peer := NewPeerNode(peerIP, peerPort, false, database.NewAccount(minerRaw), true)
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

func getBlockByHashOrHeight(w http.ResponseWriter, r *http.Request, node *Node) {
	errorRequiredParams := errors.New("height or hash parameter is required")

	params := strings.Split(r.URL.Path, "/")[1:]
	if len(params) < 2 {
		writeErrorRes(w, errorRequiredParams)
	}

	part := strings.TrimSpace(params[1])
	if len(part) == 0 {
		writeErrorRes(w, errorRequiredParams)
		return
	}

	hash := ""
	height, err := strconv.ParseUint(part, 10, 64)
	if err != nil {
		hash = part
	}

	block, err := database.GetBlockByHashOrHeight(node.state, height, hash, node.dataDir)
	if err != nil {
		writeErrorRes(w, errorRequiredParams)
		return
	}

	writeRes(w, block)
}
