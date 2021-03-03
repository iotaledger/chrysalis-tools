// Snapshot is a tool which verifies a legacy node ledger, generates information about it
// and then creates a global and genesis snapshot.
package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/gohornet/hornet/pkg/model/hornet"
	"github.com/gohornet/hornet/pkg/model/utxo"
	"github.com/gohornet/hornet/pkg/snapshot"
	"github.com/iotaledger/chrysalis-tools/common"
	"github.com/iotaledger/iota.go/address"
	"github.com/iotaledger/iota.go/api"
	"github.com/iotaledger/iota.go/consts"
	"github.com/iotaledger/iota.go/trinary"
	"github.com/iotaledger/iota.go/v2"
	"github.com/spf13/pflag"
	"golang.org/x/crypto/blake2b"
)

var (
	legacyNodeURI                = pflag.String("node", "http://localhost:14265", "the node URI of the legacy node to query")
	minMigratedFundsAmount       = pflag.Uint64("min-migration-token-amount", 1_000_000, "the minimum amount migrated funds must have")
	globalSnapshotFileName       = pflag.String("global-snapshot-file", "global_snapshot.csv", "the name of the global snapshot file to generate")
	genesisSnapshotFileName      = pflag.String("genesis-snapshot-file", "genesis_snapshot.bin", "the name of the genesis snapshot file to generate")
	genesisSnapshotFileNetworkID = pflag.String("genesis-snapshot-file-network-id", "mainnet1", "the network ID to put into the genesis snapshot")
	genesisSnapshotTimestamp     = pflag.Uint64("genesis-snapshot-file-timestamp", 0, "the timestamp to use for the genesis snapshot")
)

func must(err error) {
	if err != nil {
		log.Panic(err)
	}
}

func main() {
	pflag.Parse()

	log.Println("querying legacy node for info...")
	legacyAPI, err := api.ComposeAPI(api.HTTPClientSettings{
		URI: *legacyNodeURI,
		Client: &http.Client{
			Timeout: 5 * time.Second,
		},
	})
	must(err)

	nodeInfo, err := legacyAPI.GetNodeInfo()
	must(err)

	if nodeInfo.LatestMilestoneIndex != nodeInfo.LatestSolidSubtangleMilestoneIndex {
		log.Panicf("lsmi/lmi %d/%d don't match", nodeInfo.LatestSolidSubtangleMilestoneIndex, nodeInfo.LatestMilestoneIndex)
	}

	log.Printf("legacy node state: lsmi/lsm %d/%d", nodeInfo.LatestSolidSubtangleMilestoneIndex, nodeInfo.LatestMilestoneIndex)
	log.Printf("fetching ledger state at %d, this might take a while...go grab a coffee...", nodeInfo.LatestSolidSubtangleMilestoneIndex)

	resObj, err := common.QueryLedgerState(*legacyNodeURI, int(nodeInfo.LatestSolidSubtangleMilestoneIndex))
	must(err)

	log.Printf("total ledger entries: %d", len(resObj.Balances))
	type migration struct {
		ed25519Addr [32]byte
		value       uint64
	}
	var migrations []migration
	var totalMigration uint64
	var eligibleAddrsForMigration, eligibleAddrsTokensTotal uint64

	globalSnapshotFile, err := os.OpenFile(*globalSnapshotFileName, os.O_TRUNC|os.O_CREATE|os.O_RDWR, os.ModePerm)
	must(err)
	defer globalSnapshotFile.Close()

	type legacyLedgerEntry struct {
		addr    trinary.Hash
		balance uint64
	}

	var legacyLedgerEntries []legacyLedgerEntry
	for addr, balance := range resObj.Balances {
		legacyLedgerEntries = append(legacyLedgerEntries, legacyLedgerEntry{
			addr:    addr,
			balance: balance,
		})
	}

	sort.Slice(legacyLedgerEntries, func(i, j int) bool {
		return legacyLedgerEntries[i].addr < legacyLedgerEntries[j].addr
	})

	legacyLedgerEntriesHash, err := blake2b.New256(nil)
	must(err)
	for _, entry := range legacyLedgerEntries {
		legacyLedgerEntriesHash.Write([]byte(fmt.Sprintf("%s%d", entry.addr, entry.balance)))

		// write to global snapshot file
		_, err := fmt.Fprintf(globalSnapshotFile, "%s;%d\n", entry.addr, entry.balance)
		must(err)

		if ed25519Addr, err := address.ParseMigrationAddress(entry.addr); err == nil {
			if entry.balance < *minMigratedFundsAmount {
				continue
			}
			migrations = append(migrations, migration{
				ed25519Addr: ed25519Addr,
				value:       entry.balance,
			})
			totalMigration += entry.balance
			continue
		}
		if entry.balance >= *minMigratedFundsAmount {
			eligibleAddrsForMigration++
			eligibleAddrsTokensTotal += entry.balance
		}
	}

	log.Println("ledger state integrity hash:", hex.EncodeToString(legacyLedgerEntriesHash.Sum(nil)))
	log.Printf("migration: addrs %d, tokens total %d", len(migrations), totalMigration)
	log.Printf("eligible for migration: addrs %d, tokens total %d", eligibleAddrsForMigration, eligibleAddrsTokensTotal)

	genesisSnapshotFile, err := os.OpenFile(*genesisSnapshotFileName, os.O_TRUNC|os.O_CREATE|os.O_RDWR, os.ModePerm)
	must(err)
	defer genesisSnapshotFile.Close()

	genesisTreasuryOutput := &utxo.TreasuryOutput{
		MilestoneID: iota.MilestoneID{},
		Amount:      consts.TotalSupply - eligibleAddrsTokensTotal,
		Spent:       false,
	}

	var migrationOutputs []*snapshot.Output
	var outputIndex uint16
	fakeTransactionID := [32]byte{}
	var fakeTransactionIDBoundary uint16
	for _, migration := range migrations {
		if outputIndex == iota.MaxOutputsCount {
			outputIndex = 0
			fakeTransactionIDBoundary++
			binary.LittleEndian.PutUint16(fakeTransactionID[30:], fakeTransactionIDBoundary)
		}

		// construct fake output ID
		var outputID [34]byte
		copy(outputID[:32], fakeTransactionID[:])
		binary.LittleEndian.PutUint16(outputID[32:], outputIndex)

		output := &snapshot.Output{
			MessageID:  [32]byte{},
			OutputID:   outputID,
			OutputType: 0,
			Amount:     migration.value,
		}

		edSeri := &iota.Ed25519Address{}
		copy(edSeri[:], migration.ed25519Addr[:])
		output.Address = edSeri

		migrationOutputs = append(migrationOutputs)
		outputIndex++
	}

	var currentOutput int

	must(snapshot.StreamSnapshotDataTo(genesisSnapshotFile, *genesisSnapshotTimestamp, &snapshot.FileHeader{
		Version:              snapshot.SupportedFormatVersion,
		Type:                 0,
		NetworkID:            iota.NetworkIDFromString(*genesisSnapshotFileNetworkID),
		SEPMilestoneIndex:    0,
		LedgerMilestoneIndex: 0,
		TreasuryOutput:       genesisTreasuryOutput,
	}, func() (hornet.MessageID, error) {
		// no SEPs either
		return nil, nil
	}, func() (*snapshot.Output, error) {
		if len(migrationOutputs) == 0 || currentOutput+1 == len(migrationOutputs) {
			return nil, nil
		}
		// write out migrated funds
		output := migrationOutputs[currentOutput]
		currentOutput++
		return output, nil
	}, func() (*snapshot.MilestoneDiff, error) {
		// no milestone diffs within genesis snapshot
		return nil, nil
	}))
}
