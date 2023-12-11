package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

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
	for {
		for i := 0; i < 50; i++ {
			go func() {
				log.Println("Sending transaction")
				if err := sendMessage(w, api, client); err != nil {
					log.Println("Error sending messages:", err.Error())
				}
				log.Println("Sent", txAmount, "transactions")
				txAmount -= 1
			}()
		}
		time.Sleep(4 * time.Second)
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

func sendMessage(w *wallet.Wallet, api *ton.APIClient, client *liteclient.ConnectionPool) error {
	ctx := client.StickyContext(context.Background())

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

	if balance.Nano().Uint64() >= 1.4e7 {
		log.Println("sending transaction and waiting for confirmation...")

		comment, err := wallet.CreateCommentCell("data:application/json,{\"p\":\"ton-20\",\"op\":\"mint\",\"tick\":\"nano\",\"amt\":\"100000000000\"}")
		if err != nil {
			log.Println("CreateCommentCell err:", err.Error())
			return err
		}

		transfer := &wallet.Message{
			Mode: 1, // pay fee separately
			InternalMessage: &tlb.InternalMessage{
				Bounce:  false, // force send, even to uninitialized wallets
				DstAddr: w.WalletAddress(),
				Amount:  tlb.MustFromTON("0"),
				Body:    comment,
			},
		}

		err = w.SendMany(ctx, []*wallet.Message{transfer, transfer, transfer, transfer})

		if err != nil {
			log.Println("Transfer err:", err.Error())
			return nil
		}

		time.Sleep(1 * time.Second)

		log.Printf("Transaction sent")
		return nil
	}
	log.Println("not enough balance:", balance.String())
	return nil
}
