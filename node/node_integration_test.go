package node

import (
	"context"
	"kryptcoin/database"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func getTestDataDirPath() string {
	return filepath.Join(os.TempDir(), ".test_db")
}

func TestNode_Run(t *testing.T) {
	// Remove the test directory if it already exists
	dataDir := getTestDataDirPath()
	err := os.RemoveAll(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	n := NewNode(
		dataDir,
		"127.0.0.1",
		8089,
		PeerNode{},
		database.NewAccount("gold_rodger"),
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
	// Remove the test directory if it already exists
	dataDir := getTestDataDirPath()
	err := os.RemoveAll(dataDir)
	if err != nil {
		t.Fatal(err)
	}

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
		database.NewAccount("gold_rodger"),
	)

	// Allow the mining to run for 10 minutes, worst case
	ctx, shutDownNode := context.WithTimeout(context.Background(), time.Minute*10)

	// Send a new TXN 3 seconds from now, in a separate goroutine because n.Run()
	// is a blocking call
	go func() {
		time.Sleep(time.Second * 3)
		txn := database.NewTxn("gold_rodger", "white_beard", 1, "")

		_ = n.AddPendingTxn(txn, pN) // Add txn to Mempool
	}()

	// Send a new TXN 12 seconds from now, in a separate goroutine
	// simulating that it came in while the first TXN is being mined
	go func() {
		time.Sleep(time.Second * 12)
		txn := database.NewTxn("gold_rodger", "white_beard", 2, "")

		_ = n.AddPendingTxn(txn, pN) // Add txn to Mempool
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

func TestNode_MiningStopsOnNewSyncedBlock(t *testing.T) {
	// Remove the test directory if it already exists
	dataDir := getTestDataDirPath()
	err := os.RemoveAll(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	// Would be used to describe the node(local) the TXN came from
	pN := NewPeerNode(
		"127.0.0.1",
		8089,
		false,
		database.NewAccount(""),
		true,
	)

	goldRodgerAcct := database.NewAccount("gold_rodger")
	whiteBeardAcct := database.NewAccount("white_beard")

	n := NewNode(dataDir, pN.IP, pN.Port, pN, whiteBeardAcct)

	// Allow the mining to run for 10 minutes, worst case
	ctx, shutDownNode := context.WithTimeout(context.Background(), time.Minute*10)

	txn1 := database.NewTxn("gold_rodger", "white_beard", 1, "")
	txn2 := database.NewTxn("gold_rodger", "white_beard", 2, "")
	txn2Hash, _ := txn2.Hash()

	// Pre-mine a valid block without running the `n.Run()`
	// with gold_rodger as a miner who will receive the block reward,
	// to simulate the block came on the fly from another peer
	validPreMinedPendingBlock := NewPendingBlock(
		database.Hash{},
		0,
		goldRodgerAcct,
		[]database.Txn{txn1},
	)

	validSyncedBlock, err := Mine(ctx, validPreMinedPendingBlock)
	if err != nil {
		t.Fatal(err)
	}

	// Add 2 new TXNs into white_beard's node
	go func() {
		time.Sleep(time.Second * (miningIntervalSeconds - 2))

		err := n.AddPendingTxn(txn1, pN)
		if err != nil {
			t.Error(err)
			return
		}

		err = n.AddPendingTxn(txn2, pN)
		if err != nil {
			t.Error(err)
			return
		}
	}()

	// Once white_beard is mining the block, simulate
	// that gold_rodger mines the block with TXN1 faster
	go func() {
		time.Sleep(time.Second * (miningIntervalSeconds + 2))
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
		startingGoldRodgerBalance := n.state.Balances[goldRodgerAcct]
		startingWhiteBeardBalance := n.state.Balances[whiteBeardAcct]

		// Wait until the 10min timeout is reached or
		// the 2 blocks got already mined and the closeNode() was triggered
		<-ctx.Done()

		// Check balances again
		endGoldRodgerBalance := n.state.Balances[goldRodgerAcct]
		endWhiteBeardBalance := n.state.Balances[whiteBeardAcct]

		// In TXN1 gold_rodger transferred 1 OPB token to white_beard
		// In TXN2 gold_rodger transferred 2 OPB token to white_beard
		expectedEndGoldRodgerBalance := startingGoldRodgerBalance - txn1.Value - txn2.Value + database.Reward
		expectedEndWhiteBeardBalance := startingWhiteBeardBalance + txn1.Value + txn2.Value + database.Reward

		if endGoldRodgerBalance != expectedEndGoldRodgerBalance {
			t.Fatalf(
				"gold_rodger's expected end balance is %d not %d",
				expectedEndGoldRodgerBalance, endGoldRodgerBalance,
			)
		}

		if endWhiteBeardBalance != expectedEndWhiteBeardBalance {
			t.Fatalf(
				"white_beard's expected end balance is %d not %d",
				expectedEndWhiteBeardBalance, endWhiteBeardBalance,
			)
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
