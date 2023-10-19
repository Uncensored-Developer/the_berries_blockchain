package database

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
)

func GetBlocksAfter(blockHash Hash, dataDir string) ([]Block, error) {
	f, err := os.OpenFile(
		getBlocksDbFilePath(dataDir),
		os.O_RDONLY,
		0600,
	)
	if err != nil {
		return nil, err
	}

	blocks := make([]Block, 0)
	shouldStartCollecting := false

	if reflect.DeepEqual(blockHash, Hash{}) {
		shouldStartCollecting = true
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		//read one line at a time
		if err := scanner.Err(); err != nil {
			return nil, err
		}

		var blockFs BlockFS
		err = json.Unmarshal(scanner.Bytes(), &blockFs)
		if err != nil {
			return nil, err
		}

		if shouldStartCollecting {
			blocks = append(blocks, blockFs.Value)
			continue
		}

		if blockHash == blockFs.Key {
			shouldStartCollecting = true
		}
	}
	return blocks, nil
}

func GetBlockByHashOrHeight(state *State, height uint64, hash, dataDir string) (BlockFS, error) {
	var blockFs BlockFS

	key, ok := state.HeightCache[height]
	if hash != "" {
		key, ok = state.HashCache[hash]
	}

	if !ok {
		if hash != "" {
			return blockFs, fmt.Errorf("invalid hash: %v", hash)
		}
		return blockFs, fmt.Errorf("invalid height: %v", height)
	}

	f, err := os.OpenFile(getBlocksDbFilePath(dataDir), os.O_RDONLY, 0600)
	if err != nil {
		return blockFs, err
	}
	defer f.Close()

	_, err = f.Seek(key, 0)
	if err != nil {
		return blockFs, err
	}

	scanner := bufio.NewScanner(f)
	if scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return blockFs, err
		}

		err = json.Unmarshal(scanner.Bytes(), &blockFs)
		if err != nil {
			return blockFs, err
		}
	}
	return blockFs, nil
}
