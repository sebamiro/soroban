package soroban

import (
	"encoding/json"

	"github.com/sebamiro/soroban/internal/rpc"
	"github.com/stellar/go/txnbuild"
)

// Client wrapper of rpc.Client
type Client struct {
	rpc.Client
	PassPhrase   string
	FriendbotURL string
}

// Methods
const (
	SendTransaction     = "sendTransaction"
	SimulateTransaction = "simulateTransaction"
	GetTransaction      = "getTransaction"
	GetHealth           = "getHealth"
	GetNetwork          = "getNetwork"
	GetLedgerEntries    = "getLedgerEntries"
)

type transaction struct {
	Transaction string `json:"transaction"`
}

// SendTransactionResult as defined in the docs https://developers.stellar.org/docs/data/rpc/api-reference/methods/sendTransaction
type SendTransactionResult struct {
	Hash                  string   `json:"hash"`
	Status                string   `json:"status"`
	LatestLedger          int64    `json:"latestLedger"`
	LatestLedgerCloseTime string   `json:"latestLedgerCloseTime"`
	ErrorResultXdr        string   `json:"errorResultXdr"`
	DiagnosticEventsXdr   []string `json:"diagnosticEventsXdr"`
}

// SendTransaction sends a signed transaction and returns its result.
// Returns an error if unmarshal, http call, etc; fail, NOT if the transaction faild.
// Result matches the result in the docs https://developers.stellar.org/docs/data/rpc/api-reference/methods/sendTransaction
func (c Client) SendTransaction(tx *txnbuild.Transaction) (*SendTransactionResult, error) {
	base64, err := tx.Base64()
	if err != nil {
		return nil, err
	}
	var sendTransactionResult SendTransactionResult
	err = c.CallResult(SendTransaction, &sendTransactionResult, transaction{base64})
	if err != nil {
		return nil, err
	}
	return &sendTransactionResult, nil
}

// SimulateTransactionResult as defined in the docs https://developers.stellar.org/docs/data/rpc/api-reference/methods/simulateTransaction
type SimulateTransactionResult struct {
	Error           string   `json:"error,omitempty"`
	TransactionData string   `json:"transactionData"`
	MinResourceFee  int64    `json:"minResourceFee,string"`
	LatestLedger    int64    `json:"latestLedger"`
	Events          []string `json:"events"`

	Results []struct {
		Auth []string `json:"auth"`
		XDR  string   `json:"xdr"`
	} `json:"results"`

	RestorePreamble struct {
		MinResourceFee  int64  `json:"minResourceFee,string"`
		TransactionData string `json:"transactionData"`
	} `json:"restorePreamble"`

	StateChange struct {
		Type   int    `json:"type"`
		Key    string `json:"key"`
		Before string `json:"before"`
		After  string `json:"after"`
	} `json:"stateChange"`
}

// SimulateTransaction simulates a transaction and returns its result.
// Returns an error if unmarshal, http call, etc; fail, NOT if the transaction faild.
// Result matches the result in the docs https://developers.stellar.org/docs/data/rpc/api-reference/methods/simulateTransaction
func (c Client) SimulateTransaction(tx *txnbuild.Transaction) (*SimulateTransactionResult, error) {
	base64, err := tx.Base64()
	if err != nil {
		return nil, err
	}
	var simulateTransactionResult SimulateTransactionResult
	err = c.CallResult(SimulateTransaction, &simulateTransactionResult, transaction{base64})
	if err != nil {
		return nil, err
	}
	return &simulateTransactionResult, nil
}

// GetTransactionResult as defined in the docs https://developers.stellar.org/docs/data/rpc/api-reference/methods/getTransaction
type GetTransactionResult struct {
	Status                string `json:"status"`
	LatestLedger          int64  `json:"latestLedger"`
	LatestLedgerCloseTime string `json:"latestLedgerCloseTime"`
	OldestLedger          int64  `json:"oldestLedger"`
	OldestLedgerCloseTime string `json:"oldestLedgerCloseTime"`
	Ledger                int64  `json:"ledger"`
	CreatedAt             string `json:"createdAt"`
	ApplicationOrder      int64  `json:"applicationOrder"`
	FeeBump               bool   `json:"feeBump"`
	EnvelopeXdr           string `json:"envelopeXdr"`
	ResultXdr             string `json:"resultXdr"`
	ResultMetaXdr         string `json:"resultMetaXdr"`
}

// GetTransaction provides details about the specified transaction.
// Returns an error if unmarshal, http call, etc; fail, NOT if the transaction faild.
// Result matches the result in the docs https://developers.stellar.org/docs/data/rpc/api-reference/methods/getTransaction
func (c Client) GetTransaction(hash string) (*GetTransactionResult, error) {
	var getTransactionResult GetTransactionResult
	err := c.CallResult(GetTransaction, &getTransactionResult, struct {
		Hash string `json:"hash"`
	}{hash})
	if err != nil {
		return nil, err
	}
	return &getTransactionResult, nil
}

// GetHealthResult as defined in the docs https://developers.stellar.org/docs/data/rpc/api-reference/methods/getHealth
type GetHealthResult struct {
	Status                string `json:"status"`
	LatestLedger          int64  `json:"latestLedger"`
	OldestLedger          int64  `json:"oldestLedger"`
	LedgerRetentionWindow int64  `json:"ledgerRetentionWindow"`
}

// GetHealth provides details about the health of the network.
// Returns an error if unmarshal, http call, etc; fail, NOT if the transaction faild.
// Result matches the result in the docs https://developers.stellar.org/docs/data/rpc/api-reference/methods/getHealth
func (c Client) GetHealth() (*GetHealthResult, error) {
	var getHealthResult GetHealthResult
	err := c.CallResult(GetHealth, &getHealthResult)
	if err != nil {
		return nil, err
	}
	return &getHealthResult, nil
}

type GetLedgerEntriesResult struct {
	LatestLedger int64             `json:"latestLedger"`
	Entries      []GetLedgerEntrie `json:"entries"`
}

type GetLedgerEntrie struct {
	Key                   string `json:"key"`
	Xdr                   string `json:"xdr"`
	LastModifiedLedgerSeq int64  `json:"lastModifiedLedgerSeq"`
	LiveUntilLedgerSeq    int64  `json:"liveUntilLedgerSeq"`
}

// GetLedgerEntries provides details about the health of the network.
// Result matches the result in the docs https://developers.stellar.org/docs/data/rpc/api-reference/methods/getLedgerEntries
func (c Client) GetLedgerEntries(keys ...string) (*GetLedgerEntriesResult, error) {
	var getLedgerEntriesResult GetLedgerEntriesResult
	err := c.CallResult(GetLedgerEntries, &getLedgerEntriesResult, struct {
		Keys []string `json:"keys"`
	}{keys})
	if err != nil {
		return nil, err
	}
	return &getLedgerEntriesResult, nil
}

// GetNetworkResult as defined in the docs https://developers.stellar.org/docs/data/rpc/api-reference/methods/getNetwork
type GetNetworkResult struct {
	Passphrase      string `json:"passphrase"`
	FriendbotURL    string `json:"friendbotUrl,omitempty"`
	ProtocolVersion int64  `json:"protocolVersion"`
}

// GetNetwork provides details about the the network.
// Returns an error if unmarshal, http call, etc; fail, NOT if the transaction faild.
// Result matches the result in the docs https://developers.stellar.org/docs/data/rpc/api-reference/methods/getNetwork
func (c Client) GetNetwork() (*GetNetworkResult, error) {
	var getNetworkResult GetNetworkResult
	err := c.CallResult(GetNetwork, &getNetworkResult)
	if err != nil {
		return nil, err
	}
	return &getNetworkResult, nil
}

// CallResult executes a call, with params if any, and saves the result into
// the interface passed as param.
func (c Client) CallResult(method string, result interface{}, params ...interface{}) error {
	resp, err := c.Call(method, params...)
	if err != nil {
		return err
	}
	err = json.Unmarshal(*resp.Result, result)
	if err != nil {
		return err
	}
	return nil
}
