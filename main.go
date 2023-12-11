package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/wallet"
)

type Receiver struct {
	Address string
	Amount  string
}

func main() {
	client := liteclient.NewConnectionPool()
	err := client.AddConnectionsFromConfigUrl(context.Background(), "https://ton.org/global.config.json")
	if err != nil {
		log.Fatalln("connection err: ", err.Error())
		return
	}

	api := ton.NewAPIClient(client)

	ctx := client.StickyContext(context.Background())

	reader := bufio.NewReader(os.Stdin)

	// Get seed phrase from user
	fmt.Print("Enter seed phrase (or press enter to generate new wallet): ")
	seedPhraseInput, _ := reader.ReadString('\n')
	seedPhraseInput = strings.TrimSpace(seedPhraseInput)
	var seedPhrase *string
	if seedPhraseInput != "" {
		seedPhrase = &seedPhraseInput
	}

	// Initiate Wallet
	w := initiateWallet(seedPhrase, api)

	// Get transaction amount from user
	fmt.Print("Enter number of transactions to send: ")
	var txAmount int
	_, err = fmt.Scan(&txAmount)
	if err != nil {
		log.Println("Invalid transaction amount:", err.Error())
		return
	}

	// Repeat sending transactions txAmount times.
	for txAmount > 0 && txAmount != 0 {
		log.Println("Sending transaction")
		if err := sendMessage(w, api, ctx); err != nil {
			log.Println("Error sending messages:", err.Error())
		}

		log.Println("Sent", txAmount, "transactions")
		txAmount -= 1
	}

}

func initiateWallet(seedPhrase *string, api *ton.APIClient) *wallet.Wallet {
	var words []string

	if seedPhrase == nil {
		words = wallet.NewSeed()

	} else {
		words = strings.Split(*seedPhrase, " ")
	}

	w, err := wallet.FromSeed(api, words, wallet.V4R2)
	if err != nil {
		log.Fatalln("FromSeed err:", err.Error())
		return nil
	}

	log.Println("Wallet address:", w.Address())
	log.Println("Generated seed phrase:", strings.Join(words, " "))
	return w
}

func sendMessage(w *wallet.Wallet, api *ton.APIClient, ctx context.Context) error {
	block, err := api.CurrentMasterchainInfo(context.Background())
	if err != nil {
		log.Println("CurrentMasterchainInfo err:", err.Error())
		return err
	}

	balance, err := w.GetBalance(context.Background(), block)
	if err != nil {
		log.Println("GetBalance err:", err.Error())
		return err
	}

	if balance.Nano().Uint64() < 1.5e7 {
		log.Println("Not enough balance:", balance.String(), "\nRequired minimum: 0.015 TON")
		return errors.New("not enough balance")
	}

	bounce := false
	transfer, err := w.BuildTransfer(w.WalletAddress(), tlb.MustFromTON("0"), bounce, "data:application/json,{\"p\":\"ton-20\",\"op\":\"mint\",\"tick\":\"nano\",\"amt\":\"100000000000\"}")
	if err != nil {
		log.Println("Transfer err:", err.Error())
		return err
	}

	tx, _, err := w.SendWaitTransaction(ctx, transfer)
	if err != nil {
		log.Println("SendWaitTransaction err:", err.Error())
		return err
	}

	log.Println("transaction sent, hash:", base64.StdEncoding.EncodeToString(tx.Hash))
	log.Println("explorer link: https://tonscan.org/tx/" + base64.URLEncoding.EncodeToString(tx.Hash))

	return nil
}
