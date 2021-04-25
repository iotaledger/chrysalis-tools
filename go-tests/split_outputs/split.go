package main

import (
	"flag"
	. "github.com/iotaledger/chrysalis-tools/go-tests/lib"
)

func main() {
	nodeDomain, apiPort := DefineNodeFlags()
	amount := flag.Uint64("amount", 7_936_447_619_000, "How much iotas should be split to each output")
	flag.Parse()
	nodeAPI, nodeInfo := ObtainAPI(*nodeDomain, *apiPort)

	SplitFunds(nodeAPI, nodeInfo, amount)
}

