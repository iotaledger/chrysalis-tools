package main

import (
	"encoding/hex"
	"flag"
	iota "github.com/GalRogozinski/iota.go/v2"
	ed "github.com/GalRogozinski/iota.go/v2/ed25519"
	. "github.com/iotaledger/chrysalis-tools/go-tests/lib"
	"log"
	"time"
)

func main() {
	nodeDomain, apiPort := DefineNodeFlags()
	flag.Parse()

	nodeAPI, nodeInfo := ObtainAPI(*nodeDomain, *apiPort)
	nullOutput := [iota.TransactionIDLength]byte{}

	inputSeed := CreateSeed([]byte{0xde, 0xad, 0xbe, 0xef})
	privateKey, _, address1 := GenerateAddressFromSeed(inputSeed)
	log.Println("input ed_address 1 is ", hex.EncodeToString(address1[:]))
	log.Println("input bech 32_1 address is ", address1.Bech32(iota.PrefixTestnet))
	signer := iota.NewInMemoryAddressSigner(iota.AddressKeys{
		Address: &address1,
		Keys:    privateKey,
	})

	outputSeed2 := CreateSeed([]byte{0xde, 0xad, 0xbe, 0xe1})
	_, _, address2 := GenerateAddressFromSeed(outputSeed2)
	log.Println("input ed_address 2 is ", hex.EncodeToString(address2[:]))
	log.Println("input bech 32_2 address is ", address2.Bech32(iota.PrefixTestnet))

	outputSeed3 := CreateSeed([]byte{0xae, 0xad, 0xbe, 0xe1})
	_, _, address3 := GenerateAddressFromSeed(outputSeed3)
	log.Println("input ed_address 3 is ", hex.EncodeToString(address3[:]))
	log.Println("input bech 32_3 address is ", address3.Bech32(iota.PrefixTestnet))
	tx, err := iota.NewTransactionBuilder().
		AddInput(CreateInput(&address1, nullOutput, 0)).
		AddOutput(CreateOutput(&address2, 1000005000000061-10_000_000)).
		AddOutput(CreateOutput(&address3, 10_000_000)).
		AddIndexationPayload(&iota.Indexation{Index: []byte("value"), Data: []byte("test")}).
		Build(signer)
	Must(err)

	milestoneResponse, err := nodeAPI.MilestoneByIndex(nodeInfo.LatestMilestoneIndex)
	Must(err)

	parentBytes, err := hex.DecodeString(milestoneResponse.MessageID)
	Must(err)

	var parent iota.MessageID
	copy(parent[:], parentBytes)

	message1 := SendValueMessage(nodeAPI, &nodeInfo.NetworkID, &iota.MessageIDs{parent}, tx)

	tx2, err := iota.NewTransactionBuilder().
		AddInput(CreateInput(&address1, nullOutput, 0)).
		AddOutput(CreateOutput(&address2, 1000005000000061-20_000_000)).
		AddOutput(CreateOutput(&address3, 20_000_000)).
		AddIndexationPayload(&iota.Indexation{Index: []byte("value"), Data: []byte("test")}).
		Build(signer)
	Must(err)

	message2 := SendValueMessage(nodeAPI, &nodeInfo.NetworkID, &iota.MessageIDs{parent}, tx2)

	//**** Set Up Milestone ****/

	milestoneSeed := CreateSeed([]byte{0xef, 0x13, 0xab, 0xed})
	privateKey, milestonePublicKey := GenerateMilestoneKeys(milestoneSeed)
	keyMap := CreateMilestoneKeyMapping([]ed.PrivateKey{privateKey}, []iota.MilestonePublicKey{milestonePublicKey})

	id, err := message2.ID()
	Must(err)
	log.Print("message2 id is ", hex.EncodeToString((*id)[:]))
	sendMilestone(id, milestonePublicKey, keyMap, nodeAPI, nodeInfo, nodeInfo.LatestMilestoneIndex+1)

	id2, err := message1.ID()
	Must(err)
	log.Print("message1 id is ", hex.EncodeToString((*id2)[:]))
	sendMilestone(id2, milestonePublicKey, keyMap, nodeAPI, nodeInfo, nodeInfo.LatestMilestoneIndex+1)
}

func sendMilestone(parent *iota.MessageID, milestonePublicKey iota.MilestonePublicKey, keyMap iota.MilestonePublicKeyMapping, nodeAPI *iota.NodeHTTPAPIClient, info *iota.NodeInfoResponse, index uint32) {
	parents := iota.MessageIDs{*parent}
	proof := CreateMilestoneInclusionMerkleProof(parents)
	log.Print("The merkle root is ", hex.EncodeToString(proof[:]))
	milestone := CreateSignedMilestone(index, uint64(time.Now().Unix()), parents, proof, []iota.MilestonePublicKey{milestonePublicKey},
		nil, keyMap)
	message := SendMilestone(nodeAPI, &info.NetworkID, parents, milestone)
	id, err := message.ID()
	Must(err)
	log.Print("sent milestone ", hex.EncodeToString(id[:]))
}
