package node

import (
	"context"
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"io"
	"kryptcoin/database"
	"kryptcoin/wallet"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Pre-generated for testing purposes using wallet_test.go.
// It's necessary to have pre-existing accounts before a new node
// with fresh new, empty keystore is initialized and booted in order
// to configure the accounts balances in genesis.json
const (
	testKeystoreGoldRodgerAccount = "0x0418A658C5874D2Fe181145B685d2e73D761865D"
	testKeystoreWhiteBeardAccount = "0x486512fA9fbaF06568D13826afe7822842b9E685"
	testKeystoreGoldRodgerFile    = "test_goldRodger--0418A658C5874D2Fe181145B685d2e73D761865D"
	testKeystoreWhiteBeardFile    = "test_whiteBeard--486512fA9fbaF06568D13826afe7822842b9E685"
	testKeystorePassword          = "goodbrain"
)

func getTestDataDirPath() (string, error) {
	return os.MkdirTemp(os.TempDir(), "opbb_test")
}

func TestNode_Run(t *testing.T) {
	// Remove the test directory if it already exists
	dataDir, err := getTestDataDirPath()
	if err != nil {
		t.Fatal(err)
	}
	err = os.RemoveAll(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	n := NewNode(
		dataDir,
		"127.0.0.1",
		8089,
		PeerNode{},
		database.NewAccount(DefaultMiner),
	)

	// Define a context with timeout so the Node.Run() will
	// only run for 5s
	ctx, _ := context.WithTimeout(context.Background(), time.Second*5)
	err = n.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
}

func TestNode_Mining(t *testing.T) {
	dataDir, goldRodger, whiteBeard, err := setupTestDir(10_000_000)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dataDir)

	// Would be used to describe the node(local) the TXN came from
	pN := NewPeerNode(
		"127.0.0.1",
		8089,
		false,
		database.NewAccount(""),
		true,
	)

	// New node with gold_rodger miner
	n := NewNode(
		dataDir,
		pN.IP,
		pN.Port,
		pN,
		goldRodger,
	)

	// Allow the mining to run for 10 minutes, worst case
	ctx, shutDownNode := context.WithTimeout(context.Background(), time.Minute*10)

	// Send a new TXN 3 seconds from now, in a separate goroutine because n.Run()
	// is a blocking call
	go func() {
		time.Sleep(time.Second * 3)
		txn := database.NewTxn(goldRodger, whiteBeard, 1, 1, "")
		signedTxn, err := wallet.SignWithKeystoreAccount(
			txn,
			goldRodger,
			testKeystorePassword,
			wallet.GetKeystoreDirPath(dataDir),
		)
		if err != nil {
			t.Error(err)
			return
		}

		_ = n.AddPendingTxn(signedTxn, pN) // Add txn to Mempool
	}()

	// Send a new TXN 12 seconds from now, in a separate goroutine
	// simulating that it came in while the first TXN is being mined
	go func() {
		time.Sleep(time.Second * 12)
		txn := database.NewTxn(goldRodger, whiteBeard, 2, 2, "")
		signedTxn, err := wallet.SignWithKeystoreAccount(
			txn,
			goldRodger,
			testKeystorePassword,
			wallet.GetKeystoreDirPath(dataDir),
		)
		if err != nil {
			t.Error(err)
			return
		}

		err = n.AddPendingTxn(signedTxn, pN) // Add txn to Mempool
		if err != nil {
			t.Error(err)
			return
		}
	}()

	go func() {
		// Periodically check if we have mined the 2 txn
		ticker := time.NewTicker(time.Second * 10)

		for {
			select {
			case <-ticker.C:
				// Has the 2 blocks been mined as expected?
				if n.state.LatestBlock().Header.Height == 1 {
					shutDownNode()
					return
				}
			}
		}
	}()
	_ = n.Run(ctx)

	if n.state.LatestBlock().Header.Height != 1 {
		t.Fatal("2 pending TXNs not mined under 10min")
	}
}

// The test logic summary:
//
//	WhiteBeard runs the node
//	WhiteBeard tries to mine 2 Txns
//	The mining gets interrupted because a new block from GoldRodger gets synced
//	GoldRodger will get the block reward for this synced block
//	The synced block contains 1 of the Txns WhiteBeard tried to mine
//	WhiteBeard tries to mine 1 Txn left
//	WhiteBeard succeeds and gets her block reward
func TestNode_MiningStopsOnNewSyncedBlock(t *testing.T) {
	dataDir, goldRodger, whiteBeard, err := setupTestDir(10_000_000)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dataDir)

	// Would be used to describe the node(local) the TXN came from
	pN := NewPeerNode(
		"127.0.0.1",
		8089,
		false,
		database.NewAccount(""),
		true,
	)

	n := NewNode(dataDir, pN.IP, pN.Port, pN, whiteBeard)

	// Allow the mining to run for 10 minutes, worst case
	ctx, shutDownNode := context.WithTimeout(context.Background(), time.Minute*10)

	txn1 := database.NewTxn(goldRodger, whiteBeard, 1, 1, "")
	signedTxn1, err := wallet.SignWithKeystoreAccount(
		txn1,
		goldRodger,
		testKeystorePassword,
		wallet.GetKeystoreDirPath(dataDir),
	)
	if err != nil {
		t.Fatal(err)
	}
	txn2 := database.NewTxn(goldRodger, whiteBeard, 2, 2, "")
	signedTxn2, err := wallet.SignWithKeystoreAccount(
		txn2,
		goldRodger,
		testKeystorePassword,
		wallet.GetKeystoreDirPath(dataDir),
	)
	if err != nil {
		t.Fatal(err)
	}
	txn2Hash, err := txn2.Hash()
	if err != nil {
		t.Fatal(err)
	}

	// Pre-mine a valid block without running the `n.Run()`
	// with gold_rodger as a miner who will receive the block reward,
	// to simulate the block came on the fly from another peer
	validPreMinedPendingBlock := NewPendingBlock(
		database.Hash{},
		0,
		goldRodger,
		[]database.SignedTxn{signedTxn1},
	)

	validSyncedBlock, err := Mine(ctx, validPreMinedPendingBlock)
	if err != nil {
		t.Fatal(err)
	}

	// Add 2 new TXNs into white_beard's node
	go func() {
		time.Sleep(time.Second * (miningIntervalSeconds - 2))

		err := n.AddPendingTxn(signedTxn1, pN)
		if err != nil {
			t.Error(err)
			return
		}

		err = n.AddPendingTxn(signedTxn2, pN)
		if err != nil {
			t.Error(err)
			return
		}
	}()

	// Once white_beard is mining the block, simulate
	// that gold_rodger mines the block with TXN1 faster
	go func() {
		time.Sleep(time.Second * 12)
		if n.isMining {
			t.Error("Node should be mining")
			return
		}

		_, err := n.state.AddBlock(validSyncedBlock)
		if err != nil {
			t.Error(err)
			return
		}

		// Mock that gold_rodger's block came from a network
		n.newSyncedBlocks <- validSyncedBlock

		time.Sleep(time.Second * 2)
		if n.isMining {
			t.Error("Synced block should have canceled the mining")
			return
		}

		// Mined TXN1 by gold_rodger should be removed from the Mempool
		_, onlyTxn2IsPending := n.pendingTxns[txn2Hash.Hex()]
		if len(n.pendingTxns) != 1 && !onlyTxn2IsPending {
			t.Error("Synced block should have canceled the mining of already mined TXN1")
			return
		}

		time.Sleep(time.Second * (miningIntervalSeconds + 2))
		if !n.isMining {
			t.Error("Should attempt to mine TXN2 npot included in the synced block")
			return
		}
	}()

	go func() {
		// Regularly check when both TXNs are mined
		ticker := time.NewTicker(time.Second * 10)

		for {
			select {
			case <-ticker.C:
				if n.state.LatestBlock().Header.Height == 1 {
					shutDownNode()
					return
				}
			}
		}
	}()

	go func() {
		time.Sleep(time.Second * 2)

		// Take a snapshot of the DB balances
		// before the mining is finished and the 2 blocks
		// are created.
		startingGoldRodgerBalance := n.state.Balances[goldRodger]
		startingWhiteBeardBalance := n.state.Balances[whiteBeard]

		// Wait until the 10min timeout is reached or
		// the 2 blocks got already mined and the closeNode() was triggered
		<-ctx.Done()

		// Check balances again
		endGoldRodgerBalance := n.state.Balances[goldRodger]
		endWhiteBeardBalance := n.state.Balances[whiteBeard]

		// In TXN1 gold_rodger transferred 1 OPB token to white_beard
		// In TXN2 gold_rodger transferred 2 OPB token to white_beard
		expectedEndGoldRodgerBalance := startingGoldRodgerBalance - txn1.Value - txn2.Value + database.Reward
		expectedEndWhiteBeardBalance := startingWhiteBeardBalance + txn1.Value + txn2.Value + database.Reward

		if endGoldRodgerBalance != expectedEndGoldRodgerBalance {
			t.Errorf(
				"gold_rodger's expected end balance is %d not %d",
				expectedEndGoldRodgerBalance, endGoldRodgerBalance,
			)
			return
		}

		if endWhiteBeardBalance != expectedEndWhiteBeardBalance {
			t.Errorf(
				"white_beard's expected end balance is %d not %d",
				expectedEndWhiteBeardBalance, endWhiteBeardBalance,
			)
			return
		}

		t.Logf("Starting gold_rodger balance: %d", startingGoldRodgerBalance)
		t.Logf("Starting white_beard balance: %d", startingWhiteBeardBalance)
		t.Logf("Ending gold_rodger balance: %d", endGoldRodgerBalance)
		t.Logf("Ending white_beard balance: %d", endWhiteBeardBalance)
	}()

	_ = n.Run(ctx)

	if n.state.LatestBlock().Header.Height != 1 {
		t.Fatal("2 pending TXNs not mined into 2 valid blocks under 10min")
	}

	if len(n.pendingTxns) != 0 {
		t.Fatal("no pending TXNs should be left to mine")
	}
}

func TestNode_ForgedTxn(t *testing.T) {
	dataDir, goldRodger, whiteBeard, err := setupTestDir(10_000_000)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dataDir)

	n := NewNode(
		dataDir,
		"127.0.0.1",
		8081,
		PeerNode{},
		goldRodger,
	)

	// Allow the mining to run for 10 minutes, worst case
	ctx, shutDownNode := context.WithTimeout(context.Background(), time.Minute*10)

	goldRodgerPeerNode := NewPeerNode(
		"127.0.0.1",
		8081,
		false,
		goldRodger,
		true,
	)

	amount := uint(5)
	txnNonce := uint(1)
	txn := database.NewTxn(goldRodger, whiteBeard, amount, txnNonce, "")

	// Create a valid TXN sending 5 OPB tokens from gold_rodger to white_beard
	validSignedTxn, err := wallet.SignWithKeystoreAccount(
		txn,
		goldRodger,
		testKeystorePassword,
		wallet.GetKeystoreDirPath(dataDir),
	)
	if err != nil {
		t.Fatal(err)
	}

	// Trigger mining
	_ = n.AddPendingTxn(validSignedTxn, goldRodgerPeerNode)

	go func() {
		ticker := time.NewTicker(time.Second * 10)
		forgedTxnAdded := false
		for {
			select {
			case <-ticker.C:
				if !n.state.LatestBlockHash().IsEmpty() {
					if forgedTxnAdded && !n.isMining {
						shutDownNode()
						return
					}

					if !forgedTxnAdded {
						// Try to forge the same TXN but with a modified time
						// Because the Txn.time changed, then the signature would be considered forged
						forgedTxn := database.NewTxn(
							goldRodger,
							whiteBeard,
							amount,
							txnNonce,
							"",
						)
						// Construct SignedTxn using signature from previous valid Txn
						forgedSignedTxn := database.NewSignedTxn(forgedTxn, validSignedTxn.Sig)
						_ = n.AddPendingTxn(forgedSignedTxn, goldRodgerPeerNode)

						forgedTxnAdded = true

						time.Sleep(time.Second * 13)
					}
				}
			}
		}
	}()

	_ = n.Run(ctx)
	if n.state.LatestBlock().Header.Height != 0 {
		t.Fatal("should mine only one Txn since the second Txn was forged.")
	}

	if n.state.Balances[whiteBeard] != amount {
		t.Fatalf("forged Txn succeeded")
	}
}

func TestNode_ReplayedTxn(t *testing.T) {
	dataDir, goldRodger, whiteBeard, err := setupTestDir(10_000_000)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dataDir)

	n := NewNode(
		dataDir,
		"127.0.0.1",
		8089,
		PeerNode{},
		goldRodger,
	)

	ctx, shutDownNode := context.WithCancel(context.Background())

	goldRodgerPeerNode := NewPeerNode(
		"127.0.0.1",
		8081,
		false,
		goldRodger,
		true,
	)
	whiteBeardPeerNode := NewPeerNode(
		"127.0.0.1",
		8082,
		false,
		whiteBeard,
		true,
	)

	amount := uint(5)
	txnNonce := uint(1)
	txn := database.NewTxn(goldRodger, whiteBeard, amount, txnNonce, "")

	// Create a valid TXN sending 5 OPB tokens from gold_rodger to white_beard
	validSignedTxn, err := wallet.SignWithKeystoreAccount(
		txn,
		goldRodger,
		testKeystorePassword,
		wallet.GetKeystoreDirPath(dataDir),
	)
	if err != nil {
		t.Fatal(err)
	}

	_ = n.AddPendingTxn(validSignedTxn, goldRodgerPeerNode)

	go func() {
		ticker := time.NewTicker(time.Second * 10)
		replayedTxnAdded := false

		for {
			select {
			case <-ticker.C:
				if !n.state.LatestBlockHash().IsEmpty() {
					if replayedTxnAdded && !n.isMining {
						shutDownNode()
						return
					}

					// gold_rodger's original Txn got mined
					// Execute the attack by replaying the Txn again
					if !replayedTxnAdded {
						// Simulate the Txn was sent to a different node
						n.archivedTxns = make(map[string]database.SignedTxn)

						_ = n.AddPendingTxn(validSignedTxn, whiteBeardPeerNode)
						replayedTxnAdded = true

						time.Sleep(time.Second * 13)
					}
				}

			}
		}
	}()

	_ = n.Run(ctx)

	if n.state.Balances[whiteBeard] == amount*2 {
		t.Fatalf("replayed attack was successful")
	}

	if n.state.LatestBlock().Header.Height == 1 {
		t.Fatalf("only the 1st block should be saved since the 2nd block contains a malicious Txn")
	}
}

func TestNode_SpamTransactions(t *testing.T) {
	goldRodgerBalance := uint(1_000_000)
	whiteBeardBalance := uint(0)
	minerBalance := uint(0)

	dataDir, goldRodger, whiteBeard, err := setupTestDir(goldRodgerBalance)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dataDir)

	minerKey, err := wallet.NewRandomKey()
	if err != nil {
		t.Fatal(err)
	}
	miner := minerKey.Address

	n := NewNode(
		dataDir,
		"127.0.0.1",
		8081,
		PeerNode{},
		miner,
	)

	ctx, shutDownNode := context.WithCancel(context.Background())
	minerPeerNode := NewPeerNode(
		"127.0.0.1",
		8081,
		false,
		miner,
		true,
	)

	amount := uint(500)
	count := uint(4)

	go func() {
		// Wait for the node to run and initialize its state
		time.Sleep(time.Second)

		spamTxns := make([]database.SignedTxn, count)
		now := uint64(time.Now().Unix())

		for i := uint(1); i <= count; i++ {
			txnNonce := i
			txn := database.NewTxn(goldRodger, whiteBeard, amount, txnNonce, "")

			// Ensure every Txn has a unique timestamp and the nonce 0 is the oldest
			txn.Time = now - uint64(count-i*100)

			signedTxn, err := wallet.SignWithKeystoreAccount(
				txn,
				goldRodger,
				testKeystorePassword,
				wallet.GetKeystoreDirPath(dataDir),
			)
			if err != nil {
				t.Error(err)
				return
			}

			spamTxns[i-1] = signedTxn
		}

		for _, txn := range spamTxns {
			_ = n.AddPendingTxn(txn, minerPeerNode)
		}
	}()

	go func() {
		// Periodically check if we mined the block
		ticker := time.NewTicker(time.Second * 10)

		for {
			select {
			case <-ticker.C:
				if !n.state.LatestBlockHash().IsEmpty() {
					shutDownNode()
					return
				}
			}
		}
	}()

	_ = n.Run(ctx)

	expectedGoldRodgerBalance := goldRodgerBalance - (count * amount) - (count * database.TxnGasFee)
	expectedWhiteBeardBalance := whiteBeardBalance + (count * amount)
	expectedMinerBalance := minerBalance + database.Reward + (count * database.TxnGasFee)

	if n.state.Balances[whiteBeard] != expectedWhiteBeardBalance {
		t.Errorf(
			"white_beard balance incorrect. Expected %d, got %d",
			expectedWhiteBeardBalance,
			n.state.Balances[whiteBeard],
		)
	}
	if n.state.Balances[goldRodger] != expectedGoldRodgerBalance {
		t.Errorf(
			"gold_rodger balance incorrect. Expected %d, got %d",
			expectedGoldRodgerBalance,
			n.state.Balances[goldRodger],
		)
	}
	if n.state.Balances[miner] != expectedMinerBalance {
		t.Errorf(
			"miner balance incorrect. Expected %d, got %d",
			expectedMinerBalance,
			n.state.Balances[miner],
		)
	}

	t.Logf("gold_rodger final balance: %d OPB", n.state.Balances[goldRodger])
	t.Logf("white_beard final balance: %d OPB", n.state.Balances[whiteBeard])
	t.Logf("miner final balance: %d OPB", n.state.Balances[miner])
}

// Copy the pre-generated keystore files from this folder into the new testDataDirPath()
// Afterwards the test data_dir path will look like:
// "/tmp/opbb_test/keystore/test_goldRodger--0418A658C5874D2Fe181145B685d2e73D761865D"
// "/tmp/opbb_test/keystore/test_whiteBeard--486512fA9fbaF06568D13826afe7822842b9E685"
func copyKeystoreFileToTestDataDirPath(dataDir string) error {
	goldRodgerKsSrc, err := os.Open(testKeystoreGoldRodgerFile)
	if err != nil {
		return err
	}
	defer goldRodgerKsSrc.Close()

	ksDir := filepath.Join(wallet.GetKeystoreDirPath(dataDir))
	err = os.Mkdir(ksDir, 0777)
	if err != nil {
		return err
	}

	goldRodgerKsDst, err := os.Create(filepath.Join(ksDir, testKeystoreGoldRodgerFile))
	if err != nil {
		return err
	}
	defer goldRodgerKsDst.Close()

	_, err = io.Copy(goldRodgerKsDst, goldRodgerKsSrc)
	if err != nil {
		return err
	}

	whiteBeardKsSrc, err := os.Open(testKeystoreWhiteBeardFile)
	if err != nil {
		return err
	}
	defer whiteBeardKsSrc.Close()

	whiteBeardKsDst, err := os.Create(filepath.Join(ksDir, testKeystoreWhiteBeardFile))
	if err != nil {
		return err
	}
	defer whiteBeardKsDst.Close()

	_, err = io.Copy(whiteBeardKsDst, whiteBeardKsSrc)
	if err != nil {
		return err
	}
	return nil
}

// setupTestNodeDir creates a default testing node directory with 2 keystore accounts
func setupTestDir(goldRodgerStartBalance uint) (dataDir string, goldRodger, whiteBeard common.Address, err error) {
	goldRodger = database.NewAccount(testKeystoreGoldRodgerAccount)
	whiteBeard = database.NewAccount(testKeystoreWhiteBeardAccount)

	dataDir, err = getTestDataDirPath()
	if err != nil {
		return "", common.Address{}, common.Address{}, err
	}

	genesisBalances := make(map[common.Address]uint)
	genesisBalances[goldRodger] = goldRodgerStartBalance
	genesis := database.Genesis{Balances: genesisBalances}
	genesisJson, err := json.Marshal(genesis)
	if err != nil {
		return "", common.Address{}, common.Address{}, err
	}

	err = database.InitDataDirIfNotExists(dataDir, genesisJson)
	if err != nil {
		return "", common.Address{}, common.Address{}, err
	}

	err = copyKeystoreFileToTestDataDirPath(dataDir)
	if err != nil {
		return "", common.Address{}, common.Address{}, err
	}
	return dataDir, goldRodger, whiteBeard, nil
}
