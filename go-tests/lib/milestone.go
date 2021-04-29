package lib

import (
	"crypto"
	"encoding"
	"encoding/hex"
	. "github.com/GalRogozinski/iota.go/v2"
	ed "github.com/GalRogozinski/iota.go/v2/ed25519"
)

const (
	MilestonePrivate = "ef13abedef13abedef13abedef13abedef13abedef13abedef13abedef13abed"
	MilestonePublic  = "6ce02bb80d2db7759863543278b3f04e5f945131077574148b1907d692f80e7c"
)

var (
	MilestoneSeed = []byte{0xef, 0x13, 0xab, 0xed}
)

func SendMilestone(api *NodeHTTPAPIClient, networkId *string, parents MessageIDs, milestone Milestone) *Message {
	message := CreateMilestoneMessage(networkId, parents, milestone)
	msg, err := api.SubmitMessage(message)
	Must(err)
	return msg
}

func CreateMilestoneMessage(networkId *string, parents MessageIDs, milestone Milestone) *Message {
	if parents == nil {
		parents = milestone.Parents
	}
	msg := Message{
		NetworkID: NetworkIDFromString(*networkId),
		Parents:   parents,
		Payload:   &milestone,
	}

	return &msg
}

func CreateSignedMilestone(index uint32, timestamp uint64, parents MilestoneParentMessageIDs, proof MilestoneInclusionMerkleProof, publicKeys []MilestonePublicKey, reciepts Serializable, keyMap MilestonePublicKeyMapping) Milestone {
	milestone := CreateMilestone(index, timestamp, parents, proof, publicKeys, reciepts, nil)
	essence, err := milestone.Essence()
	Must(err)
	milestone.Signatures = SignMilstoneEssence(keyMap, publicKeys, essence)
	return milestone
}

func CreateMilestone(index uint32, timestamp uint64, parents MilestoneParentMessageIDs, proof MilestoneInclusionMerkleProof, publicKeys []MilestonePublicKey, reciepts Serializable, sigs []MilestoneSignature) Milestone {
	return Milestone{
		Index:                index,
		Timestamp:            timestamp,
		Parents:              parents,
		InclusionMerkleProof: proof,
		PublicKeys:           publicKeys,
		Receipt:              reciepts,
		Signatures:           sigs,
	}
}

func CreateMilestoneInclusionMerkleProof(msgIds MessageIDs) MilestoneInclusionMerkleProof {
	hasher := NewHasher(crypto.BLAKE2b_256)
	marshalers := make([]encoding.BinaryMarshaler, len(msgIds))
	for i, msgId := range msgIds {
		var marshId = make(MarshableID, 32)
		copy(marshId, msgId[:])
		marshalers[i] = marshId
	}
	hash, err := hasher.Hash(marshalers)
	Must(err)
	proof := MilestoneInclusionMerkleProof{}
	copy(proof[:], hash)
	return proof
}

func CreateMilestoneKeyMapping(privateKeys []ed.PrivateKey, publicKeys []MilestonePublicKey) MilestonePublicKeyMapping {
	if len(privateKeys) != len(publicKeys) {
		panic("private and public keys should be of the same length")
	}
	milestonePublicKeyMapping := MilestonePublicKeyMapping{}
	for i := range publicKeys {
		milestonePublicKeyMapping[publicKeys[i]] = privateKeys[i]
	}

	return milestonePublicKeyMapping
}

func CreateMilestoneKeyStringMapping(privateKeys []string, publicKeys []string) MilestonePublicKeyMapping {
	privateArr := []ed.PrivateKey{}
	for _, key := range privateKeys {
		decodeKey, err := hex.DecodeString(key)
		Must(err)
		privateArr = append(privateArr, decodeKey)
	}

	publicArr := []MilestonePublicKey{}
	for _, key := range publicKeys {
		decodeKey, err := hex.DecodeString(key)
		Must(err)
		pubKey := MilestonePublicKey{}
		copy(pubKey[:], decodeKey)
		publicArr = append(publicArr, pubKey)
	}

	return CreateMilestoneKeyMapping(privateArr, publicArr)
}

func FlattenKeyMap(milestoneKeyMapping MilestonePublicKeyMapping) ([]ed.PrivateKey, []MilestonePublicKey) {
	var milestonePrivateKeys []ed.PrivateKey
	var milestonePublicKeys []MilestonePublicKey
	for k, v := range milestoneKeyMapping {
		milestonePrivateKeys = append(milestonePrivateKeys, v)
		milestonePublicKeys = append(milestonePublicKeys, k)
	}

	return milestonePrivateKeys, milestonePublicKeys
}

func SignMilestone(keyMap MilestonePublicKeyMapping, pubKeys []MilestonePublicKey, milestone Milestone) []MilestoneSignature {
	essence, err := milestone.Essence()
	Must(err)
	return SignMilstoneEssence(keyMap, pubKeys, essence)
}

func GenerateMilestoneKeys(seed []byte) (ed.PrivateKey, MilestonePublicKey) {
	privateKey := ed.NewKeyFromSeed(seed)
	publicKey := MilestonePublicKey{}
	copy(publicKey[:], privateKey[32:])
	return privateKey, publicKey
}

func SignMilstoneEssence(keyMap MilestonePublicKeyMapping, pubKeys []MilestonePublicKey, essence []byte) []MilestoneSignature {
	signer := InMemoryEd25519MilestoneSigner(keyMap)
	signatures, err := signer(pubKeys, essence)
	Must(err)
	return signatures
}
