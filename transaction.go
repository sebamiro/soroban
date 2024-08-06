package soroban

import (
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
)

type (
	Transaction struct {
		client *Client
		build  *transactionBuild
	}

	transactionBuild struct {
		source                     txnbuild.Account
		operations                 []txnbuild.Operation
		signers                    []*keypair.Full
		timeBounds                 txnbuild.TimeBounds
		ledgerBounds               *txnbuild.LedgerBounds
		minSequenceNumber          *int64
		minSequenceNumberAge       uint64
		minSequenceNumberLedgerGap uint32
		extraSigners               []string
		Memo                       txnbuild.Memo
		baseFee                    int64
		incrementSequenceNum       bool
		// sorobanData                *xdr.SorobanTransactionData
	}
)

func NewTransctionBuilder() *Transaction {
	return &Transaction{
		build: &transactionBuild{
			baseFee:              txnbuild.MinBaseFee,
			incrementSequenceNum: true,
		},
	}
}

func (t *Transaction) Client(c *Client) *Transaction {
	t.client = c
	return t
}

func (t *Transaction) SourceAccount(s txnbuild.Account) *Transaction {
	t.build.source = s
	return t
}

func (t *Transaction) Operation(ops ...txnbuild.Operation) *Transaction {
	t.build.operations = append(t.build.operations, ops...)
	return t
}

func (t *Transaction) Signer(signers ...*keypair.Full) *Transaction {
	t.build.signers = append(t.build.signers, signers...)
	return t
}

// Transaction is only valid during a certain time range (units are seconds).
func (t *Transaction) TimeBounds(tb txnbuild.TimeBounds) *Transaction {
	t.build.timeBounds = tb
	return t
}

// Transaction is valid for ledger numbers n such that minLedger <= n <
// maxLedger (if maxLedger == 0, then only minLedger is checked)
func (t *Transaction) LedgerBounds(lb *txnbuild.LedgerBounds) *Transaction {
	t.build.ledgerBounds = lb
	return t
}

// If nil, the transaction is only valid when sourceAccount's sequence
// number "N" is seqNum - 1. Otherwise, valid when N satisfies minSeqNum <=
// N < tx.seqNum.
func (t *Transaction) MinSequenceNumber(mn *int64) *Transaction {
	t.build.minSequenceNumber = mn
	return t
}

// Transaction is valid if the current ledger time is at least
// minSequenceNumberAge greater than the source account's seqTime (units are
// seconds).
func (t *Transaction) MinSequenceNumberAge(mn uint64) *Transaction {
	t.build.minSequenceNumberAge = mn
	return t
}

// Transaction is valid if the current ledger number is at least
// minSequenceNumberLedgerGap greater than the source account's seqLedger.
func (t *Transaction) MinSequenceNumberLedgerGap(mn uint32) *Transaction {
	t.build.minSequenceNumberLedgerGap = mn
	return t
}

// Transaction is valid if there is a signature corresponding to every
// Signer in this array, even if the signature is not otherwise required by
// the source account or operations.
func (t *Transaction) ExtraSigner(s ...string) *Transaction {
	t.build.extraSigners = append(t.build.extraSigners, s...)
	return t
}

func (t *Transaction) Memo(m txnbuild.Memo) *Transaction {
	t.build.Memo = m
	return t
}

func (t *Transaction) BaseFee(f int64) *Transaction {
	t.build.baseFee = f
	return t
}

// Authorizationa sets Soroban Authorization. Its only possible if there is only one
// InvokeFunctionOperation, else does nothing
func (t *Transaction) Authorization(auth []xdr.SorobanAuthorizationEntry) *Transaction {
	op, ok := t.build.operations[0].(*txnbuild.InvokeHostFunction)
	if ok {
		op.Auth = auth
	}
	return t
}

// Authorizationa sets Soroban Authorization. Its only possible if there is only one
// InvokeFunctionOperation, else does nothing
func (t *Transaction) SorobanData(data xdr.SorobanTransactionData) *Transaction {
	op, ok := t.build.operations[0].(*txnbuild.InvokeHostFunction)
	if ok {
		op.Ext = xdr.TransactionExt{
			V:           1,
			SorobanData: &data,
		}
	}
	return t
}

func (t *Transaction) Simulate() (*SimulateTransactionResult, error) {
	increase := t.build.incrementSequenceNum
	t.build.incrementSequenceNum = false
	tx, err := t.buildTx()
	t.build.incrementSequenceNum = increase
	if err != nil {
		return nil, err
	}
	return t.client.SimulateTransaction(tx)
}

func (t *Transaction) Send() (*SendTransactionResult, error) {
	tx, err := t.buildTx()
	if err != nil {
		return nil, err
	}
	tx, err = tx.Sign(t.client.PassPhrase, t.build.signers...)
	if err != nil {
		return nil, err
	}
	return t.client.SendTransaction(tx)
}

func (t *Transaction) buildTx() (*txnbuild.Transaction, error) {
	precondirtions := txnbuild.Preconditions{
		TimeBounds:                 t.build.timeBounds,
		LedgerBounds:               t.build.ledgerBounds,
		MinSequenceNumber:          t.build.minSequenceNumber,
		MinSequenceNumberAge:       t.build.minSequenceNumberAge,
		MinSequenceNumberLedgerGap: t.build.minSequenceNumberLedgerGap,
		ExtraSigners:               t.build.extraSigners,
	}
	params := txnbuild.TransactionParams{
		SourceAccount:        t.build.source,
		Operations:           t.build.operations,
		Preconditions:        precondirtions,
		BaseFee:              t.build.baseFee,
		IncrementSequenceNum: t.build.incrementSequenceNum,
	}
	return txnbuild.NewTransaction(params)
}
