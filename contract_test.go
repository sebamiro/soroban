package soroban_test

import (
	"os"
	"testing"
	"time"

	"github.com/sebamiro/soroban"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
	"github.com/stellar/go/xdr"
)

const (
	HelloWorldContract = "testdata/hello_world.wasm"

	LocalNetwork    = "http://localhost:8000/rpc"
	LocalFriendbot  = "http://localhost:8000/friendbot"
	LocalPassphrase = "Standalone Network ; February 2017"

	TestNetwork    = "https://soroban-testnet.stellar.org"
	TestPassphrase = network.TestNetworkPassphrase
	TestFriendbot  = "https://friendbot.stellar.org"
)

func WaitCompletedTransaction(c soroban.Client, hash string, maxAttempts int) (*soroban.GetTransactionResult, error) {
	for i := 0; i < maxAttempts; i++ {
		res, err := c.GetTransaction(hash)
		if err != nil {
			return nil, err
		}
		if res.Status != "NOT_FOUND" {
			return res, nil
		}
		time.Sleep(time.Duration(i) * 2 * time.Second)
	}
	return nil, nil
}

var (
	sorobanClient = soroban.Client{}
	contractWasm  []byte
)

func TestMain(t *testing.M) {
	sorobanClient.URL = TestNetwork
	sorobanClient.PassPhrase = TestPassphrase
	sorobanClient.FriendbotURL = TestFriendbot
	contractWasm, _ = os.ReadFile(HelloWorldContract)
	t.Run()
}


func TestInstallContract(t *testing.T) {
	pair, _ := keypair.Random()
	sorobanClient.Fund(pair.Address())
	account, err := sorobanClient.GetAccount(pair.Address())
	if err != nil {
		t.Fatal(err)
	}

	res, err := soroban.NewContract().
		Wasm(contractWasm).
		Client(&sorobanClient).
		Salt("a1").
		SourceAccount(account).
		KeyPair(pair).
		Install()
	if err != nil {
		t.Fatal(err)
	}

	completed, err := WaitCompletedTransaction(sorobanClient, res.Hash, 10)
	if err != nil {
		t.Fatal(err)
	}
	if completed.Status != "SUCCESS" {
		t.Fatal(completed)
	}
}

func TestCreateContract(t *testing.T) {
	pair, _ := keypair.Random()
	err := sorobanClient.Fund(pair.Address())
	if err != nil {
		t.Fatal(err)
	}
	account, err := sorobanClient.GetAccount(pair.Address())
	if err != nil {
		t.Fatal(err)
	}

	res, err := soroban.NewContract().
		Wasm(contractWasm).
		Client(&sorobanClient).
		Salt("a1").
		SourceAccount(account).
		KeyPair(pair).
		Deploy()
	if err != nil {
		t.Fatal(err)
	}
	completed, err := WaitCompletedTransaction(sorobanClient, res.Hash, 10)
	if err != nil {
		t.Fatal(err)
	}
	if completed.Status != "SUCCESS" {
		t.Fatal(completed)
	}
}


var pair *keypair.Full
var account *soroban.Account

func TestDeployContract(t *testing.T) {
	pair, _ = keypair.Random()
	sorobanClient.Fund(pair.Address())
	account, _ = sorobanClient.GetAccount(pair.Address())

	contract := soroban.NewContract().
		Wasm(contractWasm).
		Client(&sorobanClient).
		Salt("TestDeployContract").
		SourceAccount(account).
		KeyPair(pair)

	res, err := contract.Install()
	completed, err := WaitCompletedTransaction(sorobanClient, res.Hash, 10)
	if err != nil {
		t.Fatal(err)
	}
	if completed.Status != "SUCCESS" {
		t.Fatal(completed)
	}
	res, err = contract.Deploy()
	if err != nil {
		t.Fatal(err)
	}
	completed, err = WaitCompletedTransaction(sorobanClient, res.Hash, 10)
	if err != nil {
		t.Fatal(err)
	}
	if completed.Status != "SUCCESS" {
		t.Fatal(completed)
	}
}

func TestInvokeContractFunction(t *testing.T) {
	contract := soroban.NewContract().
		Wasm(contractWasm).
		Client(&sorobanClient).
		Salt("TestDeployContract").
		SourceAccount(account).
		KeyPair(pair)

	res, err := contract.Invoke().
		Function("hello").
		Symbol("World").
		Send()
	if err != nil {
		t.Fatal(err)
	}
	completed, err := WaitCompletedTransaction(sorobanClient, res.Hash, 10)
	if err != nil {
		t.Fatal(err)
	}
	if completed.Status != "SUCCESS" {
		t.Fatal(completed)
	}

	var resultXdr xdr.TransactionResult
	xdr.SafeUnmarshalBase64(completed.ResultXdr, &resultXdr)
	resultXdr.Result.MustResults()[0].Tr.MustInvokeHostFunctionResult().MustSuccess()

	var transactionMeta xdr.TransactionMeta
	xdr.SafeUnmarshalBase64(completed.ResultMetaXdr, &transactionMeta)

	if *(*transactionMeta.V3.SorobanMeta.ReturnValue.MustVec())[1].Sym != xdr.ScSymbol("World") {
		t.Fatal("Missmatch result")
	}
}
