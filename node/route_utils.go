package node

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

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
