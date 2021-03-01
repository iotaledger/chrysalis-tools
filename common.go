package chrysalis_tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/gohornet/hornet/plugins/webapi"
	"github.com/iotaledger/iota.go/trinary"
)

type LedgerQueryResponse struct {
	Balances       map[trinary.Hash]uint64 `json:"balances"`
	MilestoneIndex uint32                  `json:"milestoneIndex"`
	Duration       int                     `json:"duration"`
}

// QueryLedgerState queries for the ledger state given the legacy node URI and target LSMI.
func QueryLedgerState(legacyNodeURI string, lsmi int) (*LedgerQueryResponse, error) {
	req := buildLegacyRequest(legacyNodeURI, fmt.Sprintf(`{"command": "getLedgerState", "targetIndex": %d}`, lsmi))
	http.DefaultClient.Timeout = 0
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unable to query ledger state: %w", err)
	}
	defer res.Body.Close()

	var resObj LedgerQueryResponse
	jsonRes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read response body from ledger state query response: %w", err)
	}

	if err := json.Unmarshal(jsonRes, &resObj); err != nil {
		return nil, fmt.Errorf("unable to JSON unmarshal ledger state query response: %w", err)
	}

	return &resObj, nil
}

// QueryLedgerDiffExtended queries for an extended ledger diff of a given milestone.
func QueryLedgerDiffExtended(legacyNodeURI string, milestoneIndex int) (*webapi.GetLedgerDiffExtReturn, error) {
	req := buildLegacyRequest(legacyNodeURI, fmt.Sprintf(`{"command": "getLedgerDiffExt", "milestoneIndex": %d}`, milestoneIndex))
	http.DefaultClient.Timeout = 0
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unable to query ledger extended diff: %w", err)
	}
	defer res.Body.Close()

	var resObj webapi.GetLedgerDiffExtReturn
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
