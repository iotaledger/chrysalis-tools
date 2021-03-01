package migration_api

import (
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/iotaledger/chrysalis-tools"
	"github.com/iotaledger/iota.go/address"
	"github.com/iotaledger/iota.go/api"
	"github.com/iotaledger/iota.go/consts"
	"github.com/iotaledger/iota.go/v2"
	"github.com/labstack/echo/v4"
)

var e = echo.New()

// StateResponse contains the information of a /state response.
type StateResponse struct {
	TreasuryTokens    uint64            `json:"treasuryTokens"`
	TokensMigrated    uint64            `json:"tokensMigrated"`
	LegacyFundsLocked LegacyFundsLocked `json:"legacyFundsLocked"`
}

// LegacyFundsLocked contains information about funds locked on the legacy network.
type LegacyFundsLocked struct {
	TokensTotal                   uint64  `json:"tokensTotal"`
	MigratedAddressesTotal        uint64  `json:"migratedAddressesTotal"`
	TokensPercentageOfTotalSupply float64 `json:"tokensPercentageOfTotalSupply"`
}

// Funds represents locked or already migrated funds.
type Funds struct {
	TailTransactionHash  string `json:"tailTransactionHash"`
	Value                uint64 `json:"value"`
	TargetEd25519Address string `json:"targetEd25519Address"`
}

// RecentReceipt entails the information about a recently issued receipt on the C2 network.
type RecentReceipt struct {
	EmbeddedMilestoneIndex uint32  `json:"embeddedMilestoneIndex"`
	LegacyMilestoneIndex   uint32  `json:"legacyMilestoneIndex"`
	Funds                  []Funds `json:"funds"`
}

// Starts the API.
func Start(config *Config) error {
	e.HideBanner = true

	// create APIs
	legacyAPI, err := api.ComposeAPI(api.HTTPClientSettings{
		URI: config.LegacyNode.URI,
		Client: &http.Client{
			Timeout: config.LegacyNode.Timeout,
		},
	})
	if err != nil {
		return fmt.Errorf("unable to build legacy API: %w", err)
	}

	c2API := iotago.NewNodeAPIClient(config.C2Node.URI,
		iotago.WithNodeAPIClientHTTPClient(&http.Client{Timeout: config.C2Node.Timeout}),
	)

	e.GET("/state", func(c echo.Context) error {

		state := &StateResponse{}

		treasuryRes, err := c2API.Treasury()
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("unable to query treasury: %w", err))
		}

		state.TreasuryTokens = treasuryRes.Amount
		state.TokensMigrated = consts.TotalSupply - treasuryRes.Amount

		legacyNodeInfo, err := legacyAPI.GetNodeInfo()
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("unable to query node info from legacy node: %w", err))
		}

		ledgerQueryRes, err := chrysalis_tools.QueryLedgerState(config.LegacyNode.URI, int(legacyNodeInfo.LatestSolidSubtangleMilestoneIndex))
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("unable to query ledger state from legacy node for milestone %d: %w", legacyNodeInfo.LatestSolidSubtangleMilestoneIndex, err))
		}

		var totalLocked uint64
		for addr, balance := range ledgerQueryRes.Balances {
			if balance < uint64(config.MinTokenAmountForMigration) {
				continue
			}

			if _, err := address.ParseMigrationAddress(addr); err != nil {
				continue
			}
			totalLocked += balance
			state.LegacyFundsLocked.MigratedAddressesTotal++
		}

		state.LegacyFundsLocked.TokensTotal = totalLocked
		state.LegacyFundsLocked.TokensPercentageOfTotalSupply = math.Floor((float64(totalLocked)/float64(consts.TotalSupply))*100) / 100

		return c.JSON(http.StatusOK, state)
	})

	e.GET("/recentlyLocked/:numEntries", func(c echo.Context) error {

		return nil
	})

	e.GET("/recentlyMigrated/:numReceipts", func(c echo.Context) error {
		receipts, err := c2API.Receipts()
		if err != nil {
			return fmt.Errorf("unable to retrieve receipts from C2 node: %w", err)
		}

		numReceiptsWanted, err := strconv.Atoi(c.Param("numReceipts"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Errorf("unable to parse numReceipts paramter: %w", err))
		}

		recentReceipts := make([]*RecentReceipt, numReceiptsWanted)
		sort.Slice(receipts, func(i, j int) bool {
			return receipts[i].MilestoneIndex > receipts[j].MilestoneIndex
		})

		if numReceiptsWanted > len(receipts) {
			numReceiptsWanted = len(receipts)
		}

		for i := 0; i < numReceiptsWanted; i++ {
			recentReceipts[i] = &RecentReceipt{
				EmbeddedMilestoneIndex: receipts[i].MilestoneIndex,
				LegacyMilestoneIndex:   receipts[i].Receipt.MigratedAt,
				Funds: func() []Funds {
					funds := make([]Funds, len(receipts[i].Receipt.Funds))
					for j, f := range receipts[i].Receipt.Funds {
						entry := f.(*iotago.MigratedFundsEntry)
						addr := entry.Address.(*iotago.Ed25519Address)
						funds[j] = Funds{
							TailTransactionHash:  hex.EncodeToString(entry.TailTransactionHash[:]),
							Value:                entry.Deposit,
							TargetEd25519Address: hex.EncodeToString(addr[:]),
						}
					}
					return funds
				}(),
			}
		}

		return c.JSON(http.StatusOK, recentReceipts)
	})

	if err := e.Start(config.ListenAddress); err != nil {
		return err
	}

	return nil
}

// Shutdown shuts down the API.
func Shutdown() error {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFunc()
	if err := e.Shutdown(ctx); err != nil {
		return err
	}
	return nil
}
