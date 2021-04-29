package main

import (
	"encoding/hex"
	"flag"
	iota "github.com/GalRogozinski/iota.go/v2"
	. "github.com/iotaledger/chrysalis-tools/go-tests/lib"
	"log"
)

func main() {
	nodeDomain, apiPort := DefineNodeFlags()
	blowballSize := flag.Int("blowball", 10, "size of a single blowball")
	flag.Parse()

	nodeAPI, nodeInfo := ObtainAPI(*nodeDomain, *apiPort)

	for {
		milestoneResponse, err := nodeAPI.MilestoneByIndex(nodeInfo.LatestMilestoneIndex)
		Must(err)
		messageIdBytes, err := hex.DecodeString(milestoneResponse.MessageID)
		Must(err)
		var parent iota.MessageID
		copy(parent[:], messageIdBytes)
		parents := []iota.MessageID{parent}
		for i := 0; i < *blowballSize; i++ {
			m := SendDataMessage(nodeAPI, &nodeInfo.NetworkID, &parents, "blowball", string(i))
			id, err := m.ID()
			Must(err)
			log.Println("sent blowball message ", hex.EncodeToString(id[:]))
		}

	}
}
