package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/krakovia/blockchain/pkg/blockchain"
	"github.com/krakovia/blockchain/pkg/node"
	"github.com/krakovia/blockchain/pkg/wallet"
)

// cores para output
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
)

func main() {
	fmt.Printf("%s", colorCyan)
	fmt.Println("==============================================")
	fmt.Println("  Krakovia Blockchain - Integration Demo")
	fmt.Println("==============================================")
	fmt.Printf("%s\n", colorReset)

	// Limpar diretórios de teste
	cleanupDirs()
	defer cleanupDirs()

	// 1. Criar wallets
	fmt.Printf("%s[Setup] Creating wallets...%s\n", colorYellow, colorReset)
	wallet1, err := wallet.NewWallet()
	if err != nil {
		log.Fatal(err)
	}
	wallet2, err := wallet.NewWallet()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("  Wallet 1: %s\n", wallet1.GetAddress()[:20]+"...")
	fmt.Printf("  Wallet 2: %s\n\n", wallet2.GetAddress()[:20]+"...")

	// 2. Criar genesis block
	fmt.Printf("%s[Setup] Creating genesis block...%s\n", colorYellow, colorReset)
	genesisTx := blockchain.NewCoinbaseTransaction(wallet1.GetAddress(), 1000000000, 0)
	genesisBlock := blockchain.GenesisBlock(genesisTx)
	fmt.Printf("  Genesis hash: %s\n", genesisBlock.Hash[:16]+"...")
	fmt.Printf("  Initial supply: 1,000,000,000 tokens to wallet 1\n\n")

	// 3. Criar Node 1
	fmt.Printf("%s[Node 1] Creating and starting...%s\n", colorBlue, colorReset)
	config1 := createNodeConfig("demo_node1", ":29001", wallet1, genesisBlock)
	node1, err := node.NewNode(config1)
	if err != nil {
		log.Fatal(err)
	}
	if err := node1.Start(); err != nil {
		log.Fatal(err)
	}
	defer node1.Stop()

	fmt.Printf("%s[Node 1] ✓ Started%s\n", colorGreen, colorReset)
	fmt.Printf("  Balance: %d\n", node1.GetBalance())
	fmt.Printf("  Height: %d\n\n", node1.GetChainHeight())

	// 4. Node 1 faz stake
	fmt.Printf("%s[Node 1] Staking 100,000 tokens...%s\n", colorBlue, colorReset)
	stakeTx, err := node1.CreateStakeTransaction(100000, 10)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s[Node 1] ✓ Stake transaction: %s%s\n\n", colorGreen, stakeTx.ID[:8]+"...", colorReset)

	// 5. Node 1 inicia mineração
	fmt.Printf("%s[Node 1] Starting mining...%s\n", colorBlue, colorReset)
	if err := node1.StartMining(); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s[Node 1] ✓ Mining started%s\n\n", colorGreen, colorReset)

	// 6. Aguardar alguns blocos
	fmt.Printf("%s[Demo] Waiting for blocks to be mined...%s\n", colorYellow, colorReset)
	time.Sleep(3 * time.Second)
	height1 := node1.GetChainHeight()
	fmt.Printf("%s[Node 1] ✓ Mined %d blocks%s\n\n", colorGreen, height1, colorReset)

	// 7. Node 1 cria transação para wallet 2
	fmt.Printf("%s[Node 1] Creating transaction to wallet 2...%s\n", colorBlue, colorReset)
	tx, err := node1.CreateTransaction(wallet2.GetAddress(), 50000, 5, "demo transfer")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s[Node 1] ✓ Transaction: %s%s\n", colorGreen, tx.ID[:8]+"...", colorReset)
	fmt.Printf("  From: wallet 1\n")
	fmt.Printf("  To: wallet 2\n")
	fmt.Printf("  Amount: 50,000\n")
	fmt.Printf("  Fee: 5\n\n")

	// 8. Aguardar transação ser minerada
	fmt.Printf("%s[Demo] Waiting for transaction to be mined...%s\n", colorYellow, colorReset)
	time.Sleep(2 * time.Second)
	fmt.Printf("%s[Node 1] ✓ Transaction mined%s\n\n", colorGreen, colorReset)

	// 9. Criar Node 2 (vai sincronizar)
	fmt.Printf("%s[Node 2] Creating and starting (will sync)...%s\n", colorPurple, colorReset)
	config2 := createNodeConfig("demo_node2", ":29002", wallet2, genesisBlock)
	node2, err := node.NewNode(config2)
	if err != nil {
		log.Fatal(err)
	}
	if err := node2.Start(); err != nil {
		log.Fatal(err)
	}
	defer node2.Stop()

	fmt.Printf("%s[Node 2] ✓ Started%s\n", colorGreen, colorReset)
	fmt.Printf("  Initial height: %d\n\n", node2.GetChainHeight())

	// 10. Aguardar sincronização
	fmt.Printf("%s[Demo] Waiting for synchronization...%s\n", colorYellow, colorReset)
	time.Sleep(5 * time.Second)

	height2 := node2.GetChainHeight()
	balance2 := node2.GetBalance()

	fmt.Printf("%s[Node 2] ✓ Synchronized%s\n", colorGreen, colorReset)
	fmt.Printf("  Height: %d\n", height2)
	fmt.Printf("  Balance: %d (received from transaction)\n\n", balance2)

	// 11. Node 2 responde com uma transação
	fmt.Printf("%s[Node 2] Sending transaction back to wallet 1...%s\n", colorPurple, colorReset)
	tx2, err := node2.CreateTransaction(wallet1.GetAddress(), 5000, 5, "demo reply")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s[Node 2] ✓ Transaction: %s%s\n", colorGreen, tx2.ID[:8]+"...", colorReset)
	fmt.Printf("  Amount: 5,000\n\n")

	// 12. Aguardar propagação
	fmt.Printf("%s[Demo] Waiting for propagation...%s\n", colorYellow, colorReset)
	time.Sleep(1 * time.Second)
	fmt.Printf("%s[Node 1] Mempool size: %d%s\n\n", colorBlue, node1.GetMempoolSize(), colorReset)

	// 13. Aguardar mais blocos
	fmt.Printf("%s[Demo] Waiting for more blocks...%s\n", colorYellow, colorReset)
	time.Sleep(3 * time.Second)

	// 14. Estatísticas finais
	fmt.Printf("\n%s", colorCyan)
	fmt.Println("==============================================")
	fmt.Println("              Final Statistics")
	fmt.Println("==============================================")
	fmt.Printf("%s", colorReset)

	fmt.Printf("\n%s[Node 1]%s\n", colorBlue, colorReset)
	fmt.Printf("  Height: %d\n", node1.GetChainHeight())
	fmt.Printf("  Balance: %d\n", node1.GetBalance())
	fmt.Printf("  Stake: %d\n", node1.GetStake())
	fmt.Printf("  Mempool: %d transactions\n", node1.GetMempoolSize())
	fmt.Printf("  Peers: %d\n", len(node1.GetPeers()))

	fmt.Printf("\n%s[Node 2]%s\n", colorPurple, colorReset)
	fmt.Printf("  Height: %d\n", node2.GetChainHeight())
	fmt.Printf("  Balance: %d\n", node2.GetBalance())
	fmt.Printf("  Mempool: %d transactions\n", node2.GetMempoolSize())
	fmt.Printf("  Peers: %d\n", len(node2.GetPeers()))

	// Verificação
	fmt.Printf("\n%s", colorCyan)
	fmt.Println("==============================================")
	fmt.Println("              Verification")
	fmt.Println("==============================================")
	fmt.Printf("%s", colorReset)

	if node1.GetChainHeight() == node2.GetChainHeight() {
		fmt.Printf("%s✓ Chains synchronized (height %d)%s\n", colorGreen, node1.GetChainHeight(), colorReset)
	} else {
		fmt.Printf("%s✗ Chain heights differ: %d vs %d%s\n", colorRed, node1.GetChainHeight(), node2.GetChainHeight(), colorReset)
	}

	if balance2 > 0 {
		fmt.Printf("%s✓ Transaction propagated (wallet 2 balance: %d)%s\n", colorGreen, balance2, colorReset)
	} else {
		fmt.Printf("%s✗ Transaction not received%s\n", colorRed, colorReset)
	}

	if node1.GetStake() > 0 {
		fmt.Printf("%s✓ Staking working (stake: %d)%s\n", colorGreen, node1.GetStake(), colorReset)
	}

	if node1.GetChainHeight() > 1 {
		fmt.Printf("%s✓ PoS mining working (%d blocks)%s\n", colorGreen, node1.GetChainHeight(), colorReset)
	}

	fmt.Printf("\n%s", colorCyan)
	fmt.Println("==============================================")
	fmt.Println("      Demo completed successfully!")
	fmt.Println("      Press Ctrl+C to exit")
	fmt.Println("==============================================")
	fmt.Printf("%s\n", colorReset)

	// Aguardar interrupção
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Printf("\n%s[Demo] Shutting down...%s\n", colorYellow, colorReset)
}

func createNodeConfig(nodeID, port string, w *wallet.Wallet, genesis *blockchain.Block) node.Config {
	dbPath := filepath.Join(os.TempDir(), "krakovia_demo_"+nodeID)

	return node.Config{
		ID:                nodeID,
		Address:           port,
		DBPath:            dbPath,
		SignalingServer:   "ws://localhost:9000/ws",
		MaxPeers:          10,
		MinPeers:          1,
		DiscoveryInterval: 5,
		Wallet:            w,
		GenesisBlock:      genesis,
		ChainConfig:       blockchain.DefaultChainConfig(),
	}
}

func cleanupDirs() {
	os.RemoveAll(filepath.Join(os.TempDir(), "krakovia_demo_demo_node1"))
	os.RemoveAll(filepath.Join(os.TempDir(), "krakovia_demo_demo_node2"))
}
