package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	. "github.com/iotaledger/chrysalis-tools/go-tests/lib"
	iota "github.com/iotaledger/iota.go/v2"
)

const blowballSize = 10

func main() {
	endpoint := flag.String("endpoint", "http://chrysalis-net.twilightparadox.com:14266", "endpoint")
	nodeAPI := iota.NewNodeAPIClient(*endpoint)

	for {
		nodeInfo, err := nodeAPI.Info()
		Must(err)
		milestoneResponse, err := nodeAPI.MilestoneByIndex(nodeInfo.LatestMilestoneIndex)
		Must(err)
		messageIdBytes, err := hex.DecodeString(milestoneResponse.MessageID)
		Must(err)
		var parent iota.MessageID
		copy(parent[:], messageIdBytes)
		parents := []iota.MessageID{parent}
		for i := 0; i < blowballSize; i++ {
			m := SendDataMessage(nodeAPI, &nodeInfo.NetworkID, &parents, "blowball", string(i))
			id, err := m.ID()
			Must(err)
			fmt.Println("sent blowball message ", hex.EncodeToString(id[:]))
		}

	}
}
