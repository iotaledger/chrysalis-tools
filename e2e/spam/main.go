package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/iotaledger/iota.go/address"
	"github.com/iotaledger/iota.go/api"
	"github.com/iotaledger/iota.go/bundle"
	"github.com/iotaledger/iota.go/pow"
	"github.com/iotaledger/iota.go/trinary"
)

var (
	nodeAPIURI = flag.String("node", "https://api.coo.manapotion.io", "the API URI of the node")
)

var emptyTrytes = strings.Repeat("9", 81)
var emptyTrytesWithChecksum string

func init() {
	checksum, err := address.Checksum(emptyTrytes)
	if err != nil {
		panic(err)
	}
	emptyTrytesWithChecksum = emptyTrytes + checksum
}

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

	spamLegacy(legacyAPI)
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
