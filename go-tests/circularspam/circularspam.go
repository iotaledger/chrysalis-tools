package main

import (
	"encoding/hex"
	"flag"
	iota "github.com/GalRogozinski/iota.go/v2"
	. "github.com/iotaledger/chrysalis-tools/go-tests/lib"
	"log"
	"math/rand"
	"sync"
)

type futInput = struct {
	Address iota.Ed25519Address
	amount  uint64
}

func main() {
	nodeDomain, apiPort := DefineNodeFlags()
	amount := flag.Uint64("amount", 7_936_447_619_000, "How much iotas should be split to each output")
	flag.Parse()
	nodeAPI, nodeInfo := ObtainAPI(*nodeDomain, *apiPort)

	keyTriplets := SplitFunds(nodeAPI, nodeInfo, amount)
	signer := CreateSigner(keyTriplets)

	tripMap := make(map[string]struct{})
	for _, trip := range keyTriplets {
		tripMap[trip.Address.Bech32("atoi")] = struct{}{}
	}

	if len(tripMap) != len(keyTriplets) {
		log.Println("different lengths")
	} else {
		log.Println("same length ", len(tripMap))
	}

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go conflictingSpam(nodeAPI, nodeInfo, keyTriplets, signer)
	go circularSpam(nodeAPI, nodeInfo, keyTriplets, signer)
	go outOfOrderSpam(nodeAPI, nodeInfo, keyTriplets, signer)
	go invalidSpam(nodeAPI, nodeInfo, keyTriplets, signer)
	wg.Wait()
}

func invalidSpam(api *iota.NodeHTTPAPIClient, info *iota.NodeInfoResponse, triplets []KeyTriplet, signer iota.AddressSigner) {
	//for each address
	for i := 0; i < len(triplets)-1; i++ {
		txBuilder := iota.NewTransactionBuilder()
		//query outputs in address
		addr := &triplets[i].Address
		outputResp, _, err := api.OutputsByEd25519Address(addr, false)
		tripletCopy := make([]KeyTriplet, len(triplets))
		copy(tripletCopy, triplets)
		tripletCopy = append(tripletCopy[:i], tripletCopy[i+1:]...)
		Must(err)
		//transfer outputs to a different address
		for j, outputId := range outputResp.OutputIDs {
			txBuilder = sendAndResetBuilder(api, info, j, txBuilder, signer, "Sending out invalid spam ")
			outputByID, err := api.OutputByID(outputId.MustAsUTXOInput().ID())
			Must(err)
			txId, err := outputByID.TxID()
			Must(err)
			txBuilder.AddInput(CreateInput(addr, *txId, outputByID.OutputIndex))
			amount := rand.Intn(4236899)
			change := rand.Intn(amount)
			var targetAddress iota.Ed25519Address
			targetAddress, tripletCopy = randAddress(tripletCopy)
			txBuilder.AddOutput(CreateOutput(&targetAddress, uint64(amount-change)))
			var targetAddress2 iota.Ed25519Address
			targetAddress2, tripletCopy = randAddress(tripletCopy)
			txBuilder.AddOutput(CreateOutput(&targetAddress2, uint64(change)))
		}
		sendTransaction(api, info, txBuilder, signer, "Sending out invalid spam ")
	}

	go invalidSpam(api, info, triplets, signer)
}

func sendAndResetBuilder(api *iota.NodeHTTPAPIClient, info *iota.NodeInfoResponse, i int, txBuilder *iota.TransactionBuilder, signer iota.AddressSigner, logMsg string) *iota.TransactionBuilder {
	if i != 0 && i%125 == 0 {
		sendTransaction(api, info, txBuilder, signer, logMsg)
		txBuilder = iota.NewTransactionBuilder()
	}
	return txBuilder
}

func sendTransaction(api *iota.NodeHTTPAPIClient, info *iota.NodeInfoResponse, txBuilder *iota.TransactionBuilder, signer iota.AddressSigner, logMsg string) {
	tx, err := txBuilder.Build(signer)
	Must(err)
	message := SendValueMessage(api, &info.NetworkID, nil, tx)
	id := message.MustID()
	log.Println(logMsg, hex.EncodeToString(id[:]))
}

func randAddress(tripletCopy []KeyTriplet) (iota.Ed25519Address, []KeyTriplet) {
	randIndex := rand.Intn(len(tripletCopy))
	targetAddress := tripletCopy[randIndex].Address
	tripletCopy = append(tripletCopy[:randIndex], tripletCopy[randIndex+1:]...)
	log.Printf("target address %s   index %d", targetAddress.Bech32("atoi"), randIndex)
	return targetAddress, tripletCopy
}

func outOfOrderSpam(api *iota.NodeHTTPAPIClient, info *iota.NodeInfoResponse, triplets []KeyTriplet, signer iota.AddressSigner) {
	size := len(triplets) - 1

	var futureIn []futInput
	//for each address
	for i := 0; i < size; i++ {
		//query outputs in address
		addr := &triplets[i].Address
		tripletCopy := make([]KeyTriplet, len(triplets))
		copy(tripletCopy, triplets)
		tripletCopy = append(tripletCopy[:i], tripletCopy[i+1:]...)
		outputResp, _, err := api.OutputsByEd25519Address(addr, false)
		Must(err)
		txBuilder := iota.NewTransactionBuilder()
		//transfer outputs to a different address
		var changeAccumulator uint64 = 0
		for i, outputId := range outputResp.OutputIDs {
			if i == 125 {
				break
			}
			outputByID, err := api.OutputByID(outputId.MustAsUTXOInput().ID())
			Must(err)
			output, err := outputByID.Output()
			Must(err)
			txId, err := outputByID.TxID()
			Must(err)
			txBuilder.AddInput(CreateInput(addr, *txId, outputByID.OutputIndex))
			var targetAddress iota.Ed25519Address
			targetAddress, tripletCopy = randAddress(tripletCopy)
			amount, err := output.Deposit()
			Must(err)
			change := amount - uint64(rand.Int63n(int64(amount)))
			txBuilder.AddOutput(CreateOutput(&targetAddress, amount-change))
			futureIn = append(futureIn, futInput{Address: targetAddress, amount: amount - change})
			changeAccumulator += change
		}
		tx, err := txBuilder.AddOutput(CreateOutput(addr, changeAccumulator)).
			Build(signer)
		Must(err)

		tripletCopy = make([]KeyTriplet, len(triplets))
		copy(tripletCopy, triplets)
		txBuilder = iota.NewTransactionBuilder()
		for index, input := range futureIn {
			if index == 125 {
				break
			}
			log.Print("triplet copy len is ", len(tripletCopy))
			id, err := tx.ID()
			Must(err)
			var targetAddress iota.Ed25519Address
			targetAddress, tripletCopy = randAddress(tripletCopy)
			txBuilder.AddInput(CreateInput(&input.Address, *id, uint16(index))).
				AddOutput(CreateOutput(&targetAddress, input.amount))
		}
		tx2, err := txBuilder.Build(signer)
		Must(err)
		message2 := SendValueMessage(api, &info.NetworkID, nil, tx2)
		id2 := message2.MustID()
		message1 := SendValueMessage(api, &info.NetworkID, &iota.MessageIDs{id2}, tx)
		id1 := message1.MustID()
		log.Println("Sending out of order spam ", hex.EncodeToString(id1[:]), " ", hex.EncodeToString(id2[:]))
	}

	go outOfOrderSpam(api, info, triplets, signer)
}

func CreateSigner(triplets []KeyTriplet) iota.AddressSigner {
	var keys []iota.AddressKeys
	for i := 0; i < len(triplets); i++ {
		addressKeys := iota.AddressKeys{
			Address: &triplets[i].Address,
			Keys:    triplets[i].Sk,
		}
		keys = append(keys, addressKeys)
	}
	return iota.NewInMemoryAddressSigner(keys...)
}

func circularSpam(api *iota.NodeHTTPAPIClient, info *iota.NodeInfoResponse, triplets []KeyTriplet, signer iota.AddressSigner) {
	size := len(triplets) - 1
	//for each address
	for i := 0; i < size; i++ {
		txBuilder := iota.NewTransactionBuilder()
		//query outputs in address
		addr := triplets[i].Address
		outputResp, _, err := api.OutputsByEd25519Address(&addr, false)
		tripletsCopy := make([]KeyTriplet, size)
		copy(tripletsCopy, triplets)
		tripletsCopy = append(tripletsCopy[:i], tripletsCopy[i+1:]...)
		Must(err)
		//transfer outputs to a different address
		randomTx(api, outputResp, txBuilder, addr, tripletsCopy)
		tx, err := txBuilder.Build(signer)
		Must(err)
		message := SendValueMessage(api, &info.NetworkID, nil, tx)
		id := message.MustID()
		log.Println("circular spam message sent ", hex.EncodeToString(id[:]))
	}

	go circularSpam(api, info, triplets, signer)
}

func conflictingSpam(api *iota.NodeHTTPAPIClient, info *iota.NodeInfoResponse, triplets []KeyTriplet, signer iota.AddressSigner) {
	//for each address
	for i := 0; i < len(triplets)-1; i++ {
		tripletsCopy := make([]KeyTriplet, len(triplets))
		//query outputs in address
		addr := triplets[i].Address
		log.Printf("main address %s   index %d", addr.Bech32("atoi"), i)
		outputResp, _, err := api.OutputsByEd25519Address(&addr, false)
		copied := copy(tripletsCopy, triplets)
		log.Printf("%d elements copied", copied)
		tripletsCopy = append(tripletsCopy[:i], tripletsCopy[i+1:]...)
		Must(err)

		//transfer outputs to a different address
		txBuilder := iota.NewTransactionBuilder()
		randomTx(api, outputResp, txBuilder, addr, tripletsCopy)
		tx, err := txBuilder.Build(signer)
		Must(err)
		message := SendValueMessage(api, &info.NetworkID, nil, tx)
		id := message.MustID()

		txBuilder = iota.NewTransactionBuilder()
		randomTx(api, outputResp, txBuilder, addr, tripletsCopy)
		tx, err = txBuilder.Build(signer)
		Must(err)
		message2 := SendValueMessage(api, &info.NetworkID, nil, tx)
		id2 := message2.MustID()
		log.Println("conflicting messages sent ", hex.EncodeToString(id[:]), " ", hex.EncodeToString(id2[:]))
	}

	go conflictingSpam(api, info, triplets, signer)
}

func randomTx(api *iota.NodeHTTPAPIClient, outputResp *iota.AddressOutputsResponse, txBuilder *iota.TransactionBuilder, addr iota.Ed25519Address, triplets []KeyTriplet) {
	var changeAggregator uint64 = 0

	tripletCopy := make([]KeyTriplet, len(triplets))
	copy(tripletCopy, triplets)
	for _, outputId := range outputResp.OutputIDs {
		outputByID, err := api.OutputByID(outputId.MustAsUTXOInput().ID())
		Must(err)
		output, err := outputByID.Output()
		Must(err)
		txId, err := outputByID.TxID()
		Must(err)
		txBuilder.AddInput(CreateInput(&addr, *txId, outputByID.OutputIndex))
		var targetAddress iota.Ed25519Address
		targetAddress, tripletCopy = randAddress(tripletCopy)
		copyAddress := targetAddress

		amount, err := output.Deposit()
		Must(err)
		change := amount - uint64(rand.Int63n(int64(amount)))
		txBuilder.AddOutput(CreateOutput(&copyAddress, amount-change))
		changeAggregator += change
	}
	txBuilder.AddOutput(CreateOutput(&addr, changeAggregator))
}
