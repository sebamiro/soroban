package soroban

import (
	"errors"
	"fmt"
	"math"
	"net/http"

	"github.com/stellar/go/xdr"
)

type Account struct {
	AccountId            string            `json:"account_id"`
	Sequence             int64             `json:"sequence,string"`
	SubentryCount        int32             `json:"subentry_count"`
	InflationDestination string            `json:"inflation_destination,omitempty"`
	HomeDomain           string            `json:"home_domain,omitempty"`
	Thresholds           AccountThresholds `json:"thresholds"`
	Flags                AccountFlags      `json:"flags"`
	Balance              int64             `json:"balance"` // int stroops
	Signers              []Signer          `json:"signers"`
	NumSponsored         uint32            `json:"sponsored_count"`
	NumSponsoring        uint32            `json:"sponsoring_count"`
	SignerSponsoringIDs  []string          `json:"singer_sponsoring_ids" xdrmaxsize:"20"`
	SeqLedger            uint32            `json:"sequence_ledger"`
	SeqTime              uint64            `json:"sequence_time"`
}

// GetAccountID returns the Stellar account ID. This is to satisfy the
// Account interface of txnbuild.
func (a Account) GetAccountID() string {
	return a.AccountId
}

// GetSequenceNumber returns the sequence number of the account,
// and returns it as a 64-bit integer.
func (a Account) GetSequenceNumber() (int64, error) {
	return a.Sequence, nil
}

// IncrementSequenceNumber increments the internal record of the account's sequence
// number by 1. This is typically used after a transaction build so that the next
// transaction to be built will be valid.
func (a *Account) IncrementSequenceNumber() (int64, error) {
	if a.Sequence == math.MaxInt64 {
		return 0, fmt.Errorf("sequence cannot be increased, it already reached MaxInt64 (%d)", int64(math.MaxInt64))
	}
	a.Sequence++
	return a.Sequence, nil
}

// SignerSummary returns a map of signer's keys to weights.
func (a *Account) SignerSummary() map[string]int32 {
	m := map[string]int32{}
	for _, s := range a.Signers {
		m[s.Key] = s.Weight
	}
	return m
}

type Signer struct {
	Weight int32  `json:"weight"`
	Key    string `json:"key"`
}

type AccountThresholds struct {
	LowThreshold  byte `json:"low_threshold"`
	MedThreshold  byte `json:"med_threshold"`
	HighThreshold byte `json:"high_threshold"`
}

type AccountFlags struct {
	AthRequired        bool `json:"auth_required"`
	AthRevocable       bool `json:"auth_revocable"`
	AthImmutable       bool `json:"auth_immutable"`
	AthClawbackEnabled bool `json:"auth_clawback_enabled"`
}

// GetAccountEntry returns the ledger entry of the sourceAccount
func (c Client) GetAccountEntry(publicKey string) (*xdr.AccountEntry, error) {
	acountId := xdr.MustAddress(publicKey)
	key := xdr.LedgerKey{
		Account: &xdr.LedgerKeyAccount{
			AccountId: acountId,
		},
	}
	base64Key, err := key.MarshalBinaryBase64()
	if err != nil {
		return nil, err
	}
	res, err := c.GetLedgerEntries(base64Key)
	if err != nil {
		return nil, err
	}
	if len(res.Entries) < 1 {
		return nil, errors.New("Account not found")
	}
	var ledgerEntry xdr.LedgerEntryData
	err = xdr.SafeUnmarshalBase64(res.Entries[0].Xdr, &ledgerEntry)
	if err != nil {
		return nil, err
	}
	return ledgerEntry.Account, nil
}

// GetAccount returns a txnbuild.Account interface retrive from AccountEntry
func (c Client) GetAccount(publicKey string) (account *Account, err error) {
	accountEntry, err := c.GetAccountEntry(publicKey)
	if err != nil {
		return nil, err
	}
	inflationDestination, err := accountEntry.InflationDest.GetAddress()
	if err != nil {
		return nil, err
	}
	account = &Account{
		AccountId:            publicKey,
		Sequence:             int64(accountEntry.SeqNum),
		SubentryCount:        int32(accountEntry.NumSubEntries),
		InflationDestination: inflationDestination,
		HomeDomain:           string(accountEntry.HomeDomain),
		Thresholds: AccountThresholds{
			LowThreshold:  accountEntry.ThresholdLow(),
			MedThreshold:  accountEntry.ThresholdMedium(),
			HighThreshold: accountEntry.ThresholdHigh(),
		},
		Flags: AccountFlags{
			AthRequired:        xdr.AccountFlags(accountEntry.Flags).IsAuthRequired(),
			AthRevocable:       xdr.AccountFlags(accountEntry.Flags).IsAuthRevocable(),
			AthImmutable:       xdr.AccountFlags(accountEntry.Flags).IsAuthImmutable(),
			AthClawbackEnabled: xdr.AccountFlags(accountEntry.Flags).IsAuthClawbackEnabled(),
		},
		Balance: int64(accountEntry.Balance),
		Signers: make([]Signer, 0),
		// NumSponsored:        uint32(accountEntry.Ext.V1.Ext.V2.NumSponsored),
		// NumSponsoring:       uint32(accountEntry.Ext.V1.Ext.V2.NumSponsoring),
		SignerSponsoringIDs: make([]string, 0),
		// SeqLedger:           uint32(accountEntry.Ext.V1.Ext.V2.Ext.V3.SeqLedger),
		// SeqTime:             uint64(accountEntry.Ext.V1.Ext.V2.Ext.V3.SeqTime),
	}
	for _, s := range accountEntry.Signers {
		account.Signers = append(account.Signers, Signer{
			Key:    s.Key.Address(),
			Weight: int32(s.Weight),
		})
	}
	for _, s := range accountEntry.SignerSponsoringIDs() {
		account.SignerSponsoringIDs = append(account.SignerSponsoringIDs, s.Address())
	}
	return
}

// Fund funds the publicKey recived. It only works with test networks.
// If FriendbotURL is not set, it will get it from the network.
func (c *Client) Fund(publicKey string) error {
	if c.FriendbotURL == "" {
		network, err := c.GetNetwork()
		if err != nil {
			return err
		}
		c.FriendbotURL = network.FriendbotURL
	}
	friendbotURL := fmt.Sprintf("%s?addr=%s", c.FriendbotURL, publicKey)
	req, err := http.NewRequest("GET", friendbotURL, nil)
	if err != nil {
		return err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return errors.New("Bad status code:" + res.Status)
	}
	return nil
}
