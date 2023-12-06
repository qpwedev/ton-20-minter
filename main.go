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

	"github.com/xssnick/tonutils-go/address"
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
	for txAmount > 0 && txAmount != 0 {
		var txAmountToSend int

		if txAmount/100 > 0 {
			txAmountToSend = 100
		} else {
			txAmountToSend = txAmount
		}

		messages, err := formSendMessages(txAmountToSend)
		if err != nil {
			log.Println("Error forming messages:", err.Error())
			return
		}

		log.Println("Sending", txAmountToSend, "transactions")
		if err := sendMessages(w, messages, api); err != nil {
			log.Println("Error sending messages:", err.Error())
		}

		log.Println("Sent", txAmountToSend, "transactions")
		txAmount -= txAmountToSend
	}

}

func initiateWallet(seedPhrase *string, api *ton.APIClient) *wallet.Wallet {
	var words []string

	if seedPhrase == nil {
		words = wallet.NewSeed()

	} else {
		words = strings.Split(*seedPhrase, " ")
	}

	w, err := wallet.FromSeed(api, words, wallet.HighloadV2R2)
	if err != nil {
		log.Fatalln("FromSeed err:", err.Error())
		return nil
	}

	log.Println("Wallet address:", w.Address())
	log.Println("Generated seed phrase:", strings.Join(words, " "))
	return w
}

func formSendMessages(txAmount int) ([]*wallet.Message, error) {
	var receivers []Receiver

	for i := 0; i < txAmount; i++ {
		receivers = append(receivers, Receiver{
			Address: "EQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAM9c",
			Amount:  "0",
		})
	}

	comment, err := wallet.CreateCommentCell("data:application/json,{\"p\":\"ton-20\",\"op\":\"mint\",\"tick\":\"nano\",\"amt\":\"100000000000\"}")
	if err != nil {
		log.Fatalln("CreateComment err:", err.Error())
		return nil, err
	}

	var messages []*wallet.Message

	for _, receiver := range receivers {
		messages = append(messages, &wallet.Message{
			Mode: 1, // pay fee separately
			InternalMessage: &tlb.InternalMessage{
				Bounce:  false,
				DstAddr: address.MustParseAddr(receiver.Address),
				Amount:  tlb.MustFromTON(receiver.Amount),
				Body:    comment,
			},
		})
	}

	return messages, nil
}

func sendMessages(w *wallet.Wallet, messages []*wallet.Message, api *ton.APIClient) error {
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

	if balance.Nano().Uint64() < 6e8 {
		log.Println("Not enough balance:", balance.String(), "\nRequired minimum: 0.6 TON")
		return errors.New("not enough balance")
	}

	txHash, err := w.SendManyWaitTxHash(context.Background(), messages)
	if err != nil {
		log.Println("Transfer err:", err.Error())
		return err
	}

	log.Println("transaction sent, hash:", base64.StdEncoding.EncodeToString(txHash))
	log.Println("explorer link: https://tonscan.org/tx/" + base64.URLEncoding.EncodeToString(txHash))

	return nil
}
