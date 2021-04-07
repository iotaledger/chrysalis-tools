package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"os"

	"github.com/iotaledger/iota.go/address"
	"github.com/iotaledger/iota.go/consts"
	"github.com/iotaledger/iota.go/kerl"
)

var (
	addrsToGenerateCount = flag.Int("addrs-count", 100000, "the amount of genesis addresses to generate")
	seedMapFileName      = flag.String("seed-map-file", "seedmap.csv", "the file to which to write the seed map to")
	snapshotFileName     = flag.String("snapshot-file-file", "snapshot.csv", "the file to which to write the global snapshot data to")
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	generateGlobalSnapshotAddresses(*addrsToGenerateCount, *seedMapFileName, *snapshotFileName)
}

func generateGlobalSnapshotAddresses(count int, seedMapFileName string, snapshotFileName string) {

	remainder := consts.TotalSupply % uint64(count)
	fundsPerAddr := (consts.TotalSupply - remainder) / uint64(count)

	if remainder != 0 {
		count++
	}

	seedMapfile, err := os.OpenFile(seedMapFileName, os.O_TRUNC|os.O_CREATE|os.O_RDWR, os.ModePerm)
	must(err)
	defer seedMapfile.Close()

	snapshotFile, err := os.OpenFile(snapshotFileName, os.O_TRUNC|os.O_CREATE|os.O_RDWR, os.ModePerm)
	must(err)
	defer snapshotFile.Close()

	for i := 0; i < count; i++ {

		dep := fundsPerAddr
		if remainder != 0 && i+1 == count {
			dep = remainder
		}

		seed, addr := seedAndFirstAddr()
		_, err := fmt.Fprintln(seedMapfile, seed, addr, dep)
		must(err)

		_, err = snapshotFile.WriteString(fmt.Sprintf("%s;%d\n", addr, dep))
		must(err)
	}
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
