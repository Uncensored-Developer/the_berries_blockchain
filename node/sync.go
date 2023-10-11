package node

import (
	"context"
	"fmt"
	"kryptcoin/database"
	"net/http"
	"time"
)

func (n *Node) sync(ctx context.Context) error {
	ticker := time.NewTicker(45 * time.Second)

	for {
		select {
		case <-ticker.C:
			fmt.Println("Searching for new Peers and Blocks...")
			n.doSync()
		case <-ctx.Done():
			ticker.Stop()
		}
	}
}

func (n *Node) syncBlocks(peer PeerNode, status StatusRes) error {
	localBlockHeight := n.state.LatestBlock().Header.Height

	// Ignore if peer has no blocks
	if status.Hash.IsEmpty() {
		return nil
	}

	// Ignore if peer has fewer blocks than local
	if status.Height < localBlockHeight {
		return nil
	}

	// Ignore if it's the genesis block and it has already been synced
	if status.Height == 0 && !n.state.LatestBlockHash().IsEmpty() {
		return nil
	}

	newBlockCount := status.Height - localBlockHeight
	if localBlockHeight == 0 && status.Height == 0 {
		newBlockCount = 1
	}
	fmt.Printf("Found %d new block(s) from Peer %s\n", newBlockCount, peer.TcpAddress())

	blocks, err := fetchBlocksFromPeer(peer, n.state.LatestBlockHash())
	if err != nil {
		fmt.Println(err)
		return err
	}
	return n.state.AddBlocks(blocks)
}

func (n *Node) syncKnownPeers(status StatusRes) error {
	for _, statusPeer := range status.KnownPeers {
		if !n.IsKnownPeer(statusPeer) {
			fmt.Printf("Found new Peer %s\n", statusPeer.TcpAddress())
			n.AddPeer(statusPeer)
		}
	}
	return nil
}

func (n *Node) doSync() {
	for _, peer := range n.knownPeers {
		if n.info.IP == peer.IP && n.info.Port == peer.Port {
			continue
		}

		fmt.Printf("Searching for new Peers and their Blocks and Peers: %s \n", peer.TcpAddress())
		status, err := queryPeerStatus(peer)
		if err != nil {
			fmt.Printf("ERROR: %s\n", err)

			n.RemovePeer(peer)
			fmt.Printf("Peer %s was removed from the known peers\n", peer.TcpAddress())
			continue
		}

		err = n.JoinKnownPeers(peer)
		if err != nil {
			fmt.Printf("ERROR: %s\n", err)
			continue
		}

		err = n.syncBlocks(peer, status)
		if err != nil {
			fmt.Printf("ERROR: %s\n", err)
			continue
		}

		err = n.syncKnownPeers(status)
		if err != nil {
			fmt.Printf("ERROR: %s\n", err)
			continue
		}
	}
}

func (n *Node) JoinKnownPeers(peer PeerNode) error {
	if peer.connected {
		return nil
	}

	url := fmt.Sprintf(
		"http://%s%s?%s=%s&%s=%d",
		peer.TcpAddress(),
		pathAddPeer,
		pathAddPeerQueryKeyIP,
		n.info.IP,
		pathAddPeerQueryKeyPort,
		n.info.Port,
	)
	res, err := http.Get(url)
	if err != nil {
		return err
	}

	addPeerRes := AddPeerRes{}
	err = readRes(res, &addPeerRes)
	if err != nil {
		return err
	}
	if addPeerRes.Error != "" {
		return fmt.Errorf(addPeerRes.Error)
	}

	knownPeer := n.knownPeers[peer.TcpAddress()]
	knownPeer.connected = addPeerRes.Success

	n.AddPeer(knownPeer)

	if !addPeerRes.Success {
		return fmt.Errorf("unable to join knownPeers of '%s'", peer.TcpAddress())
	}
	return nil
}

func queryPeerStatus(peer PeerNode) (StatusRes, error) {
	url := fmt.Sprintf("http://%s%s", peer.TcpAddress(), pathNodeStatus)
	res, err := http.Get(url)
	if err != nil {
		return StatusRes{}, err
	}

	statusRes := StatusRes{}
	err = readRes(res, &statusRes)
	if err != nil {
		return StatusRes{}, err
	}
	return statusRes, nil
}

func fetchBlocksFromPeer(peer PeerNode, fromBlock database.Hash) ([]database.Block, error) {
	fmt.Printf("Importing blocks from Peer %s...\n", peer.TcpAddress())

	url := fmt.Sprintf(
		"http://%s%s?%s=%s",
		peer.TcpAddress(),
		pathNodeSync,
		pathSyncQueryKeyFromBlock,
		fromBlock.Hex(),
	)

	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	syncRes := SyncRes{}
	err = readRes(res, &syncRes)
	if err != nil {
		return nil, err
	}
	return syncRes.Blocks, nil
}
