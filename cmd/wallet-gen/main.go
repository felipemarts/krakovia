package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/krakovia/blockchain/pkg/wallet"
)

type WalletOutput struct {
	PrivateKey string `json:"private_key"`
	PublicKey  string `json:"public_key"`
	Address    string `json:"address"`
}

func main() {
	var outputFile string
	var count int

	flag.StringVar(&outputFile, "output", "", "Output file path (default: stdout)")
	flag.IntVar(&count, "count", 1, "Number of wallets to generate")
	flag.Parse()

	if count < 1 {
		log.Fatal("Count must be at least 1")
	}

	wallets := make([]WalletOutput, 0, count)

	for i := 0; i < count; i++ {
		w, err := wallet.NewWallet()
		if err != nil {
			log.Fatalf("Failed to create wallet %d: %v", i+1, err)
		}

		walletOutput := WalletOutput{
			PrivateKey: w.GetPrivateKeyHex(),
			PublicKey:  w.GetPublicKeyHex(),
			Address:    w.GetAddress(),
		}

		wallets = append(wallets, walletOutput)
	}

	// Se for apenas uma carteira, exibe diretamente
	var output []byte
	var err error
	if count == 1 {
		output, err = json.MarshalIndent(wallets[0], "", "  ")
	} else {
		output, err = json.MarshalIndent(wallets, "", "  ")
	}

	if err != nil {
		log.Fatalf("Failed to marshal output: %v", err)
	}

	if outputFile != "" {
		err = os.WriteFile(outputFile, output, 0644)
		if err != nil {
			log.Fatalf("Failed to write output file: %v", err)
		}
		fmt.Printf("Wallet(s) written to %s\n", outputFile)
	} else {
		fmt.Println(string(output))
	}
}
