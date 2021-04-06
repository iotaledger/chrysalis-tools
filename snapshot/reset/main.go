// This tool queries a Chrysalis Phase 2 node for UTXOs and then generates a full snapshot containing those.
package main

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gohornet/hornet/pkg/model/hornet"
	"github.com/gohornet/hornet/pkg/model/milestone"
	"github.com/gohornet/hornet/pkg/model/utxo"
	"github.com/gohornet/hornet/pkg/snapshot"
	iotago "github.com/iotaledger/iota.go/v2"
)

var (
	nodeURI         = flag.String("node", "http://localhost:14265", "the node URI of the node to query")
	utxosRoute      = flag.String("outputs-debug-route", "/api/plugins/debug/outputs/unspent", "the route to query for UTXOs")
	outputFile      = flag.String("output-file", "full_snapshot.bin", "the name of the file to output")
	networkID       = flag.String("network-id", "testnet", "the string name of the network ID")
	targetIndex     = flag.Int("target-index", 0, "the index to use for the ledger and snapshot index")
	parallelQueries = flag.Int("parallel-queries", 200, "the amount of simultaneous requests to query outputs")
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

type expectedres struct {
	Data struct {
		OutputIDs []string `json:"outputIds"`
	}
}

func main() {
	flag.Parse()

	uri := fmt.Sprintf("%s%s", *nodeURI, *utxosRoute)
	s := time.Now()
	log.Printf("querying node at %s for UTXO IDs...", uri)

	res, err := http.Get(uri)
	must(err)
	defer res.Body.Close()

	bodyContent, err := ioutil.ReadAll(res.Body)
	must(err)

	if res.StatusCode != http.StatusOK {
		panic(fmt.Sprintf("got non 200 http status from query (status code %d instead) and msg '%s'", res.StatusCode, string(bodyContent)))
	}

	query := &expectedres{}
	must(json.Unmarshal(bodyContent, query))
	lenOutputs := len(query.Data.OutputIDs)
	log.Printf("queried %d UTXO IDs, %v", lenOutputs, time.Since(s))

	log.Printf("querying UTXOs...")
	nodeHTTPClient := iotago.NewNodeHTTPAPIClient(*nodeURI)

	var utxosMu sync.Mutex
	utxos := make([]*iotago.NodeOutputResponse, lenOutputs)

	var loops int64
	batchSize := *parallelQueries
	for i := 0; i < lenOutputs+batchSize; i += batchSize {
		var wg sync.WaitGroup
		wg.Add(batchSize)
		for j := 0; j < batchSize; j++ {
			if i+j >= len(query.Data.OutputIDs) {
				wg.Done()
				continue
			}
			go func(index int, outputIdHex string) {
				defer wg.Done()
				outputRes, err := nodeHTTPClient.OutputByID(iotago.OutputIDHex(outputIdHex).MustAsUTXOInput().ID())
				must(err)
				utxosMu.Lock()
				utxos[index] = outputRes
				utxosMu.Unlock()
				atomic.AddInt64(&loops, 1)
				fmt.Printf("%d of %d UTXOs queried\t\r", loops, lenOutputs)
			}(i+j, query.Data.OutputIDs[i+j])
		}
		wg.Wait()
	}

	fmt.Println()

	log.Printf("querying treasury")
	treasuryRes, err := nodeHTTPClient.Treasury()
	must(err)
	log.Printf("treasury size: %d", treasuryRes.Amount)

	if err := os.Remove(*outputFile); !os.IsNotExist(err) {
		panic(err)
	}

	snapshotFile, err := os.OpenFile(*outputFile, os.O_RDWR|os.O_CREATE, 0666)
	must(err)

	// create snapshot file
	log.Printf("generating full snapshot file for target index %d with network ID %s", *targetIndex, *networkID)
	header := &snapshot.FileHeader{
		Version:              snapshot.SupportedFormatVersion,
		Type:                 snapshot.Full,
		NetworkID:            iotago.NetworkIDFromString(*networkID),
		SEPMilestoneIndex:    milestone.Index(*targetIndex),
		LedgerMilestoneIndex: milestone.Index(*targetIndex),
		TreasuryOutput: &utxo.TreasuryOutput{
			MilestoneID: iotago.MilestoneID{},
			Amount:      treasuryRes.Amount,
		},
	}

	nullHashAdded := false
	solidEntryPointProducerFunc := func() (hornet.MessageID, error) {
		if nullHashAdded {
			return nil, nil
		}

		nullHashAdded = true
		return hornet.GetNullMessageID(), nil
	}

	// unspent transaction outputs
	var utxoIndex int
	outputProducerFunc := func() (*snapshot.Output, error) {
		if utxoIndex == len(utxos) {
			return nil, nil
		}

		outputRes := utxos[utxoIndex]
		output, err := outputRes.Output()
		must(err)

		target, err := output.Target()
		must(err)

		deposit, err := output.Deposit()
		must(err)

		snapOutput := &snapshot.Output{
			OutputType: output.Type(),
			Address:    target,
			Amount:     deposit,
		}

		messageID, err := hex.DecodeString(outputRes.MessageID)
		must(err)
		copy(snapOutput.MessageID[:], messageID)

		txID, err := outputRes.TxID()
		must(err)

		copy(snapOutput.OutputID[:], txID[:])
		binary.LittleEndian.PutUint16(snapOutput.OutputID[iotago.TransactionIDLength:], outputRes.OutputIndex)
		utxoIndex++
		return snapOutput, nil
	}

	// milestone diffs
	milestoneDiffProducerFunc := func() (*snapshot.MilestoneDiff, error) { return nil, nil }

	err, _ = snapshot.StreamSnapshotDataTo(snapshotFile, uint64(time.Now().Unix()), header, solidEntryPointProducerFunc, outputProducerFunc, milestoneDiffProducerFunc)
	must(err)
	must(snapshotFile.Close())

	log.Printf("done, took %v", time.Since(s))
}
