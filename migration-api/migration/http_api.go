package migration

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"sort"
	"strconv"

	"github.com/iotaledger/chrysalis-tools/common"
	"github.com/iotaledger/iota.go/address"
	"github.com/iotaledger/iota.go/api"
	"github.com/iotaledger/iota.go/consts"
	"github.com/iotaledger/iota.go/encoding/b1t6"
	"github.com/iotaledger/iota.go/v2"
	"github.com/labstack/echo/v4"
)

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

// NewHTTPAPIService creates a new HTTPAPIService.
func NewHTTPAPIService(e *echo.Echo, listenAddr string, cfg *HTTPAPIServiceConfig) *HTTPAPIService {
	return &HTTPAPIService{cfg: cfg, listenAddr: listenAddr, e: e}
}

// HTTPAPIService serves an API to query for migration related data.
type HTTPAPIService struct {
	cfg        *HTTPAPIServiceConfig
	e          *echo.Echo
	listenAddr string
}

// Init does nothing.
func (httpAPI *HTTPAPIService) Init() error {
	return nil
}

// Run starts the API.
func (httpAPI *HTTPAPIService) Run() error {
	log.Println("running HTTP API service")

	// create APIs
	legacyAPI, err := api.ComposeAPI(api.HTTPClientSettings{
		URI: httpAPI.cfg.LegacyNode.URI,
		Client: &http.Client{
			Timeout: httpAPI.cfg.LegacyNode.Timeout,
		},
	})
	if err != nil {
		return fmt.Errorf("unable to build legacy API: %w", err)
	}

	c2API := iotago.NewNodeAPIClient(httpAPI.cfg.C2Node.URI,
		iotago.WithNodeAPIClientHTTPClient(&http.Client{Timeout: httpAPI.cfg.C2Node.Timeout}),
	)

	httpAPI.e.GET("/state", func(c echo.Context) error {

		state := &StateResponse{}

		treasuryRes, err := c2API.Treasury()
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("unable to query treasury: %v", err))
		}

		state.TreasuryTokens = treasuryRes.Amount
		state.TokensMigrated = consts.TotalSupply - treasuryRes.Amount

		legacyNodeInfo, err := legacyAPI.GetNodeInfo()
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("unable to query node info from legacy node: %v", err))
		}

		ledgerQueryRes, err := common.QueryLedgerState(httpAPI.cfg.LegacyNode.URI, int(legacyNodeInfo.LatestSolidSubtangleMilestoneIndex))
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("unable to query ledger state from legacy node for milestone %d: %v", legacyNodeInfo.LatestSolidSubtangleMilestoneIndex, err))
		}

		var totalLocked uint64
		for addr, balance := range ledgerQueryRes.Balances {
			if balance < uint64(httpAPI.cfg.MinTokenAmountForMigration) {
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

	httpAPI.e.GET("/receipts/integrity", func(c echo.Context) error {
		receipts, err := c2API.Receipts()
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("unable to retrieve receipts from C2 node: %v", err))
		}

		seenTailTx := make(map[string]struct{})
		for _, receipt := range receipts {
			for _, seri := range receipt.Receipt.Funds {
				entry := seri.(*iotago.MigratedFundsEntry)
				if _, seen := seenTailTx[string(entry.TailTransactionHash[:])]; seen {
					tailTxTrytes := b1t6.EncodeToTrytes(entry.TailTransactionHash[:])
					tailTxHex := hex.EncodeToString(entry.TailTransactionHash[:])
					return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("tail tx %s / %s was migrated multiple times", tailTxTrytes, tailTxHex))
				}
				seenTailTx[string(entry.TailTransactionHash[:])] = struct{}{}
			}
		}

		return c.String(http.StatusOK, "integrity ok")
	})

	httpAPI.e.GET("/recentlyLocked/:numEntries", func(c echo.Context) error {
		numEntriesWanted, err := strconv.Atoi(c.Param("numEntries"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("unable to parse numEntries parameter: %v", err))
		}

		// use the latest solid milestone as a base and then go back a max amount of milestones to collect entries
		nodeInfo, err := legacyAPI.GetNodeInfo()
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("unable to parse numEntries parameter: %v", err))
		}

		target := nodeInfo.LatestSolidSubtangleMilestoneIndex - int64(httpAPI.cfg.MaxMilestonesToQueryForEntries)
		switch {
		case target < 0:
			target = 0
		}

		// we can't check for the pruning index since it is not part of the legacy node info
		var funds []Funds

	out:
		for msIndex := nodeInfo.LatestSolidSubtangleMilestoneIndex; msIndex > target; msIndex-- {
			res, err := common.QueryLedgerDiffExtended(httpAPI.cfg.LegacyNode.URI, int(msIndex))
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("unable to extended ledger diff for milestone %d: %v", msIndex, err))
			}

			for _, tx := range res.ConfirmedTxWithValue {
				if tx.Value <= 0 {
					continue
				}

				edAddr, err := address.ParseMigrationAddress(tx.Address)
				if err != nil {
					continue
				}

				funds = append(funds, Funds{
					TailTransactionHash:  tx.TailTxHash,
					Value:                uint64(tx.Value),
					TargetEd25519Address: hex.EncodeToString(edAddr[:]),
				})

				if len(funds) == numEntriesWanted {
					break out
				}
			}
		}

		return c.JSON(http.StatusOK, funds)
	})

	httpAPI.e.GET("/recentlyMinted/:numReceipts", func(c echo.Context) error {
		receipts, err := c2API.Receipts()
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("unable to retrieve receipts from C2 node: %v", err))
		}

		numReceiptsWanted, err := strconv.Atoi(c.Param("numReceipts"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("unable to parse numReceipts parameter: %v", err))
		}

		if len(receipts) == 0 {
			return c.JSON(http.StatusOK, make([]*RecentReceipt, 0))
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

	if err := httpAPI.e.Start(httpAPI.listenAddr); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}

	return nil
}

// Shutdown shuts down the service.
func (httpAPI *HTTPAPIService) Shutdown(ctx context.Context) error {
	log.Println("shutting down HTTP API service...")
	if err := httpAPI.e.Shutdown(ctx); err != nil {
		return err
	}
	return nil
}
