package main

import (
	. "github.com/iotaledger/chrysalis-tools/go-tests/lib"
	iota "github.com/iotaledger/iota.go/v2"
	ed "github.com/iotaledger/iota.go/v2/ed25519"
	"time"
)

const (
	nodeUrl = LocalHost
	apiPort = ApiPort
)

func main() {
	nodeAPI, info := ObtainAPI(nodeUrl, apiPort)

	//TODO specify correct genesis address and output
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

	message1 := SendValueMessage(nodeAPI, &info.NetworkID, &iota.MessageIDs{iota.MessageID{}}, tx)

	tx, err = iota.NewTransactionBuilder().
		AddInput(CreateInput(&genesisAddress, genesisOutput, 0)).
		AddOutput(CreateOutput(&address1, 700)).
		AddIndexationPayload(&iota.Indexation{Index: []byte("value"), Data: []byte("test")}).
		Build(signer)
	Must(err)

	message2 := SendValueMessage(nodeAPI, &info.NetworkID, &iota.MessageIDs{iota.MessageID{}}, tx)

	//**** Set Up Milestone ****/

	milestoneSeed := CreateSeed([]byte{0xef, 0x13, 0xab, 0xed})
	privateKey, milestonePublicKey := GenerateMilestoneKeys(milestoneSeed)
	keyMap := CreateMilestoneKeyMapping([]ed.PrivateKey{privateKey}, []iota.MilestonePublicKey{milestonePublicKey})

	id, err := message2.ID()
	Must(err)
	sendMilestone(id, milestonePublicKey, keyMap, nodeAPI, info)

	id, err = message1.ID()
	Must(err)
	sendMilestone(id, milestonePublicKey, keyMap, nodeAPI, info)

}

func sendMilestone(id *iota.MessageID, milestonePublicKey iota.MilestonePublicKey, keyMap iota.MilestonePublicKeyMapping, nodeAPI *iota.NodeAPIClient, info *iota.NodeInfoResponse) {
	parents := iota.MessageIDs{*id}
	proof := CreateMilestoneInclusionMerkleProof(parents)
	milestone := CreateSignedMilestone(2, uint64(time.Now().Unix()), parents, proof, []iota.MilestonePublicKey{milestonePublicKey},
		nil, keyMap)
	SendMilestone(nodeAPI, &info.NetworkID, parents, milestone)
}
