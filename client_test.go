package soroban_test

import (
	"testing"

	"github.com/sebamiro/soroban"
	"github.com/sebamiro/soroban/internal/rpc"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
)

const (
	HELLO_WORD_WASM = "test/hello_world.wasm"

	SEED = "SDBIZIYGYODMURTQIGFRK2NRIVOVOOS7DE5HGYXOBRTN3GA7G6QZX672"
)

var client = soroban.Client{Client: rpc.Client{URL: LocalNetwork}, PassPhrase: LocalPassphrase}

func TestSimulateTransaction(t *testing.T) {
	contractid := []byte("CAOCKSQN7D2XXP3XEYYPB3F6SGMYUNTBYSDCCML6QJYJ75H2KNZ3I23Z")
	contractIDAddress := xdr.ScAddress{
		Type:       xdr.ScAddressTypeScAddressTypeContract,
		ContractId: (*xdr.Hash)(contractid),
	}

	world := xdr.ScString("world")
	firstParamScVal := xdr.ScVal{
		Type: xdr.ScValTypeScvString,
		Str:  &world,
	}

	kp, _ := keypair.Parse(SEED)
	signer := kp.(*keypair.Full)
	account := txnbuild.NewSimpleAccount(signer.Address(), int64(40385577484298))

	invokeHostFunctionOp := &txnbuild.InvokeHostFunction{
		HostFunction: xdr.HostFunction{
			Type: xdr.HostFunctionTypeHostFunctionTypeInvokeContract,
			InvokeContract: &xdr.InvokeContractArgs{
				ContractAddress: contractIDAddress,
				FunctionName:    "hello",
				Args: xdr.ScVec{
					firstParamScVal,
				},
			},
		},
		SourceAccount: account.AccountID,
	}

	param := txnbuild.TransactionParams{
		SourceAccount:        &account,
		Operations:           []txnbuild.Operation{invokeHostFunctionOp},
		BaseFee:              txnbuild.MinBaseFee,
		IncrementSequenceNum: true,
		Preconditions: txnbuild.Preconditions{
			TimeBounds: txnbuild.NewInfiniteTimeout(),
		},
	}

	tx, err := txnbuild.NewTransaction(param)
	if err != nil {
		t.Fatal(err)
	}
	tx, err = tx.Sign(network.TestNetworkPassphrase, signer)
	if err != nil {
		t.Fatal(err)
	}

	r, err := client.SendTransaction(tx)
	if err != nil {
		t.Fatal(err)
	}

	hash, _ := tx.HashHex(network.TestNetworkPassphrase)
	t.Log(r.Hash)
	t.Log(hash)

	t.Log(r)
	var errRes xdr.TransactionResult
	err = xdr.SafeUnmarshalBase64(r.ErrorResultXdr, &errRes)
	if err != nil {
		t.Fatal(err)
	}
	t.Fatal(errRes)
}

func TestGetHealth(t *testing.T) {
	r, err := client.GetHealth()
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != "healthy" {
		t.Fatal(err)
	}
}
