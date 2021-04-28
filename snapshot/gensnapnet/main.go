// This tool takes a genesis snapshot and then alters the outputs in it to incorporate the network ID.
// Mainly used to be able to alter a genesis snapshot so that its bootstrapped network is not subject to replay attacks.
package main

import (
	"encoding/binary"
	"flag"
	"log"
	"os"
	"time"

	"github.com/gohornet/hornet/pkg/model/hornet"
	"github.com/gohornet/hornet/pkg/model/utxo"
	"github.com/gohornet/hornet/pkg/snapshot"
	"github.com/iotaledger/iota.go/v2"
)

var (
	snapshotFileName = flag.String("source-file", "genesis_snapshot_alt.bin", "the name of the genesis snapshot file to alter")
	targetFileName   = flag.String("target-file", "mod_genesis_snapshot_alt.bin", "the name of the modified genesis snapshot")
	networkID        = flag.String("network-id", "as-network", "the name of the network to incorporate into the outputs")
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	flag.Parse()
	s := time.Now()

	log.Printf("converting %s to %s by applying '%s' to the genesis outputs placeholder tx IDs", *snapshotFileName, *targetFileName, *networkID)

	sourceFile, err := os.OpenFile(*snapshotFileName, os.O_RDONLY, 0666)
	must(err)
	defer func() {
		if err := sourceFile.Close(); err != nil {
			log.Panicln("could not close source file:", err)
		}
	}()

	targetFile, err := os.OpenFile(*targetFileName, os.O_RDWR|os.O_CREATE, 0666)
	must(err)
	defer func() {
		if err := targetFile.Close(); err != nil {
			log.Panicln("could not close target file:", err)
		}
	}()

	netIDNum := iotago.NetworkIDFromString(*networkID)

	// dont even bother directly streaming into to the target file
	var readHeader *snapshot.ReadFileHeader
	var seps []hornet.MessageID
	var outputs []*snapshot.Output
	var treasuryOutput *utxo.TreasuryOutput
	var msDiffs []*snapshot.MilestoneDiff
	must(snapshot.StreamSnapshotDataFrom(sourceFile, func(header *snapshot.ReadFileHeader) error {
		readHeader = header
		return nil
	}, func(id hornet.MessageID) error {
		seps = append(seps, id)
		return nil
	}, func(output *snapshot.Output) error {
		outputs = append(outputs, output)
		return nil
	}, func(output *utxo.TreasuryOutput) error {
		treasuryOutput = output
		return nil
	}, func(milestoneDiff *snapshot.MilestoneDiff) error {
		msDiffs = append(msDiffs, milestoneDiff)
		return nil
	}))

	var sepsIndex, outputsIndex int
	err, _ = snapshot.StreamSnapshotDataTo(targetFile, readHeader.Timestamp, &readHeader.FileHeader, func() (hornet.MessageID, error) {
		if sepsIndex == len(seps) {
			return nil, nil
		}
		sep := seps[sepsIndex]
		sepsIndex++
		return sep, nil
	}, func() (*snapshot.Output, error) {
		if outputsIndex == len(outputs) {
			return nil, nil
		}
		output := outputs[outputsIndex]

		// alter the output ID to incorporate the network ID
		binary.LittleEndian.PutUint64(output.OutputID[:iotago.UInt64ByteSize], netIDNum)

		outputsIndex++
		return output, nil
	}, func() (*snapshot.MilestoneDiff, error) {
		return nil, nil
	})
	must(err)

	log.Printf("converted, took %v", time.Since(s))
}
