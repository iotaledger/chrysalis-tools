package main

import (
	"flag"
	iota "github.com/GalRogozinski/iota.go/v2"
	. "github.com/iotaledger/chrysalis-tools/go-tests/lib"
	"log"
	"time"
)

func main() {
	nodeDomain, apiPort := DefineNodeFlags()
	flag.Parse()

	nodeAPI, nodeInfo := ObtainAPI(*nodeDomain, *apiPort)

	//genesisAddress := iota.Ed25519Address{}
	genesisOutput := [iota.TransactionIDLength]byte{}

	seed := CreateSeed([]byte{0xde, 0xad, 0xbe, 0xef})
	privateKey, _, address1 := GenerateAddressFromSeed(seed)
	signer := iota.NewInMemoryAddressSigner(iota.AddressKeys{
		Address: &address1,
		Keys:    privateKey,
	})

	seed = CreateSeed([]byte{0xde, 0xad, 0xbe, 0xea})
	_, _, address2 := GenerateAddressFromSeed(seed)

	seed = CreateSeed([]byte{0xae, 0xad, 0xbe, 0xea})
	_, _, address3 := GenerateAddressFromSeed(seed)

	balanceResp, err := nodeAPI.BalanceByEd25519Address(&address1)
	Must(err)
	amount := balanceResp.Balance

	tx, err := iota.NewTransactionBuilder().
		AddInput(CreateInput(&address1, genesisOutput, 0)).
		AddOutput(CreateOutput(&address2, amount)).
		AddIndexationPayload(&iota.Indexation{Index: []byte("value"), Data: []byte("test")}).
		Build(signer)
	Must(err)

	SendValueMessage(nodeAPI, &nodeInfo.NetworkID, nil, tx)
	log.Println("sent value to ", address2.Bech32("atoi"))

	tx, err = iota.NewTransactionBuilder().
		AddInput(CreateInput(&address1, genesisOutput, 0)).
		AddOutput(CreateOutput(&address3, amount)).
		AddIndexationPayload(&iota.Indexation{Index: []byte("value"), Data: []byte("test")}).
		Build(signer)
	Must(err)

	SendValueMessage(nodeAPI, &nodeInfo.NetworkID, nil, tx)
	log.Println("sent value to ", address3.Bech32("atoi"))

	time.Sleep(11 * time.Second)
	addressResp, err := nodeAPI.BalanceByEd25519Address(&address2)
	Must(err)
	log.Printf("balance in %s is %d", address2.Bech32("atoi"), addressResp.Balance)

	addressResp, err = nodeAPI.BalanceByEd25519Address(&address3)
	Must(err)
	log.Printf("balance in %s is %d", address3.Bech32("atoi"), addressResp.Balance)
}
