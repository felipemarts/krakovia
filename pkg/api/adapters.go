package api

import (
	"github.com/krakovia/blockchain/pkg/blockchain"
	"github.com/krakovia/blockchain/pkg/network"
)

// PeerAdapter adapta network.Peer para PeerInfo
type PeerAdapter struct {
	peer *network.Peer
}

func (p *PeerAdapter) GetID() string {
	return p.peer.ID
}

// BlockAdapter adapta blockchain.Block para BlockInfo
type BlockAdapter struct {
	block *blockchain.Block
}

func (b *BlockAdapter) GetHeight() uint64 {
	if b.block == nil {
		return 0
	}
	return b.block.Header.Height
}

func (b *BlockAdapter) GetHash() string {
	if b.block == nil {
		return ""
	}
	return b.block.Hash
}

func (b *BlockAdapter) GetTimestamp() int64 {
	if b.block == nil {
		return 0
	}
	return b.block.Header.Timestamp
}

func (b *BlockAdapter) GetTransactionCount() int {
	if b.block == nil {
		return 0
	}
	return len(b.block.Transactions)
}

// TxAdapter adapta blockchain.Transaction para TxInfo
type TxAdapter struct {
	tx *blockchain.Transaction
}

func (t *TxAdapter) GetID() string {
	if t.tx == nil {
		return ""
	}
	return t.tx.ID
}
