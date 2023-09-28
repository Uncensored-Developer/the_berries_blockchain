package node

import (
	"encoding/json"
	"fmt"
	"io"
	"kryptcoin/database"
	"log"
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

const httpPort = 8081

func Run(dataDir string) error {
	state, err := database.NewStateFromDisk(dataDir)
	if err != nil {
		return err
	}
	defer state.Close()

	http.HandleFunc("/balances/list", func(w http.ResponseWriter, req *http.Request) {
		listBalancesHandler(w, req, state)
	})

	http.HandleFunc("/txn/add", func(w http.ResponseWriter, req *http.Request) {
		txnAddHandler(w, req, state)
	})

	err = http.ListenAndServe(fmt.Sprintf(":%d", httpPort), nil)
	if err != nil {
		return err
	}
	return nil
}

func writeErrorRes(w http.ResponseWriter, err error) {
	jsonErrRes, _ := json.Marshal(ErrorResponse{err.Error()})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	_, err = w.Write(jsonErrRes)
	if err != nil {
		log.Fatalf("Error writing response: %v\n", err)
	}
}

func writeRes(w http.ResponseWriter, content any) {
	contentJson, err := json.Marshal(content)
	if err != nil {
		writeErrorRes(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(contentJson)
	if err != nil {
		log.Fatalf("Error writing response: %v\n", err)
	}
}

func readReq(r *http.Request, reqBody any) error {
	reqBodyJson, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("unable to read request body. %s", err.Error())
	}
	defer r.Body.Close()

	err = json.Unmarshal(reqBodyJson, reqBody)
	if err != nil {
		return fmt.Errorf("unable to unmarshal request body. %s", err.Error())
	}
	return nil
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
