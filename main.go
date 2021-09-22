package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"

	"contractDemo/conAbi"

	"github.com/ethereum/go-ethereum/common"
	cr "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	endPoint   = "http://18.144.29.246:8110"
	privateKey = "e484c4373db5c55a9813e4abbb74a15edd794019b8db4365a876ed538622bcf9"
	address    = "0xA4E7663A031ca1f67eEa828E4795653504d38c6e"
)

func main() {
	client, err := ethclient.Dial(endPoint)
	if err != nil {
		log.Fatal(err)
	}

	chainID, err := client.ChainID(context.Background())

	if err != nil {
		log.Fatal(err)
	}

	pk, err := cr.HexToECDSA(privateKey)
	if err != nil {
		log.Fatal(err)
	}

	publicKey := pk.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("error casting public key to ECDSA")
	}

	fromAddress := cr.PubkeyToAddress(*publicKeyECDSA)

	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		log.Fatal(err)
	}

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	if conAbi.DebugFlag {
		fmt.Println("chainid is ", chainID)
		fmt.Println("from address is ", fromAddress)
		fmt.Println("nonce is ", nonce)
		fmt.Println("gasPrice is ", gasPrice)
	}

	//get address balance
	balance, err := conAbi.GetBalance(context.Background(), fromAddress, client)
	if err != nil {
		log.Fatal(err)
	}
	if conAbi.DebugFlag {
		fmt.Println("balance is ", balance)
	}

	//txHash, err :=
	add, err := conAbi.ERC20Address(context.Background(), client)
	if err != nil {
		log.Fatal(err)
	}
	if conAbi.DebugFlag {
		fmt.Println("address is ", add)

		//deploy contract
		fmt.Println("sender addr is ", conAbi.SenderAdd)
	}

	swapAdd := common.HexToAddress("0xC721594D255Aa52B442a67603593673646835759")

	//singer := transaction.NewDefaultSigner(pk)

	//已经部署一套swap合约 0xC721594D255Aa52B442a67603593673646835759
	/*
		txhash, err := conAbi.Deploy(context.Background(), conAbi.SenderAdd, big.NewInt(100), common.BigToHash(big.NewInt(100)), client, singer, chainID)

		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("txhash is ", txhash)

		time.Sleep(time.Duration(20) * time.Second)

		swapAdd, err := conAbi.WaitDeployed(context.Background(), txhash, client)

		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("swapAdd is ", swapAdd)
	*/

	//get swap.balance
	swapBalance, err := conAbi.GetSwapBalance(context.Background(), swapAdd, client, swapAdd)

	if err != nil {
		log.Fatal(err)
	}
	if conAbi.DebugFlag {
		fmt.Println("swapBalance is ", swapBalance)

		//VerifyFactoryBytecode
		fmt.Println("begin verify factory contract code")
	}
	err = conAbi.VerifyFactoryBytecode(context.Background(), client)
	if err != nil {
		log.Fatal(err)
	} else {
		log.Println("verify factory ok")
	}

	if conAbi.DebugFlag {
		fmt.Println("verify chequebook ")
	}

	err = conAbi.VerifyChequebook(context.Background(), swapAdd, client)

	if err != nil {
		log.Fatal(err)
	} else {
		log.Println("verify chequebook ok")
	}
	/*

		if conAbi.DebugFlag {
			fmt.Println("run preWithdraw")
		}

		withdrawTime1, err := conAbi.GetWithdrawTime(context.Background(), client, swapAdd)

		if err != nil {
			log.Fatal(err)
		}

		if conAbi.DebugFlag {
			fmt.Println("withdrawTime1 is ", withdrawTime1)
		}

		txHash, err := conAbi.PreWithdraw(context.Background(), fromAddress, client, swapAdd, singer, chainID)

		if err != nil {
			log.Fatal(err)
		}

		if conAbi.DebugFlag {
			fmt.Println("PreWithdraw tx hash is  ", txHash)
		}

		time.Sleep(time.Duration(20) * time.Second)

		withdrawTime2, err := conAbi.GetWithdrawTime(context.Background(), client, swapAdd)

		if err != nil {
			log.Fatal(err)
		}

		if conAbi.DebugFlag {
			fmt.Println("withdrawTime2 is ", withdrawTime2)
			fmt.Println("withdrawTime2 is ", time.Unix(withdrawTime2.Int64(), 0))
		}
	*/

	if conAbi.DebugFlag {
		fmt.Println("run master proxy")
	}

	proxyAdd := common.HexToAddress("0x6936a4893e4d83bad993848a21bae606c15480f1")

	masterAdd, err := conAbi.GetMasterCopy(context.Background(), client, proxyAdd)

	if err != nil {
		log.Fatal(err)
	}

	if conAbi.DebugFlag {
		fmt.Println("masterAdd is ", masterAdd)
	}

	if conAbi.DebugFlag {
		fmt.Println("run oracle")
	}

	oracleAdd := common.HexToAddress("0xFB6a65aF1bb250EAf3f58C420912B0b6eA05Ea7a")

	owner, err := conAbi.GetOwner(context.Background(), client, oracleAdd)

	if err != nil {
		log.Fatal(err)
	}

	if conAbi.DebugFlag {
		fmt.Println("owner is ", owner)
	}

	if conAbi.DebugFlag {
		fmt.Println("run oracle get price")
	}

	price, err := conAbi.GetPrice(context.Background(), client, oracleAdd)

	if err != nil {
		log.Fatal(err)
	}

	if conAbi.DebugFlag {
		fmt.Println("price is ", price)
	}

}
