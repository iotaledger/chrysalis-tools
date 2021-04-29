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

	genesisAddress := iota.Ed25519Address{}
	genesisOutput := [iota.TransactionIDLength]byte{}

	//nullPk := ed.PrivateKey(make([]byte, 64))

	seed := CreateSeed([]byte{0xde, 0xad, 0xbe, 0xef})
	sk, _, address1 := GenerateAddressFromSeed(seed)
	log.Printf("address %s", address1.Bech32("atoi"))

	addressBalance, err := nodeAPI.BalanceByEd25519Address(&address1)
	Must(err)
	log.Printf("balance at genesis address is %d", addressBalance.Balance)

	signer := iota.NewInMemoryAddressSigner(iota.AddressKeys{
		Address: &address1,
		Keys:    sk,
	})

	tx, err := iota.NewTransactionBuilder().
		AddInput(CreateInput(&address1, genesisOutput, 0)).
		AddOutput(CreateOutput(&genesisAddress, addressBalance.Balance)).
		AddIndexationPayload(&iota.Indexation{Index: []byte("value"), Data: []byte("test")}).
		Build(signer)
	Must(err)

	SendValueMessage(nodeAPI, &nodeInfo.NetworkID, nil, tx)

	time.Sleep(time.Second * 10)
	addressBalance, err = nodeAPI.BalanceByEd25519Address(&genesisAddress)
	Must(err)
	log.Printf("balance at genesis address is %d", addressBalance.Balance)

	addressBalance, err = nodeAPI.BalanceByEd25519Address(&address1)
	Must(err)
	log.Printf("balance at %s address is %d", address1.String(), addressBalance.Balance)

}
