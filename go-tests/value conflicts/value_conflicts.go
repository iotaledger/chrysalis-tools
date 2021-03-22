package main

import (
	"flag"
	. "github.com/iotaledger/chrysalis-tools/go-tests/lib"
	iota "github.com/iotaledger/iota.go/v2"
)

func main() {
	nodeDomain, apiPort := DefineNodeFlags()
	flag.Parse()

	nodeAPI, nodeInfo := ObtainAPI(*nodeDomain, *apiPort)

	genesisAddress := iota.Ed25519Address{}
	genesisOutput := [iota.TransactionIDLength]byte{}

	seed := CreateSeed([]byte{0xde, 0xad, 0xbe, 0xef})
	privateKey, _, address1 := GenerateAddressFromSeed(seed)
	signer := iota.NewInMemoryAddressSigner(iota.AddressKeys{
		Address: &address1,
		Keys:    &privateKey,
	})

	tx, err := iota.NewTransactionBuilder().
		AddInput(CreateInput(&genesisAddress, genesisOutput, 0)).
		AddOutput(CreateOutput(&address1, 1_500_000)).
		AddIndexationPayload(&iota.Indexation{Index: []byte("value"), Data: []byte("test")}).
		Build(signer)
	Must(err)

	SendValueMessage(nodeAPI, &nodeInfo.NetworkID, nil, tx)

	tx, err = iota.NewTransactionBuilder().
		AddInput(CreateInput(&genesisAddress, genesisOutput, 0)).
			AddOutput(CreateOutput(&address1, 1_800_000)).
		AddIndexationPayload(&iota.Indexation{Index: []byte("value"), Data: []byte("test")}).
		Build(signer)
	Must(err)

	SendValueMessage(nodeAPI, &nodeInfo.NetworkID, nil, tx)
}
