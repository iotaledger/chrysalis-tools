package lib

import (
	"encoding/binary"
	"flag"
	"fmt"
	. "github.com/GalRogozinski/iota.go/v2"
	ed "github.com/GalRogozinski/iota.go/v2/ed25519"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	ApiPort   = 14265
	MqttPort  = 1883
	LocalHost = "127.0.0.1"
)

/**
private key, public key, and address
*/
type KeyTriplet struct {
	Sk      ed.PrivateKey
	Pk      ed.PublicKey
	Address Ed25519Address
}

var (
	GenesisOutput = [TransactionIDLength]byte{}
)

func Must(err error) {
	if err != nil {
		panic(err)
	}
}

func DefineNodeFlags() (*string, *int) {
	nodeDomain := flag.String("node", LocalHost, "Can be either domain name or ip of the node")
	blowballSize := flag.Int("port", ApiPort, "Api port")

	return nodeDomain, blowballSize
}

func ObtainAPI(nodeUrl string, apiPort int) (*NodeHTTPAPIClient, *NodeInfoResponse) {
	endpoint := fmt.Sprintf("http://%s:%d", nodeUrl, apiPort)
	nodeAPI := NewNodeHTTPAPIClient(endpoint)
	info, err := nodeAPI.Info()
	Must(err)
	return nodeAPI, info
}

func SetUpMqTT(broker string, port uint, messagePubHandler func(client mqtt.Client, msg mqtt.Message),
	connectHandler func(client mqtt.Client), connectLostHandler func(client mqtt.Client, err error)) mqtt.Client {

	if messagePubHandler == nil {
		messagePubHandler = func(client mqtt.Client, msg mqtt.Message) {
			fmt.Printf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
		}
	}

	if connectHandler == nil {
		connectHandler = func(client mqtt.Client) {
			fmt.Println("Connected")
		}
	}

	if connectLostHandler == nil {
		connectLostHandler = func(client mqtt.Client, err error) {
			fmt.Printf("Connect lost: %v", err)
		}
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", broker, port))
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	return client
}

func SendDataMessage(api *NodeHTTPAPIClient, networkId *string, parents *MessageIDs, index string, data string) *Message {
	m := CreateDataMessage(api, networkId, parents, index, data)

	message, err := api.SubmitMessage(m)
	Must(err)
	return message
}

func CreateDataMessage(api *NodeHTTPAPIClient, networkId *string, parents *MessageIDs, index string, data string) *Message {
	m := &Message{}
	m.NetworkID = NetworkIDFromString(*networkId)
	m.Payload = &Indexation{Index: []byte(index), Data: []byte(data)}
	m.Parents = *getParentsIfNil(api, parents)
	return m
}

func SendValueMessage(api *NodeHTTPAPIClient, networkId *string, parents *MessageIDs, tx *Transaction) *Message {
	m := CreateValueMessage(api, networkId, parents, tx)
	message, err := api.SubmitMessage(m)
	Must(err)
	return message
}

func CreateValueMessage(api *NodeHTTPAPIClient, networkId *string, parents *MessageIDs, tx *Transaction) *Message {
	m := &Message{}
	m.NetworkID = NetworkIDFromString(*networkId)
	m.Payload = tx
	m.Parents = *getParentsIfNil(api, parents)
	return m
}

func getParentsIfNil(api *NodeHTTPAPIClient, parents *MessageIDs) *MessageIDs {
	if parents == nil {
		tipsResponse, err := api.Tips()
		Must(err)
		tips, err := tipsResponse.Tips()
		Must(err)
		parents = &MessageIDs{}
		for _, tip := range tips {
			Must(err)
			var parent MessageID
			copy(parent[:], tip[:])
			*parents = append(*parents, parent)
		}

	}
	return parents
}

func GenerateKeys(seed []byte) (ed.PrivateKey, ed.PublicKey) {
	privateKey := ed.NewKeyFromSeed(seed)
	publicKey := make([]byte, ed.PublicKeySize)
	copy(publicKey, privateKey[32:])
	return privateKey, publicKey
}

func CreateInput(inputAddress Address, txId [32]byte, outIndex uint16) *ToBeSignedUTXOInput {
	input := UTXOInput{
		TransactionID:          txId,
		TransactionOutputIndex: outIndex,
	}

	return &ToBeSignedUTXOInput{
		Address: inputAddress,
		Input:   &input,
	}
}

func CreateOutput(outputAddress Address, amount uint64) *SigLockedSingleOutput {
	return &SigLockedSingleOutput{
		Address: outputAddress,
		Amount:  amount,
	}
}

func CreateSeed(pattern []byte) []byte {
	seed := make([]byte, ed.SeedSize)

	// Copy the pattern into the start of the container
	copy(seed, pattern)

	// Incrementally duplicate the pattern throughout the container
	for j := len(pattern); j < len(seed); j *= 2 {
		copy(seed[j:], seed[:j])
	}

	return seed
}

func GenerateAddressFromSeed(seed []byte) (ed.PrivateKey, ed.PublicKey, Ed25519Address) {
	privateKey, publicKey := GenerateKeys(seed)
	address := AddressFromEd25519PubKey(publicKey)
	return privateKey, publicKey, address
}

/**
Returns lists of keys and addresses
*/
func GenerateKeyChain(chainLength int, buf_offset int) (keyTriplets []KeyTriplet) {
	var chain []KeyTriplet

	for i := 1; i <= chainLength; i++ {
		buf := make([]byte, 4)
		binary.PutUvarint(buf, uint64(i+buf_offset))
		seed := CreateSeed(buf)
		sk, pk, address := GenerateAddressFromSeed(seed)
		chain = append(chain, KeyTriplet{sk, pk, address})
	}
	return chain
}
