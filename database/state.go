package database

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
)

type State struct {
	Balances        map[Account]uint
	txnMempool      []Txn
	dbFile          *os.File
	latestBlock     Block
	latestBlockHash Hash
	hasGenesisBlock bool
}

func (s *State) LatestBlockHash() Hash {
	return s.latestBlockHash
}

func (s *State) LatestBlock() Block {
	return s.latestBlock
}

func (s *State) NextBlockHeight() uint64 {
	if !s.hasGenesisBlock {
		return uint64(0)
	}
	return s.latestBlock.Header.Height + 1
}

func NewStateFromDisk(dataDir string) (*State, error) {
	err := initDataDirIfNotExists(dataDir)
	if err != nil {
		return nil, err
	}

	genesis, err := loadGenesis(getGenesisJsonFilePath(dataDir))
	if err != nil {
		return nil, err
	}

	balances := make(map[Account]uint)
	for account, balance := range genesis.Balances {
		balances[account] = balance
	}

	f, err := os.OpenFile(getBlocksDbFilePath(dataDir), os.O_APPEND|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(f)
	state := &State{balances, make([]Txn, 0), f, Block{}, Hash{}, false}

	//loop over each of the txn line in the txn.db file
	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, err
		}

		blockFsJson := scanner.Bytes()
		var blockFs BlockFS
		err = json.Unmarshal(blockFsJson, &blockFs)
		if err != nil {
			return nil, err
		}

		err = applyTxns(blockFs.Value.Txns, state)
		if err != nil {
			return nil, err
		}

		state.latestBlockHash = blockFs.Key
		state.latestBlock = blockFs.Value
		state.hasGenesisBlock = true
	}
	return state, nil
}

func (s *State) apply(txn Txn) error {
	if txn.IsReward() {
		s.Balances[txn.To] += txn.Value
		return nil
	}

	if txn.Value > s.Balances[txn.From] {
		return fmt.Errorf("account %s has insufficient balance for %d", txn.From, txn.Value)
	}

	s.Balances[txn.From] -= txn.Value
	s.Balances[txn.To] += txn.Value

	return nil
}

func (s *State) copy() State {
	c := State{}
	c.hasGenesisBlock = s.hasGenesisBlock
	c.latestBlock = s.latestBlock
	c.latestBlockHash = s.latestBlockHash
	c.txnMempool = make([]Txn, len(s.txnMempool))
	c.Balances = make(map[Account]uint)

	for acct, balance := range s.Balances {
		c.Balances[acct] = balance
	}

	for _, txn := range s.txnMempool {
		c.txnMempool = append(c.txnMempool, txn)
	}

	return c
}

func (s *State) AddTxn(txn Txn) error {
	if err := s.apply(txn); err != nil {
		return err
	}

	s.txnMempool = append(s.txnMempool, txn)
	return nil
}

func (s *State) AddBlock(b Block) (Hash, error) {
	pendingState := s.copy()

	err := applyBlock(b, pendingState)
	if err != nil {
		return Hash{}, err
	}

	blockHash, err := b.Hash()
	if err != nil {
		return Hash{}, err
	}

	blockFs := BlockFS{blockHash, b}
	blockFsJson, err := json.Marshal(blockFs)
	if err != nil {
		return Hash{}, err
	}

	fmt.Println("Saving new Block to disk:")
	fmt.Printf("\t%s\n", blockFsJson)

	_, err = s.dbFile.Write(append(blockFsJson, '\n'))
	if err != nil {
		return Hash{}, err
	}

	s.Balances = pendingState.Balances
	s.latestBlockHash = blockHash
	s.latestBlock = b
	s.hasGenesisBlock = true

	return blockHash, nil
}

func (s *State) AddBlocks(blocks []Block) error {
	for _, b := range blocks {
		_, err := s.AddBlock(b)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *State) Close() {
	s.dbFile.Close()
}

// applyBlock verifies if block can be added to the blockchain.
// Block metadata are verified as well as transactions within (sufficient balances, etc).
func applyBlock(b Block, s State) error {
	nextExpectedBlockHeight := s.latestBlock.Header.Height + 1

	// validate that the next block number increases by 1
	if s.hasGenesisBlock && b.Header.Height != nextExpectedBlockHeight {
		return fmt.Errorf("next expected block height must be '%d' not '%d'", nextExpectedBlockHeight, b.Header.Height)
	}

	// validate that the incoming block parent hash equals the current block hash
	if s.hasGenesisBlock && s.latestBlock.Header.Height > 0 && !reflect.DeepEqual(b.Header.Parent, s.latestBlockHash) {
		return fmt.Errorf("next block parent hash must be %x not %x", s.latestBlockHash, b.Header.Parent)
	}

	return applyTxns(b.Txns, &s)
}

func applyTxn(txn Txn, s *State) error {
	if txn.IsReward() {
		s.Balances[txn.To] += txn.Value
		return nil
	}

	if txn.Value > s.Balances[txn.From] {
		return fmt.Errorf(
			"insufficient funds; Sender (%s) balance is %d BRS, Txn cost %d BRS",
			txn.From, s.Balances[txn.From], txn.Value,
		)
	}

	s.Balances[txn.From] -= txn.Value
	s.Balances[txn.To] += txn.Value

	return nil
}

func applyTxns(txns []Txn, s *State) error {
	for _, txn := range txns {
		err := applyTxn(txn, s)
		if err != nil {
			return err
		}
	}
	return nil
}
