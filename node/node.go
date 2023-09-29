package node

import (
	"fmt"
	"kryptcoin/database"
	"net/http"
)

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

	http.HandleFunc("/node/status", func(w http.ResponseWriter, req *http.Request) {
		statusHandler(w, req, state)
	})

	err = http.ListenAndServe(fmt.Sprintf(":%d", httpPort), nil)
	if err != nil {
		return err
	}
	return nil
}
