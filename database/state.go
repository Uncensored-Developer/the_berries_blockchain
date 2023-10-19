package database

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"log"
	"os"
	"reflect"
	"sort"
)

const (
	TxnFee          = uint(20)
	TxnGas          = 10
	DefaultGasPrice = 1
)

type State struct {
	Balances        map[common.Address]uint
	AccountNonces   map[common.Address]uint
	dbFile          *os.File
	latestBlock     Block
	latestBlockHash Hash
	hasGenesisBlock bool
	forkOIP1        uint64

	HeightCache map[uint64]int64
	HashCache   map[string]int64
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

func (s *State) GetNextAccountNonce(account common.Address) uint {
	return s.AccountNonces[account] + 1
}

func (s *State) IsForkOIP1() bool {
	log.Printf("GASS: %d, %d", s.NextBlockHeight(), s.forkOIP1)
	return s.NextBlockHeight() >= s.forkOIP1
}

func NewStateFromDisk(dataDir string) (*State, error) {
	err := InitDataDirIfNotExists(dataDir, []byte(genesisJson))
	if err != nil {
		return nil, err
	}

	genesis, err := loadGenesis(getGenesisJsonFilePath(dataDir))
	if err != nil {
		return nil, err
	}

	balances := make(map[common.Address]uint)
	for account, balance := range genesis.Balances {
		balances[account] = balance
	}

	accountNonces := make(map[common.Address]uint)

	f, err := os.OpenFile(getBlocksDbFilePath(dataDir), os.O_APPEND|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(f)
	state := &State{
		balances,
		accountNonces,
		f,
		Block{},
		Hash{},
		false,
		genesis.ForkOIP1,
		map[uint64]int64{},
		map[string]int64{},
	}

	filePos := int64(0)
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

		err = applyBlock(blockFs.Value, state)
		if err != nil {
			return nil, err
		}

		state.latestBlockHash = blockFs.Key
		state.latestBlock = blockFs.Value
		state.hasGenesisBlock = true

		// Set caches
		state.HashCache[blockFs.Key.Hex()] = filePos
		state.HeightCache[blockFs.Value.Header.Height] = filePos
		filePos += int64(len(blockFsJson)) + 1
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

func (s *State) Copy() State {
	c := State{}
	c.hasGenesisBlock = s.hasGenesisBlock
	c.latestBlock = s.latestBlock
	c.latestBlockHash = s.latestBlockHash
	c.Balances = make(map[common.Address]uint)
	c.AccountNonces = make(map[common.Address]uint)
	c.forkOIP1 = s.forkOIP1

	for acct, balance := range s.Balances {
		c.Balances[acct] = balance
	}

	for acct, nonce := range s.AccountNonces {
		c.AccountNonces[acct] = nonce
	}

	return c
}

func (s *State) AddBlock(b Block) (Hash, error) {
	pendingState := s.Copy()

	err := applyBlock(b, &pendingState)
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

	log.Println("Saving new Block to disk:")
	log.Printf("\t%s\n", blockFsJson)

	// get file position for cache
	fs, _ := s.dbFile.Stat()
	filePos := fs.Size() + 1

	_, err = s.dbFile.Write(append(blockFsJson, '\n'))
	if err != nil {
		return Hash{}, err
	}

	s.Balances = pendingState.Balances
	s.AccountNonces = pendingState.AccountNonces
	s.latestBlockHash = blockHash
	s.latestBlock = b
	s.hasGenesisBlock = true

	// Set search caches
	s.HashCache[blockFs.Key.Hex()] = filePos
	s.HeightCache[blockFs.Value.Header.Height] = filePos

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
func applyBlock(b Block, s *State) error {
	nextExpectedBlockHeight := s.latestBlock.Header.Height + 1

	// validate that the next block number increases by 1
	if s.hasGenesisBlock && b.Header.Height != nextExpectedBlockHeight {
		return fmt.Errorf("next expected block height must be '%d' not '%d'", nextExpectedBlockHeight, b.Header.Height)
	}

	// validate that the incoming block parent hash equals the current block hash
	if s.hasGenesisBlock && s.latestBlock.Header.Height > 0 && !reflect.DeepEqual(b.Header.Parent, s.latestBlockHash) {
		return fmt.Errorf("next block parent hash must be %x not %x", s.latestBlockHash, b.Header.Parent)
	}

	hash, err := b.Hash()
	if err != nil {
		return err
	}

	if !IsBlockHashValid(hash) {
		return fmt.Errorf("invalid block hash %x", hash)
	}

	err = applyTxns(b.Txns, s)
	if err != nil {
		return err
	}

	// Credit the block reward and the fees from the transactions to the miner
	s.Balances[b.Header.Miner] += Reward
	if s.IsForkOIP1() {
		s.Balances[b.Header.Miner] += b.GasReward()
	} else {
		s.Balances[b.Header.Miner] += uint(len(b.Txns)) * TxnFee
	}
	return nil
}

func ApplyTxn(txn SignedTxn, s *State) error {
	// Verify the TXN was not forged
	ok, err := txn.IsAuthentic()
	if err != nil {
		return err
	}

	if !ok {
		return fmt.Errorf("forged TXN, Sender %s was forged", txn.From.String())
	}

	expectedNonce := s.GetNextAccountNonce(txn.From)
	if txn.Nonce != expectedNonce {
		return fmt.Errorf(
			"invalid Txn, Sender %s next nonce should be %d not %d",
			txn.From.String(),
			expectedNonce,
			txn.Nonce,
		)
	}

	if s.IsForkOIP1() {
		if txn.Gas != TxnGas {
			return fmt.Errorf("insufficient Txn gas, requires %d got %d", TxnGas, txn.Gas)
		}
		if txn.GasPrice < DefaultGasPrice {
			return fmt.Errorf("insufficient Txn gas price, requires at least %d", DefaultGasPrice)
		}
	} else {
		// Prior to OIP1, s signed Txn must not populate gas and gasPrice field to prevent
		// consensus from crashing
		if txn.Gas != 0 || txn.GasPrice != 0 {
			return fmt.Errorf("invalid Txn, Gas and GasPrice cannot be populated before OIP1 fork")
		}
	}

	if txn.TotalCost(s.IsForkOIP1()) > s.Balances[txn.From] {
		return fmt.Errorf("account %s has insufficient balance for %d", txn.From, txn.Value)
	}

	s.Balances[txn.From] -= txn.TotalCost(s.IsForkOIP1())
	s.Balances[txn.To] += txn.Value
	s.AccountNonces[txn.From] = txn.Nonce

	return nil
}

func applyTxns(txns []SignedTxn, s *State) error {
	sort.Slice(txns, func(i, j int) bool {
		return txns[i].Time < txns[j].Time
	})
	for _, txn := range txns {
		err := ApplyTxn(txn, s)
		if err != nil {
			return err
		}
	}
	return nil
}
