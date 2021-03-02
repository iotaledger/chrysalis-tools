package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/iotaledger/iota.go/trinary"
)

type (
	GetLedgerStateReturn struct {
		Balances       map[trinary.Hash]uint64 `json:"balances"`
		MilestoneIndex uint32                  `json:"milestoneIndex"`
		Duration       int                     `json:"duration"`
	}
)

// QueryLedgerState queries for the ledger state given the legacy node URI and target LSMI.
func QueryLedgerState(legacyNodeURI string, lsmi int) (*GetLedgerStateReturn, error) {
	req := buildLegacyRequest(legacyNodeURI, fmt.Sprintf(`{"command": "getLedgerState", "targetIndex": %d}`, lsmi))
	http.DefaultClient.Timeout = 0
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unable to query ledger state: %w", err)
	}
	defer res.Body.Close()

	var resObj GetLedgerStateReturn
	jsonRes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read response body from ledger state query response: %w", err)
	}

	if err := json.Unmarshal(jsonRes, &resObj); err != nil {
		return nil, fmt.Errorf("unable to JSON unmarshal ledger state query response: %w", err)
	}

	return &resObj, nil
}

type (
	GetLedgerDiffExtReturn struct {
		ConfirmedTxWithValue      []*TxHashWithValue     `json:"confirmedTxWithValue"`
		ConfirmedBundlesWithValue []*BundleWithValue     `json:"confirmedBundlesWithValue"`
		Diff                      map[trinary.Hash]int64 `json:"diff"`
		MilestoneIndex            uint32                 `json:"milestoneIndex"`
		Duration                  int                    `json:"duration"`
	}

	TxHashWithValue struct {
		TxHash     trinary.Hash `mapstructure:"txHash"`
		TailTxHash trinary.Hash `mapstructure:"tailTxHash"`
		BundleHash trinary.Hash `mapstructure:"bundleHash"`
		Address    trinary.Hash `mapstructure:"address"`
		Value      int64        `mapstructure:"value"`
	}

	BundleWithValue struct {
		BundleHash trinary.Hash   `mapstructure:"bundleHash"`
		TailTxHash trinary.Hash   `mapstructure:"tailTxHash"`
		LastIndex  uint64         `mapstructure:"lastIndex"`
		Txs        []*TxWithValue `mapstructure:"txs"`
	}

	TxWithValue struct {
		TxHash  trinary.Hash `mapstructure:"txHash"`
		Address trinary.Hash `mapstructure:"address"`
		Index   uint64       `mapstructure:"index"`
		Value   int64        `mapstructure:"value"`
	}
)

// QueryLedgerDiffExtended queries for an extended ledger diff of a given milestone.
func QueryLedgerDiffExtended(legacyNodeURI string, milestoneIndex int) (*GetLedgerDiffExtReturn, error) {
	req := buildLegacyRequest(legacyNodeURI, fmt.Sprintf(`{"command": "getLedgerDiffExt", "milestoneIndex": %d}`, milestoneIndex))
	http.DefaultClient.Timeout = 0
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unable to query ledger extended diff: %w", err)
	}
	defer res.Body.Close()

	var resObj GetLedgerDiffExtReturn
	jsonRes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read response body from ledger extended diff response: %w", err)
	}

	if err := json.Unmarshal(jsonRes, &resObj); err != nil {
		return nil, fmt.Errorf("unable to JSON unmarshal ledger extended diff response: %w", err)
	}

	return &resObj, nil
}

// builds up a legacy node API request
func buildLegacyRequest(legacyNodeURI string, body string) *http.Request {
	return &http.Request{
		Method: http.MethodPost,
		URL: func() *url.URL {
			u, err := url.Parse(legacyNodeURI)
			if err != nil {
				panic(err)
			}
			return u
		}(),
		Header: map[string][]string{
			"Content-Type":       {"application/json"},
			"X-IOTA-API-Version": {"1"},
		},
		Body: func() io.ReadCloser {
			cmd := []byte(body)
			return ioutil.NopCloser(bytes.NewReader(cmd))
		}(),
	}
}
