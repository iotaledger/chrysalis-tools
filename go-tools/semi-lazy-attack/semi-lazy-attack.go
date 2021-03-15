package main

import (
	"flag"
	"fmt"
	"github.com/eclipse/paho.mqtt.golang"
	. "github.com/iotaledger/chrysalis-tools/go-tests/lib"
	iota "github.com/iotaledger/iota.go/v2"
	"golang.org/x/crypto/blake2b"
)

const semLazyExplosionSize = 10

var nodeApi *iota.NodeAPIClient
var msQueue []uint32
var info *iota.NodeInfoResponse

func main() {
	endpoint := flag.String("endpoint", "http://localhost:14265", "endpoint")
	nodeApi = iota.NewNodeAPIClient(*endpoint)
	var err error
	info, err = nodeApi.Info()
	Must(err)

	fmt.Println("Listening..")
	client := SetUpMqTT("http://localhost", 1883, nil, nil, nil)
	sub(client)
	select {}
}

func sub(client mqtt.Client) (mqtt.Token, mqtt.Token) {
	messageToken := client.Subscribe("messages", 1, censor)
	milestoneToken := client.Subscribe("milestones/solid", 1, rcvMilestone)

	messageToken.Wait()
	milestoneToken.Wait()

	//currently milestone is not important
	return messageToken, milestoneToken
}

func rcvMilestone(client mqtt.Client, msg mqtt.Message) {
	milestone := iota.Milestone{}
	err := milestone.UnmarshalJSON(msg.Payload())
	Must(err)
	if len(msQueue) >= 15 {
		msQueue = msQueue[1:]
	}
	msQueue = append(msQueue, milestone.Index)
	fmt.Println("add ms index ", milestone.Index)
}

func censor(client mqtt.Client, msg mqtt.Message) {
	fmt.Println("start censoring")
	msgId := getMessageId(msg)
	semLazyMs := getSemLazyMs()
	vergeOfBecomingSemLazyMsg := send_sem_lazy(msgId, semLazyMs, -1)
	for i := 0; i < semLazyExplosionSize; i++ {
		id, err := vergeOfBecomingSemLazyMsg.ID()
		Must(err)
		stitch(*id, getLatestMsId(), i)
	}
}

func getLatestMsId() iota.MessageID {
	info, err := nodeApi.Info()
	Must(err)
	ms, err := nodeApi.MilestoneByIndex(info.LatestMilestoneIndex)
	Must(err)
	msgId, err := iota.MessageIDFromHexString(ms.MessageID)
	Must(err)
	return msgId
}

func send_sem_lazy(id iota.MessageID, ms iota.MessageID, num int) *iota.Message {
	fmt.Println("sending a tx on the verge of being semi-lazy")
	parents := iota.RemoveDupsAndSortByLexicalOrderArrayOf32Bytes(iota.MessageIDs{id, ms})
	return SendDataMessage(nodeApi, &info.NetworkID, &parents, "semlazy", string(num))
}

func stitch(id iota.MessageID, ms iota.MessageID, stichnum int) *iota.Message {
	fmt.Println("sending a stitch")
	parents := iota.RemoveDupsAndSortByLexicalOrderArrayOf32Bytes(iota.MessageIDs{id, ms})
	return SendDataMessage(nodeApi, &info.NetworkID, &parents, "censor stitch", string(stichnum))
}

func getSemLazyMs() iota.MessageID {
	info, err := nodeApi.Info()
	Must(err)
	solidMilestoneIndex := info.ConfirmedMilestoneIndex
	msResp, err := nodeApi.MilestoneByIndex(solidMilestoneIndex - 13)
	Must(err)
	msgId, err := iota.MessageIDFromHexString(msResp.MessageID)
	Must(err)
	return msgId
}

func getMessageId(msg mqtt.Message) iota.MessageID {
	return blake2b.Sum256(msg.Payload())
}
