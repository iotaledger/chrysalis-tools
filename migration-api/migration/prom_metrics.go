package migration

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/iotaledger/chrysalis-tools/common"
	"github.com/iotaledger/iota.go/api"
	iotago "github.com/iotaledger/iota.go/v2"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// NewPromMetricsService creates a new PromMetricsService with the given options.
func NewPromMetricsService(e *echo.Echo, cfg *PromMetricsServiceConfig) *PromMetricsService {
	s := &PromMetricsService{
		cfg:      cfg,
		e:        e,
		registry: prometheus.NewRegistry(),
		shutdown: make(chan struct{}),
	}
	return s
}

// PromMetricsService is in charge of fetching and preparing metrics to be queried by prometheus.
type PromMetricsService struct {
	e                     *echo.Echo
	cfg                   *PromMetricsServiceConfig
	state                 *prommetricservicestate
	legacyAPI             *api.API
	c2API                 *iotago.NodeAPIClient
	registry              *prometheus.Registry
	legacyWfTailsIncluded prometheus.Counter
	receiptEntriesApplied prometheus.Counter
	serviceErrors         prometheus.Counter
	shutdown              chan struct{}
}

// represents the state of the prom metrics service.
type prommetricservicestate struct {
	// The last milestone queried for white-flag confirmation data.
	LastLegacyMilestoneIndexQueried int `json:"lastLegacyMilestoneIndexQueried"`
	// The last milestone queried for receipt data.
	LastC2MilestoneIndexQueried int `json:"lastC2MilestoneIndexQueried"`
	// The persisted counter of confirmed legacy tail txs.
	LegacyTailsIncluded int `json:"legacyTailsIncluded"`
	// The persisted counter of applied receipt entries.
	ReceiptEntriesApplied int `json:"receiptEntriesApplied"`
}

// persists the state to the given file (overriding a previous state file).
func (pmss *prommetricservicestate) persist(filePath string) error {
	jsonState, err := json.MarshalIndent(pmss, "", "   ")
	if err != nil {
		return fmt.Errorf("unable to serialize state: %w", err)
	}
	if err := os.WriteFile(filePath, jsonState, 0666); err != nil {
		return fmt.Errorf("unable to write state to disk: %w", err)
	}
	return nil
}

// loads the state from the given file.
func (pmss *prommetricservicestate) load(filePath string) error {
	jsonState, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("unable to read state file: %w", err)
	}
	if err := json.Unmarshal(jsonState, pmss); err != nil {
		return fmt.Errorf("unable to deserialize state: %w", err)
	}
	return nil
}

// Init initializes the service state.
func (pms *PromMetricsService) Init() error {
	pms.state = &prommetricservicestate{}

	if _, err := os.Stat(pms.cfg.StateFilePath); os.IsNotExist(err) {
		log.Println("bootstrapping prom metrics service")
		pms.state.LastLegacyMilestoneIndexQueried = pms.cfg.LegacyMilestoneStartIndex
		pms.state.LastC2MilestoneIndexQueried = pms.cfg.C2MilestoneStartIndex
		if err := pms.state.persist(pms.cfg.StateFilePath); err != nil {
			return fmt.Errorf("unable to persist initial state: %w", err)
		}
	} else if err := pms.state.load(pms.cfg.StateFilePath); err != nil {
		return err
	}

	pms.legacyWfTailsIncluded = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: pms.cfg.CounterNames.IncludedLegacyTails,
			Help: "The count of tails included.",
		},
	)
	pms.receiptEntriesApplied = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: pms.cfg.CounterNames.AppliedReceiptEntries,
			Help: "The count of applied receipt entries.",
		},
	)
	pms.serviceErrors = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: pms.cfg.CounterNames.ServiceErrors,
			Help: "The count of encountered errors during the service's lifetime.",
		},
	)

	pms.registry.MustRegister(pms.legacyWfTailsIncluded, pms.receiptEntriesApplied, pms.serviceErrors)

	pms.legacyWfTailsIncluded.Add(float64(pms.state.LegacyTailsIncluded))
	pms.receiptEntriesApplied.Add(float64(pms.state.ReceiptEntriesApplied))
	pms.serviceErrors.Add(0)

	var err error
	pms.legacyAPI, err = api.ComposeAPI(api.HTTPClientSettings{
		URI:    pms.cfg.LegacyNode.URI,
		Client: &http.Client{Timeout: pms.cfg.LegacyNode.Timeout},
	})
	if err != nil {
		return fmt.Errorf("unable to init legcy API: %w", err)
	}

	pms.c2API = iotago.NewNodeAPIClient(pms.cfg.C2Node.URI, iotago.WithNodeAPIClientHTTPClient(&http.Client{
		Timeout: pms.cfg.C2Node.Timeout,
	}))

	return nil
}

// Run instructs the service to periodically update its state and update the metrics it exposes.
func (pms *PromMetricsService) Run() error {
	log.Println("running prometheus metrics service")

	if pms.state == nil {
		panic("Init() must be called before Run()")
	}

	pms.e.GET("/metrics", func(c echo.Context) error {
		handler := promhttp.HandlerFor(
			pms.registry,
			promhttp.HandlerOpts{EnableOpenMetrics: true},
		)
		handler.ServeHTTP(c.Response().Writer, c.Request())
		return nil
	})

	for {
		select {
		case <-pms.shutdown:
			return nil
		case <-time.After(pms.cfg.FetchInterval):
			if err := pms.update(); err != nil {
				log.Printf("unable to update prometheus metrics: %v", err)
				pms.serviceErrors.Inc()
			}
		}
	}
}

// Shutdown shuts down the service.
func (pms *PromMetricsService) Shutdown(ctx context.Context) error {
	log.Println("shutting down prometheus metrics service...")
	select {
	case pms.shutdown <- struct{}{}:
	case <-ctx.Done():
	}
	return nil
}

// update updates the prometheus counters and then persists the service state.
func (pms *PromMetricsService) update() error {

	tailsIncluded, legacyMsTargetIndex, err := pms.queryIncludedTails()
	if err != nil {
		return err
	}

	receiptEntriesApplied, c2MsTargetIndex, err := pms.queryC2NodeReceipts()
	if err != nil {
		return err
	}

	pms.state.LastLegacyMilestoneIndexQueried = legacyMsTargetIndex
	pms.state.LastC2MilestoneIndexQueried = c2MsTargetIndex
	pms.state.ReceiptEntriesApplied += receiptEntriesApplied
	pms.state.LegacyTailsIncluded += tailsIncluded
	if err := pms.state.persist(pms.cfg.StateFilePath); err != nil {
		return err
	}

	pms.legacyWfTailsIncluded.Add(float64(tailsIncluded))
	pms.receiptEntriesApplied.Add(float64(receiptEntriesApplied))

	if pms.cfg.Debug {
		jsonState, err := json.MarshalIndent(pms.state, "", "   ")
		if err != nil {
			return err
		}
		log.Printf("updated prom metrics service state:\n %v", string(jsonState))
	}

	return nil
}

// queries the amount of newly included tails since the last queried legacy milestone.
func (pms *PromMetricsService) queryIncludedTails() (int, int, error) {
	legacyInfo, err := pms.legacyAPI.GetNodeInfo()
	if err != nil {
		return 0, 0, fmt.Errorf("unable to query info from legacy node: %w", err)
	}

	legacyMsTarget := legacyInfo.LatestSolidSubtangleMilestoneIndex
	if int(legacyMsTarget) == pms.state.LastLegacyMilestoneIndexQueried {
		return 0, int(legacyMsTarget), nil
	}

	var tailsIncluded int

	if pms.cfg.Debug {
		log.Printf("querying white-flag from %d to %d", pms.state.LastLegacyMilestoneIndexQueried+1, legacyMsTarget)
	}

	for i := pms.state.LastLegacyMilestoneIndexQueried + 1; i <= int(legacyMsTarget); i++ {
		if pms.cfg.Debug {
			log.Printf("querying white-flag data of %d", i)
		}
		wfData, err := common.QueryWhiteFlagConfirmation(pms.cfg.LegacyNode.URI, i)
		if err != nil {
			return 0, 0, fmt.Errorf("unable to query white-flag confirmation for legacy milestone %d: %w", i, err)
		}
		if pms.cfg.Debug {
			log.Printf("white-flag-data of %d - tails included %d", i, len(wfData.IncludedBundles))
		}
		tailsIncluded += len(wfData.IncludedBundles)
	}

	return tailsIncluded, int(legacyMsTarget), nil
}

// queries the amount of newly applied receipt entries since the last queried C2 milestone.
func (pms *PromMetricsService) queryC2NodeReceipts() (int, int, error) {
	c2Info, err := pms.c2API.Info()
	if err != nil {
		return 0, 0, fmt.Errorf("unable to query info from c2 node: %w", err)
	}

	var receiptEntriesApplied int

	c2MsTarget := c2Info.ConfirmedMilestoneIndex
	if int(c2MsTarget) == pms.state.LastC2MilestoneIndexQueried {
		return 0, int(c2MsTarget), nil
	}

	if pms.cfg.Debug {
		log.Printf("querying milestones/receipts from %d to %d", pms.state.LastC2MilestoneIndexQueried+1, c2MsTarget)
	}

	for i := pms.state.LastC2MilestoneIndexQueried + 1; i <= int(c2MsTarget); i++ {
		if pms.cfg.Debug {
			log.Printf("querying C2 milestone %d", i)
		}
		msRes, err := pms.c2API.MilestoneByIndex(uint32(i))
		if err != nil {
			return 0, 0, fmt.Errorf("unable to query milestone %d from c2 node: %w", i, err)
		}

		msgIDBytes, err := hex.DecodeString(msRes.MessageID)
		if err != nil {
			return 0, 0, fmt.Errorf("unable to convert milestone %d's msg hex ID: %w", i, err)
		}
		var msID iotago.MessageID
		copy(msID[:], msgIDBytes)

		msg, err := pms.c2API.MessageByMessageID(msID)
		if err != nil {
			return 0, 0, fmt.Errorf("unable to query msg containing milestone %d from c2 node: %w", i, err)
		}

		milestone := msg.Payload.(*iotago.Milestone)
		if milestone.Receipt == nil {
			continue
		}

		receipt := milestone.Receipt.(*iotago.Receipt)
		if pms.cfg.Debug {
			log.Printf("C2 milestone %d receipt has %d entries", i, len(receipt.Funds))
		}
		receiptEntriesApplied += len(receipt.Funds)
	}

	return receiptEntriesApplied, int(c2MsTarget), nil
}
