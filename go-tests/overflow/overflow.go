package main

import (
	"encoding/binary"
	"encoding/hex"
	"flag"
	iota "github.com/GalRogozinski/iota.go/v2"
	. "github.com/iotaledger/chrysalis-tools/go-tests/lib"
	"log"
	"math"
)

func main() {
	nodeDomain, apiPort := DefineNodeFlags()
	flag.Parse()

	nodeAPI, nodeInfo := ObtainAPI(*nodeDomain, *apiPort)

	buf := make([]byte, 4)
	binary.PutUvarint(buf, uint64(127))
	seed := CreateSeed(buf)
	privateKey, _, address1 := GenerateAddressFromSeed(seed)
	signer := iota.NewInMemoryAddressSigner(iota.AddressKeys{
		Address: &address1,
		Keys:    privateKey,
	})

	balanceResponse, err := nodeAPI.BalanceByEd25519Address(&address1)
	Must(err)
	outputResponse, err := nodeAPI.OutputIDsByEd25519Address(&address1, false)
	Must(err)
	outputId := outputResponse.OutputIDs[0]
	idHex := outputId[:iota.TransactionIDLength*2]
	txIdSlice, err := hex.DecodeString(string(idHex))
	var txId [32]byte
	copy(txId[:], txIdSlice)
	Must(err)
	index := outputId[iota.TransactionIDLength*2:]
	outputIndexBytes, err := hex.DecodeString(string(index))
	outputIndex := binary.LittleEndian.Uint16(outputIndexBytes)
	Must(err)

	outputSeed2 := CreateSeed([]byte{0xce, 0xad, 0xbe, 0xe1})
	_, _, address2 := GenerateAddressFromSeed(outputSeed2)

	tx, err := iota.NewTransactionBuilder().
		AddInput(CreateInput(&address1, txId, outputIndex)).
		AddIndexationPayload(&iota.Indexation{Index: []byte("overflow"), Data: []byte("127")}).
		AddOutput(CreateOutput(&address1, math.MaxUint64)).
		AddOutput(CreateOutput(&address2, balanceResponse.Balance)).
		Build(signer)
	Must(err)

	SendValueMessage(nodeAPI, &nodeInfo.NetworkID, nil, tx)
	log.Println(" addresses are ", hex.EncodeToString(address1[:]), " ", hex.EncodeToString(address2[:]))
}
