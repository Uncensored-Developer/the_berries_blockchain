package node

import (
	"kryptcoin/database"
	"net/http"
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
	Hash   database.Hash `json:"block_hash"`
	Height uint64        `json:"block_height"`
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
	err = state.AddTxn(txn)
	if err != nil {
		writeErrorRes(w, err)
		return
	}

	hash, err := state.Persist()
	if err != nil {
		writeErrorRes(w, err)
		return
	}
	writeRes(w, TxnAddRes{hash})
}

func statusHandler(w http.ResponseWriter, r *http.Request, state *database.State) {
	res := StatusRes{
		Hash:   state.LatestBlockHash(),
		Height: state.LatestBlock().Header.Height,
	}
	writeRes(w, res)
}
