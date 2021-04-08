package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	mathrand "math/rand"
	"os"
	"strings"
	"time"

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
	nodeAPIURI   = flag.String("node", "https://api.coo.manapotion.io", "the API URI of the node")
	originSeed   = flag.String("seed", strings.Repeat("9", consts.HashTrytesSize), "the seed to use to fund the created bundles")
	infoFileName = flag.String("info-file", "bundles.csv", "the file containing the different generated bundles")
	mwm          = flag.Int("mwm", 14, "the mwm to use for generated transactions/bundles")
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

	firstAddr, err := address.GenerateAddress(*originSeed, 0, consts.SecurityLevelMedium, true)
	must(err)

	balancesRes, err := legacyAPI.GetBalances(trinary.Hashes{firstAddr})
	must(err)

	log.Printf("there are %d tokens residing on the first address of the specified seed", balancesRes.Balances[0])

	generateBundles(legacyAPI, firstAddr)
}

func generateBundles(legacyAPI *api.API, originFirstAddr trinary.Hash) {
	s := time.Now()
	if err := os.Remove(*infoFileName); err != nil && !os.IsNotExist(err) {
		panic(err)
	}

	infoFile, err := os.OpenFile(*infoFileName, os.O_RDWR|os.O_CREATE, 0666)
	must(err)
	defer infoFile.Close()

	log.Println("generating account with one address")
	generateOneAddressAccount(legacyAPI, originFirstAddr, infoFile)

	insertSeparator(infoFile)

	log.Println("generating account with sparse addresses")
	generateSparseIndexesAddressesAccount(legacyAPI, originFirstAddr, infoFile)

	insertSeparator(infoFile)

	log.Println("generating account with minimum migration amount spread across many addresses")
	minMigrationAmountSpreadAccount(legacyAPI, originFirstAddr, infoFile)

	insertSeparator(infoFile)

	log.Println("generating account with random amounts spread across many addresses")
	fundsSpreadAccount(legacyAPI, originFirstAddr, infoFile)

	insertSeparator(infoFile)

	log.Println("generating account with one spent address")
	generateOneSpentAddressAccount(legacyAPI, originFirstAddr, infoFile)

	insertSeparator(infoFile)

	log.Println("generating account with minimum migration amount spread across many spent addresses")
	minMigrationAmountSpreadSpentAccount(legacyAPI, originFirstAddr, infoFile)

	insertSeparator(infoFile)

	log.Println("generating account with random amounts spread across many spent addresses")
	fundsSpreadSpentAccount(legacyAPI, originFirstAddr, infoFile)

	log.Printf("done, goodbye! %v\n", time.Since(s))
}

func insertSeparator(infoFile *os.File) {
	_, err := fmt.Fprintf(infoFile, "\n#####################################\n")
	must(err)
}

func waitUntilConfirmed(legacyAPI *api.API, tailTx *transaction.Transaction) {
	for {
		inclState, err := legacyAPI.GetInclusionStates(trinary.Hashes{tailTx.Hash})
		must(err)

		if inclState[0] {
			break
		}
		time.Sleep(1 * time.Second)
	}
}

func sendPrepBundle(legacyAPI *api.API, infoFile io.Writer, prepBundle []trinary.Trytes) *transaction.Transaction {
	tipsRes, err := legacyAPI.GetTransactionsToApprove(3)
	must(err)

	rdyBundle, err := legacyAPI.AttachToTangle(tipsRes.TrunkTransaction, tipsRes.BranchTransaction, uint64(*mwm), prepBundle)
	must(err)

	tailTx, err := transaction.AsTransactionObject(rdyBundle[0])
	must(err)

	_, err = fmt.Fprintf(infoFile, "tail tx: %s\nbundle hash: %s\n", tailTx.Hash, tailTx.Bundle)
	must(err)

	_, err = legacyAPI.BroadcastTransactions(rdyBundle...)
	must(err)

	return tailTx
}

func sendForthAndBack(legacyAPI *api.API, infoFile io.Writer, seed trinary.Trytes, originBundle []trinary.Trytes) {
	burnerSeed := randSeed()

	txs, err := transaction.AsTransactionObjects(originBundle, nil)
	must(err)

	var transfers, backTransfers = make(bundle.Transfers, 0), make(bundle.Transfers, 0)
	var inputs, backInputs = make([]api.Input, 0), make([]api.Input, 0)

	for i, j := 0, len(txs)-1; i < j; i, j = i+1, j-1 {
		txs[i], txs[j] = txs[j], txs[i]
	}

	for i, tx := range txs {
		if tx.Value <= 0 {
			continue
		}

		originAddrChecksum, err := address.Checksum(tx.Address)
		must(err)

		addr, err := address.GenerateAddress(burnerSeed, uint64(i), consts.SecurityLevelMedium, true)
		must(err)

		transfers = append(transfers, bundle.Transfer{Address: addr, Value: uint64(tx.Value)})
		backTransfers = append(backTransfers, bundle.Transfer{
			Address: tx.Address + originAddrChecksum,
			Value:   uint64(tx.Value),
		})

		inputs = append(inputs, api.Input{
			Balance:  uint64(tx.Value),
			Address:  tx.Address + originAddrChecksum,
			KeyIndex: uint64(i),
			Security: consts.SecurityLevelMedium,
		})
		backInputs = append(backInputs, api.Input{
			Balance:  uint64(tx.Value),
			Address:  addr,
			KeyIndex: uint64(i),
			Security: consts.SecurityLevelMedium,
		})
	}

	burnerBundle, err := legacyAPI.PrepareTransfers(seed, transfers, api.PrepareTransfersOptions{
		Inputs:   inputs,
		Security: consts.SecurityLevelMedium,
	})
	must(err)

	tailTx := sendPrepBundle(legacyAPI, ioutil.Discard, burnerBundle)
	log.Printf("waiting for burner bundle to be confirmed then sending back... (tail %s)", tailTx.Hash)
	waitUntilConfirmed(legacyAPI, tailTx)
	log.Println("burner bundle confirmed, sending back to origin")

	sendBackbundle, err := legacyAPI.PrepareTransfers(burnerSeed, backTransfers, api.PrepareTransfersOptions{
		Inputs:   backInputs,
		Security: consts.SecurityLevelMedium,
	})
	must(err)

	sendPrepBundle(legacyAPI, infoFile, sendBackbundle)
}

func generateOneAddressAccount(legacyAPI *api.API, originFirstAddr trinary.Hash, infoFile *os.File) {
	const funds = 1_500_000

	// funds on one address
	fundsOnOneAddrSeed := randSeed()

	_, err := fmt.Fprintf(infoFile, "bundle type: %s", "funds on one address\n")
	must(err)

	fundsOnOneAddr, err := address.GenerateAddress(fundsOnOneAddrSeed, 0, consts.SecurityLevelMedium, true)
	must(err)
	_, err = fmt.Fprintf(infoFile, "seed %s\nfirst addr %s\naccount funds: %d\n", fundsOnOneAddrSeed, fundsOnOneAddr, funds)
	must(err)

	prepBundle, err := legacyAPI.PrepareTransfers(*originSeed, bundle.Transfers{
		{
			Address: fundsOnOneAddr,
			Value:   funds,
		},
	}, api.PrepareTransfersOptions{
		Inputs: []api.Input{
			{
				Balance:  funds,
				Address:  originFirstAddr,
				KeyIndex: 0,
				Security: consts.SecurityLevelMedium,
			},
		},
		Security: consts.SecurityLevelMedium,
	})
	must(err)

	sendPrepBundle(legacyAPI, infoFile, prepBundle)
}

func generateSparseIndexesAddressesAccount(legacyAPI *api.API, originFirstAddr trinary.Hash, infoFile *os.File) {
	const funds = 1_500_000

	fundsOnSparseAddrsSeed := randSeed()

	_, err := fmt.Fprintf(infoFile, "bundle type: %s", "funds on multiple sparse addresses\n")
	must(err)

	_, err = fmt.Fprintf(infoFile, "seed %s\naccount funds: %d\n", fundsOnSparseAddrsSeed, funds)
	must(err)

	firstAddr, err := address.GenerateAddress(fundsOnSparseAddrsSeed, 0, consts.SecurityLevelMedium, true)
	must(err)

	_, err = fmt.Fprintf(infoFile, "addr index %d: %s, funds: %d\n", 0, firstAddr, funds/2)

	secondAddr, err := address.GenerateAddress(fundsOnSparseAddrsSeed, 100, consts.SecurityLevelMedium, true)
	must(err)

	_, err = fmt.Fprintf(infoFile, "addr index %d: %s, funds: %d\n", 100, secondAddr, funds/2)

	prepBundle, err := legacyAPI.PrepareTransfers(*originSeed, bundle.Transfers{
		{
			Address: firstAddr,
			Value:   funds / 2,
		},
		{
			Address: secondAddr,
			Value:   funds / 2,
		},
	}, api.PrepareTransfersOptions{
		Inputs: []api.Input{
			{
				Balance:  funds,
				Address:  originFirstAddr,
				KeyIndex: 0,
				Security: consts.SecurityLevelMedium,
			},
		},
		Security: consts.SecurityLevelMedium,
	})
	must(err)

	sendPrepBundle(legacyAPI, infoFile, prepBundle)
}

func minMigrationAmountSpreadAccount(legacyAPI *api.API, originFirstAddr trinary.Hash, infoFile *os.File) {
	const funds = 1_000_000
	const spread = 100

	minMigAmountSpreadSeed := randSeed()

	_, err := fmt.Fprintf(infoFile, "bundle type: %s", "minimum migration amount spread across lots of addresses\n")
	must(err)

	_, err = fmt.Fprintf(infoFile, "seed %s\naccount funds: %d\n", minMigAmountSpreadSeed, funds)
	must(err)

	addrs := make(trinary.Hashes, spread)
	for i := 0; i < len(addrs); i++ {
		addr, err := address.GenerateAddress(minMigAmountSpreadSeed, uint64(i), consts.SecurityLevelMedium, true)
		must(err)
		addrs[i] = addr
		_, err = fmt.Fprintf(infoFile, "addr index %d: %s\n", i, addr)
		must(err)
	}

	transfers := make(bundle.Transfers, 0)
	for _, addr := range addrs {
		transfers = append(transfers, bundle.Transfer{
			Address: addr,
			Value:   funds / spread,
		})
	}

	prepBundle, err := legacyAPI.PrepareTransfers(*originSeed, transfers, api.PrepareTransfersOptions{
		Inputs: []api.Input{
			{
				Balance:  funds,
				Address:  originFirstAddr,
				KeyIndex: 0,
				Security: consts.SecurityLevelMedium,
			},
		},
		Security: consts.SecurityLevelMedium,
	})
	must(err)

	sendPrepBundle(legacyAPI, infoFile, prepBundle)
}

func fundsSpreadAccount(legacyAPI *api.API, originFirstAddr trinary.Hash, infoFile *os.File) {
	const funds = 10_000_000_000
	const spread = 100

	fundsSpreadSeed := randSeed()

	_, err := fmt.Fprintf(infoFile, "bundle type: %s", "funds spread across lots of addresses\n")
	must(err)

	_, err = fmt.Fprintf(infoFile, "seed %s\naccount funds: %d\n", fundsSpreadSeed, funds)
	must(err)

	addrs := make(trinary.Hashes, spread)
	for i := 0; i < len(addrs); i++ {
		addr, err := address.GenerateAddress(fundsSpreadSeed, uint64(i), consts.SecurityLevelMedium, true)
		must(err)
		addrs[i] = addr
	}

	transfers := make(bundle.Transfers, 0)
	var availFundsForDistri int64 = funds
	var fundsPerAddrRand int64 = funds / spread
	for i, addr := range addrs {
		var value int64
		if i == len(addrs)-1 {
			value = availFundsForDistri
		} else {
			value = mathrand.Int63n(fundsPerAddrRand)
		}
		availFundsForDistri -= value
		_, err = fmt.Fprintf(infoFile, "addr index %d: %s, funds: %d\n", i, addr, value)
		must(err)
		transfers = append(transfers, bundle.Transfer{Address: addr, Value: uint64(value)})
	}

	prepBundle, err := legacyAPI.PrepareTransfers(*originSeed, transfers, api.PrepareTransfersOptions{
		Inputs: []api.Input{
			{
				Balance:  funds,
				Address:  originFirstAddr,
				KeyIndex: 0,
				Security: consts.SecurityLevelMedium,
			},
		},
		Security: consts.SecurityLevelMedium,
	})
	must(err)

	sendPrepBundle(legacyAPI, infoFile, prepBundle)
}

func generateOneSpentAddressAccount(legacyAPI *api.API, originFirstAddr trinary.Hash, infoFile *os.File) {
	const funds = 1_500_000
	fundsOnOneSpentAddrSeed := randSeed()

	_, err := fmt.Fprintf(infoFile, "bundle type: %s", "funds on one spent address\n")
	must(err)

	fundsOnOneSpentAddr, err := address.GenerateAddress(fundsOnOneSpentAddrSeed, 0, consts.SecurityLevelMedium, true)
	must(err)
	_, err = fmt.Fprintf(infoFile, "seed %s\nfirst addr %s\naccount funds: %d\n", fundsOnOneSpentAddrSeed, fundsOnOneSpentAddr, funds)
	must(err)

	prepBundle, err := legacyAPI.PrepareTransfers(*originSeed, bundle.Transfers{
		{
			Address: fundsOnOneSpentAddr,
			Value:   funds,
		},
	}, api.PrepareTransfersOptions{
		Inputs: []api.Input{
			{
				Balance:  funds,
				Address:  originFirstAddr,
				KeyIndex: 0,
				Security: consts.SecurityLevelMedium,
			},
		},
		Security: consts.SecurityLevelMedium,
	})
	must(err)

	tailTx := sendPrepBundle(legacyAPI, infoFile, prepBundle)

	waitUntilConfirmed(legacyAPI, tailTx)
	log.Println("bundle confirmed, doing key-reuse forth-and-back")
	sendForthAndBack(legacyAPI, infoFile, fundsOnOneSpentAddrSeed, prepBundle)
}

func minMigrationAmountSpreadSpentAccount(legacyAPI *api.API, originFirstAddr trinary.Hash, infoFile *os.File) {
	const funds = 1_000_000
	const spread = 100

	minMigAmountSpreadSpentSeed := randSeed()

	_, err := fmt.Fprintf(infoFile, "bundle type: %s", "minimum migration amount spread across lots of spent addresses\n")
	must(err)

	_, err = fmt.Fprintf(infoFile, "seed %s\naccount funds: %d\n", minMigAmountSpreadSpentSeed, funds)
	must(err)

	addrs := make(trinary.Hashes, spread)
	for i := 0; i < len(addrs); i++ {
		addr, err := address.GenerateAddress(minMigAmountSpreadSpentSeed, uint64(i), consts.SecurityLevelMedium, true)
		must(err)
		addrs[i] = addr
		_, err = fmt.Fprintf(infoFile, "addr index %d: %s\n", i, addr)
		must(err)
	}

	transfers := make(bundle.Transfers, 0)
	for _, addr := range addrs {
		transfers = append(transfers, bundle.Transfer{
			Address: addr,
			Value:   funds / spread,
		})
	}

	prepBundle, err := legacyAPI.PrepareTransfers(*originSeed, transfers, api.PrepareTransfersOptions{
		Inputs: []api.Input{
			{
				Balance:  funds,
				Address:  originFirstAddr,
				KeyIndex: 0,
				Security: consts.SecurityLevelMedium,
			},
		},
		Security: consts.SecurityLevelMedium,
	})
	must(err)

	tailTx := sendPrepBundle(legacyAPI, infoFile, prepBundle)
	waitUntilConfirmed(legacyAPI, tailTx)
	log.Println("bundle confirmed, doing key-reuse forth-and-back")
	sendForthAndBack(legacyAPI, infoFile, minMigAmountSpreadSpentSeed, prepBundle)
}

func fundsSpreadSpentAccount(legacyAPI *api.API, originFirstAddr trinary.Hash, infoFile *os.File) {
	const funds = 10_000_000_000
	const spread = 100

	fundsSpreadSpentSeed := randSeed()

	_, err := fmt.Fprintf(infoFile, "bundle type: %s", "funds spread across lots of spent addresses\n")
	must(err)

	_, err = fmt.Fprintf(infoFile, "seed %s\naccount funds: %d\n", fundsSpreadSpentSeed, funds)
	must(err)

	addrs := make(trinary.Hashes, spread)
	for i := 0; i < len(addrs); i++ {
		addr, err := address.GenerateAddress(fundsSpreadSpentSeed, uint64(i), consts.SecurityLevelMedium, true)
		must(err)
		addrs[i] = addr
	}

	transfers := make(bundle.Transfers, 0)
	var availFundsForDistri int64 = funds
	var fundsPerAddrRand int64 = funds / spread
	for i, addr := range addrs {
		var value int64
		if i == len(addrs)-1 {
			value = availFundsForDistri
		} else {
			value = mathrand.Int63n(fundsPerAddrRand)
		}
		availFundsForDistri -= value
		_, err = fmt.Fprintf(infoFile, "addr index %d: %s, funds: %d\n", i, addr, value)
		must(err)
		transfers = append(transfers, bundle.Transfer{Address: addr, Value: uint64(value)})
	}

	prepBundle, err := legacyAPI.PrepareTransfers(*originSeed, transfers, api.PrepareTransfersOptions{
		Inputs: []api.Input{
			{
				Balance:  funds,
				Address:  originFirstAddr,
				KeyIndex: 0,
				Security: consts.SecurityLevelMedium,
			},
		},
		Security: consts.SecurityLevelMedium,
	})
	must(err)

	tailTx := sendPrepBundle(legacyAPI, infoFile, prepBundle)
	waitUntilConfirmed(legacyAPI, tailTx)
	log.Println("bundle confirmed, doing key-reuse forth-and-back")
	sendForthAndBack(legacyAPI, infoFile, fundsSpreadSpentSeed, prepBundle)
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
