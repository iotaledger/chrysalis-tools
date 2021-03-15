package main

import (
	"flag"
	"fmt"
	. "github.com/iotaledger/chrysalis-tools/go-tests/lib"
	iota "github.com/iotaledger/iota.go/v2"
)

const (
	nodeUrl = LocalHost
	apiPort = ApiPort
)

func main() {
	endpoint := flag.String("endpoint", fmt.Sprintf("http://%s:%d", nodeUrl, apiPort), "endpoint")
	nodeAPI := iota.NewNodeAPIClient(*endpoint)
	info, err := nodeAPI.Info()
	Must(err)

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
		AddOutput(CreateOutput(&address1, 825)).
		AddIndexationPayload(&iota.Indexation{Index: []byte("value"), Data: []byte("test")}).
		Build(signer)
	Must(err)

	SendValueMessage(nodeAPI, &info.NetworkID, nil, tx)

	tx, err = iota.NewTransactionBuilder().
		AddInput(CreateInput(&genesisAddress, genesisOutput, 0)).
		AddOutput(CreateOutput(&address1, 825)).
		AddIndexationPayload(&iota.Indexation{Index: []byte("value"), Data: []byte("test")}).
		Build(signer)
	Must(err)

}
