package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"

	"github.com/iotaledger/iota.go/address"
	"github.com/iotaledger/iota.go/api"
	"github.com/iotaledger/iota.go/bundle"
	"github.com/iotaledger/iota.go/consts"
	"github.com/iotaledger/iota.go/pow"
	"github.com/iotaledger/iota.go/transaction"
	"github.com/iotaledger/iota.go/trinary"
	iotago "github.com/iotaledger/iota.go/v2"
	"github.com/iotaledger/iota.go/v2/ed25519"
)

var (
	nodeAPIURI       = flag.String("node", "https://api.coo.manapotion.io", "the API URI of the node")
	batchSize        = flag.Int("batch-size", 40, "the size of the migration batch")
	fromIndex        = flag.Int("from", 0, "starting index of the migrations")
	toIndex          = flag.Int("to", 1000, "end index of the migrations")
	seedMapFile      = flag.String("source-file", "seedmap.csv", "the seed map file containing the data for the migrations")
	migratedInfoFile = flag.String("info-file", "migrated.csv", "the name of the file which holds info about which migration bundles were generated")
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

	migrateSeedMap(legacyAPI, *seedMapFile, *migratedInfoFile, *batchSize, *fromIndex, *toIndex)
}

func migrateSeedMap(legacyAPI *api.API, srcFileName string, infoFileName string, batchSize int, from int, to int) {
	seedMapFileCSV, err := os.OpenFile(srcFileName, os.O_RDONLY, os.ModePerm)
	must(err)
	defer seedMapFileCSV.Close()

	migrationInfoFile, err := os.OpenFile(infoFileName, os.O_CREATE|os.O_APPEND|os.O_RDWR, os.ModePerm)
	must(err)
	defer migrationInfoFile.Close()

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
		var funds uint64
		if _, err := fmt.Fscanln(seedMapFileCSV, &seed, &firstAddr, &funds); err == io.EOF {
			break
		}

		if i < from {
			continue
		}

		fmt.Printf("%d to %d (%d)\t\r", from, to, i+1)

		tailTxHash, ed25519PrvKey, addrHex, bndl, err := migrate(legacyAPI, seed, funds, tipsChan)
		must(err)

		if _, err := fmt.Fprintln(migrationInfoFile, tailTxHash, hex.EncodeToString(ed25519PrvKey), addrHex, funds); err != nil {
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

func migrate(legacyAPI *api.API, seed string, val uint64, tipsChan <-chan *api.TransactionsToApprove) (string, ed25519.PrivateKey, string, []trinary.Trytes, error) {
	legacyAddr, err := address.GenerateAddress(seed, 0, consts.SecurityLevelMedium, true)
	if err != nil {
		return "", nil, "", nil, err
	}

	pubKey, prvKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return "", nil, "", nil, err
	}

	ed25519Addr := iotago.AddressFromEd25519PubKey(pubKey)
	migAddr, err := address.GenerateMigrationAddress(ed25519Addr, true)
	if err != nil {
		return "", nil, "", nil, err
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
		return "", nil, "", nil, err
	}

	tips := <-tipsChan

	readyBundle, err := legacyAPI.AttachToTangle(tips.TrunkTransaction, tips.BranchTransaction, 1, prepBundle)
	if err != nil {
		return "", nil, "", nil, err
	}

	tailTx, err := transaction.AsTransactionObject(readyBundle[0])
	if err != nil {
		return "", nil, "", nil, err
	}

	return tailTx.Hash, prvKey, hex.EncodeToString(ed25519Addr[:]), readyBundle, nil
}
