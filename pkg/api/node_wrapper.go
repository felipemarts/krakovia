package api

import (
	"github.com/krakovia/blockchain/pkg/blockchain"
	"github.com/krakovia/blockchain/pkg/network"
)

// RealNode interface que representa o node real
type RealNode interface {
	GetID() string
	GetWalletAddress() string
	GetChainHeight() uint64
	GetBalance() uint64
	GetStake() uint64
	GetNonce() uint64
	GetMempoolSize() int
	GetPeers() []*network.Peer
	GetLastBlock() *blockchain.Block
	IsMining() bool
	StartMining() error
	StopMining()
	CreateTransaction(to string, amount, fee uint64, data string) (*blockchain.Transaction, error)
	CreateStakeTransaction(amount, fee uint64) (*blockchain.Transaction, error)
	CreateUnstakeTransaction(amount, fee uint64) (*blockchain.Transaction, error)
}

// NodeWrapper envolve o node real para implementar NodeInterface
type NodeWrapper struct {
	node RealNode
}

// NewNodeWrapper cria um novo wrapper
func NewNodeWrapper(node RealNode) *NodeWrapper {
	return &NodeWrapper{node: node}
}

func (w *NodeWrapper) GetID() string {
	return w.node.GetID()
}

func (w *NodeWrapper) GetWalletAddress() string {
	return w.node.GetWalletAddress()
}

func (w *NodeWrapper) GetChainHeight() uint64 {
	return w.node.GetChainHeight()
}

func (w *NodeWrapper) GetBalance() uint64 {
	return w.node.GetBalance()
}

func (w *NodeWrapper) GetStake() uint64 {
	return w.node.GetStake()
}

func (w *NodeWrapper) GetNonce() uint64 {
	return w.node.GetNonce()
}

func (w *NodeWrapper) GetMempoolSize() int {
	return w.node.GetMempoolSize()
}

func (w *NodeWrapper) GetPeers() []PeerInfo {
	realPeers := w.node.GetPeers()
	peers := make([]PeerInfo, len(realPeers))
	for i, p := range realPeers {
		peers[i] = &PeerAdapter{peer: p}
	}
	return peers
}

func (w *NodeWrapper) GetLastBlock() BlockInfo {
	block := w.node.GetLastBlock()
	return &BlockAdapter{block: block}
}

func (w *NodeWrapper) IsMining() bool {
	return w.node.IsMining()
}

func (w *NodeWrapper) StartMining() error {
	return w.node.StartMining()
}

func (w *NodeWrapper) StopMining() {
	w.node.StopMining()
}

func (w *NodeWrapper) CreateTransaction(to string, amount, fee uint64, data string) (TxInfo, error) {
	tx, err := w.node.CreateTransaction(to, amount, fee, data)
	if err != nil {
		return nil, err
	}
	return &TxAdapter{tx: tx}, nil
}

func (w *NodeWrapper) CreateStakeTransaction(amount, fee uint64) (TxInfo, error) {
	tx, err := w.node.CreateStakeTransaction(amount, fee)
	if err != nil {
		return nil, err
	}
	return &TxAdapter{tx: tx}, nil
}

func (w *NodeWrapper) CreateUnstakeTransaction(amount, fee uint64) (TxInfo, error) {
	tx, err := w.node.CreateUnstakeTransaction(amount, fee)
	if err != nil {
		return nil, err
	}
	return &TxAdapter{tx: tx}, nil
}
