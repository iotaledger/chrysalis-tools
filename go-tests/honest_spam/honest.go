package main

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	iota "github.com/GalRogozinski/iota.go/v2"
	"github.com/eclipse/paho.mqtt.golang"
	. "github.com/iotaledger/chrysalis-tools/go-tests/lib"
	"log"
)

var (
	totalMessageCounter   uint = 0
	totalMessageConfirmed uint = 0
	unconfirmedMessages        = make(map[string]struct{})
)

func main() {
	nodeDomain, apiPort := DefineNodeFlags()
	mqttPort := flag.Int("mqtt", MqttPort, "Mqtt port")
	flag.Parse()

	nodeAPI, nodeInfo := ObtainAPI(*nodeDomain, *apiPort)

	client := SetUpMqTT(*nodeDomain, uint(*mqttPort), nil, nil, nil)

	var i uint32 = 0
	for {
		i++
		msg := CreateDataMessage(nodeAPI, &nodeInfo.NetworkID, nil, "honest", string(i))
		message, err := nodeAPI.SubmitMessage(msg)
		Must(err)
		totalMessageCounter++
		msgId, err := message.ID()
		Must(err)
		id := hex.EncodeToString(msgId[:])
		subToConfirm(client, &id)
		unconfirmedMessages[id] = struct{}{}
	}

}

func subToConfirm(client mqtt.Client, msgId *string) {
	client.Subscribe(fmt.Sprintf("messages/%s/metadata", *msgId), 1,
		func(_ mqtt.Client, msg mqtt.Message) {
			metadata := iota.MessageMetadataResponse{}
			err := json.Unmarshal(msg.Payload(), &metadata)
			Must(err)
			if _, ok := unconfirmedMessages[metadata.MessageID]; ok && metadata.ReferencedByMilestoneIndex != nil {
				{
					totalMessageConfirmed++
					delete(unconfirmedMessages, *msgId)
					log.Printf("%s confirmed\n", *msgId)
					log.Printf("confirmation ratio is %.2f\n", float32(totalMessageConfirmed)/float32(totalMessageCounter))
				}
			}
		})
}
