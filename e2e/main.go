package main

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/iotaledger/iota.go/address"
	"github.com/iotaledger/iota.go/api"
	"github.com/iotaledger/iota.go/bundle"
	"github.com/iotaledger/iota.go/consts"
	"github.com/iotaledger/iota.go/kerl"
	"github.com/iotaledger/iota.go/pow"
	"github.com/iotaledger/iota.go/transaction"
	"github.com/iotaledger/iota.go/trinary"
)

var (
	mode                   = flag.String("mode", "spam", "the mode of operation for the program")
	nodeAPIURI             = flag.String("node", "https://api.coo.manapotion.io", "the API URI of the node")
	migrateBatchSize       = flag.Int("migration-batch-size", 40, "the size of the migration batch")
	migrateFrom            = flag.Int("migration-from", 0, "starting index of the migrations")
	migrateTo              = flag.Int("migration-to", 1000, "end index of the migrations")
	migrationSourceFile    = flag.String("migration-source-file", "seedmap.csv", "the seed map file containing the data for the migrations")
	migrationInfoFile      = flag.String("migration-info-file", "migrated.csv", "the name of the file which holds info about which migration bundles were generated")
	seedMapToCSVSourceFile = flag.String("to-csv-source-file", "seedmap.txt", "the source file to convert")
	seedMapToCSVTargetFile = flag.String("to-csv-target-file", "seedmap.csv", "the target file to produce")
	gsAddrsToGenerate      = flag.Int("gs-addrs-count", 100000, "the amount of genesis addresses to generate")
	gsSeedMapFileName      = flag.String("gs-seed-map-file", "seedmap.txt", "the file to which to write the seed map to")
	gsSnapshotFileName     = flag.String("gs-snapshot-file-file", "snapshot.csv", "the file to which to write the global snapshot data to")
)

type Mode string

const (
	ModeSpam                            Mode = "spam"
	ModeMigrateSeedMap                  Mode = "migrate"
	ModeConvertSeedMapToCSV             Mode = "convertSeedMapToCSV"
	ModeGenerateGlobalSnapshotAddresses Mode = "generateGSAddresses"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	flag.Parse()

	_, powF := pow.GetFastestProofOfWorkImpl()
	legacyAPI, err := api.ComposeAPI(api.HTTPClientSettings{
		URI:                  *nodeAPIURI,
		LocalProofOfWorkFunc: powF,
	})
	must(err)

	fmt.Println("running program with mode:", *mode)

	switch Mode(*mode) {
	case ModeSpam:
		spamLegacy(legacyAPI)
	case ModeMigrateSeedMap:
		migrateSeedMap(legacyAPI, *migrationSourceFile, *migrationInfoFile, *migrateBatchSize, *migrateFrom, *migrateTo)
	case ModeConvertSeedMapToCSV:
		seedMapToCsv(*seedMapToCSVSourceFile, *seedMapToCSVTargetFile)
	case ModeGenerateGlobalSnapshotAddresses:
		generateGSAddresses(*gsAddrsToGenerate, *gsSeedMapFileName, *gsSnapshotFileName)
	default:
		fmt.Println("invalid program mode, supported are:", ModeMigrateSeedMap, ModeSpam, ModeConvertSeedMapToCSV, ModeGenerateGlobalSnapshotAddresses)
	}
}

var emptyTrytes = strings.Repeat("9", 81)
var emptyTrytesWithChecksum string

func init() {
	checksum, err := address.Checksum(emptyTrytes)
	if err != nil {
		panic(err)
	}
	emptyTrytesWithChecksum = emptyTrytes + checksum
}

func spamLegacy(legacyAPI *api.API) {
	for i := 0; true; i++ {
		spamTx, err := createSpamBundle(legacyAPI)
		if err != nil {
			panic(err)
		}
		if _, err := legacyAPI.BroadcastTransactions(spamTx...); err != nil {
			panic(err)
		}
		fmt.Printf("%d\t\r", i+1)
	}
}

func createSpamBundle(legacyAPI *api.API) ([]trinary.Trytes, error) {
	prepBundle, err := legacyAPI.PrepareTransfers(emptyTrytes, bundle.Transfers{
		{
			Address: emptyTrytesWithChecksum,
			Value:   0,
			Message: "",
			Tag:     "",
		},
	}, api.PrepareTransfersOptions{})
	if err != nil {
		return nil, err
	}

	tips, err := legacyAPI.GetTransactionsToApprove(3)
	if err != nil {
		return nil, err
	}

	readyBundle, err := legacyAPI.AttachToTangle(tips.TrunkTransaction, tips.BranchTransaction, 1, prepBundle)
	if err != nil {
		return nil, err
	}

	return readyBundle, nil
}

func migrateSeedMap(legacyAPI *api.API, srcFileName string, infoFileName string, batchSize int, from int, to int) {
	seedMapFileCSV, err := os.OpenFile(srcFileName, os.O_RDONLY, os.ModePerm)
	must(err)
	defer seedMapFileCSV.Close()

	legacyMigrated, err := os.OpenFile(infoFileName, os.O_CREATE|os.O_TRUNC|os.O_RDWR, os.ModePerm)
	must(err)
	defer legacyMigrated.Close()

	exit := make(chan struct{})
	defer close(exit)

	toBroadcast := make(chan [][]trinary.Trytes, 4)
	go broadcaster(legacyAPI, toBroadcast)
	defer close(toBroadcast)

	tipsChan := make(chan *api.TransactionsToApprove, 4)
	for i := 0; i < runtime.NumCPU(); i++ {
		go requestTips(legacyAPI, exit, tipsChan)
	}

	var currentBatch [][]trinary.Trytes
	for i := 0; i < to; i++ {
		var seed, firstAddr string
		if _, err := fmt.Fscanln(seedMapFileCSV, &seed, &firstAddr); err == io.EOF {
			break
		}

		if i < from {
			continue
		}

		fmt.Printf("%d to %d (%d)\t\r", from, to, i+1)

		tailTxHash, ed25519Hex, bndl, err := migrate(legacyAPI, seed, 27795302832, tipsChan)
		must(err)

		if _, err := fmt.Fprintln(legacyMigrated, tailTxHash, ed25519Hex); err != nil {
			must(err)
		}

		currentBatch = append(currentBatch, bndl)

		if len(currentBatch) == batchSize {
			toBroadcast <- currentBatch
			currentBatch = make([][]trinary.Trytes, 0)
		}
	}
}

func requestTips(legacyAPI *api.API, exit <-chan struct{}, tips chan<- *api.TransactionsToApprove) {
	for {
		select {
		case <-exit:
			return
		default:
		}
		res, err := legacyAPI.GetTransactionsToApprove(3)
		must(err)
		select {
		case tips <- res:
		case <-exit:
			return
		}
	}
}

func broadcaster(legacyAPI *api.API, toBroadcast <-chan [][]trinary.Trytes) {
	for bundles := range toBroadcast {
		var wg sync.WaitGroup
		wg.Add(len(bundles))
		for _, bndl := range bundles {
			go func(bndl []trinary.Trytes) {
				defer wg.Done()
				if _, err := legacyAPI.BroadcastTransactions(bndl...); err != nil {
					must(err)
				}
			}(bndl)
		}
		wg.Wait()
	}
}

func seedMapToCsv(src string, target string) {
	seedMapFile, err := os.Open(src)
	must(err)
	defer seedMapFile.Close()

	seedMapFileCSV, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_RDWR, os.ModePerm)
	must(err)
	defer seedMapFileCSV.Close()

	for {
		var seed, firstAddr string
		if _, err := fmt.Fscanf(seedMapFile, "%s --> %s", &seed, &firstAddr); err == io.EOF {
			break
		}

		if _, err := fmt.Fprintln(seedMapFileCSV, seed, firstAddr); err != nil {
			must(err)
		}
	}
}

func generateGSAddresses(count int, seedMapFileName string, snapshotFileName string) {

	remainder := consts.TotalSupply % uint64(count)
	fundsPerAddr := (consts.TotalSupply - remainder) / uint64(count)

	if remainder != 0 {
		count++
	}

	var sbSnapshot, sbSeedMap strings.Builder
	seedMapfile, err := os.OpenFile("seedmap.txt", os.O_TRUNC|os.O_CREATE|os.O_RDWR, os.ModePerm)
	must(err)
	defer seedMapfile.Close()

	snapshotFile, err := os.OpenFile("snapshot.csv", os.O_TRUNC|os.O_CREATE|os.O_RDWR, os.ModePerm)
	must(err)
	defer snapshotFile.Close()

	for i := 0; i < count; i++ {
		seed, addr := seedAndFirstAddr()
		_, err := seedMapfile.WriteString(fmt.Sprintf("%s --> %s\n", seed, addr))
		must(err)

		dep := fundsPerAddr
		if remainder != 0 && i+1 == count {
			dep = remainder
		}

		_, err = snapshotFile.WriteString(fmt.Sprintf("%s;%d\n", addr, dep))
		must(err)
	}

	fmt.Print(sbSeedMap.String())
	fmt.Print(sbSnapshot.String())
}

func seedAndFirstAddr() (string, string) {
	seed := randSeed()
	addr, err := address.GenerateAddress(seed, 0, consts.SecurityLevelMedium, false)
	must(err)
	return seed, addr
}

func randSeed() string {
	b := make([]byte, consts.HashBytesSize)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}

	// convert to trytes and set the last trit to zero
	seed, err := kerl.KerlBytesToTrytes(b)
	if err != nil {
		panic(err)
	}

	return seed
}

func migrate(legacyAPI *api.API, seed string, val uint64, tipsChan <-chan *api.TransactionsToApprove) (string, string, []trinary.Trytes, error) {
	legacyAddr, err := address.GenerateAddress(seed, 0, consts.SecurityLevelMedium, true)
	if err != nil {
		return "", "", nil, err
	}

	var ed25519Addr [32]byte
	if _, err = rand.Read(ed25519Addr[:]); err != nil {
		return "", "", nil, err
	}

	migAddr, err := address.GenerateMigrationAddress(ed25519Addr, true)
	if err != nil {
		return "", "", nil, err
	}

	prepBundle, err := legacyAPI.PrepareTransfers(seed, bundle.Transfers{
		{Address: migAddr, Value: val},
	}, api.PrepareTransfersOptions{
		Inputs: []api.Input{
			{
				Balance:  val,
				Address:  legacyAddr,
				KeyIndex: 0,
				Security: consts.SecurityLevelMedium,
			},
		},
	})
	if err != nil {
		return "", "", nil, err
	}

	tips := <-tipsChan

	readyBundle, err := legacyAPI.AttachToTangle(tips.TrunkTransaction, tips.BranchTransaction, 1, prepBundle)
	if err != nil {
		return "", "", nil, err
	}

	tailTx, err := transaction.AsTransactionObject(readyBundle[0])
	if err != nil {
		return "", "", nil, err
	}

	return tailTx.Hash, hex.EncodeToString(ed25519Addr[:]), readyBundle, nil
}
