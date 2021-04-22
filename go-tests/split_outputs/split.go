package main

import (
	"encoding/binary"
	"encoding/hex"
	"flag"
	iota "github.com/GalRogozinski/iota.go/v2"
	"github.com/GalRogozinski/iota.go/v2/bech32"
	. "github.com/iotaledger/chrysalis-tools/go-tests/lib"
	"log"
)

func main() {
	nodeDomain, apiPort := DefineNodeFlags()
	amount := flag.Uint64("amount", 7_936_447_619_000, "How much iotas should be split to each output")
	flag.Parse()
	nodeAPI, nodeInfo := ObtainAPI(*nodeDomain, *apiPort)

	SplitFunds(nodeAPI, nodeInfo, amount)
}

func SplitFunds(nodeAPI *iota.NodeHTTPAPIClient, nodeInfo *iota.NodeInfoResponse, amount *uint64) []KeyTriplet {
	seed := CreateSeed([]byte{0xde, 0xad, 0xbe, 0xef})
	privateKey, _, address1 := GenerateAddressFromSeed(seed)
	signer := iota.NewInMemoryAddressSigner(iota.AddressKeys{
		Address: &address1,
		Keys:    privateKey,
	})
	txBuilder := iota.NewTransactionBuilder().
		AddInput(CreateInput(&address1, GenesisOutput, 0)).
		AddIndexationPayload(&iota.Indexation{Index: []byte("split"), Data: []byte("126")})

	keyTriplets := GenerateKeyChain(126, 0)
	for i := 0; i < len(keyTriplets); i++ {
		txBuilder.AddOutput(CreateOutput(&keyTriplets[i].Address, *amount))
	}

	var last_bal uint64 = 1_000_005_000_000_061 - (*amount * 126)
	buf := make([]byte, 4)
	binary.PutUvarint(buf, uint64(127))
	seed = CreateSeed(buf)
	_, _, address := GenerateAddressFromSeed(seed)
	tx, err := txBuilder.AddOutput(CreateOutput(&address, last_bal)).
		Build(signer)
	Must(err)

	SendValueMessage(nodeAPI, &nodeInfo.NetworkID, nil, tx)
	bech, err := bech32.Encode("atoi", address[:])
	Must(err)
	log.Println(" last address is ", hex.EncodeToString(address[:]), " ", bech)
	log.Println("Last remainder balance is ", last_bal)
	transactionID, err := tx.ID()
	Must(err)
	log.Println("tx has is ", hex.EncodeToString(transactionID[:]))

	return keyTriplets
}
