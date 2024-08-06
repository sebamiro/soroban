package soroban

import (
	"crypto/sha256"
	"errors"
	"time"

	"github.com/stellar/go/keypair"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
)

type (
	Contract struct {
		wasm     []byte
		wasmHash [32]byte
		salt     [32]byte
		client   *Client
		source   txnbuild.Account
		kp       *keypair.Full
		address  *xdr.ScAddress
	}

	invokeBuilder struct {
		contract *Contract
		build    *invokeBuild
	}

	invokeBuild struct {
		function string
		prams    []xdr.ScVal
	}
)

const (
	ErrorRequiredSource           = "Source Address is required"
	ErrorRequiredWasm             = "Wasm is required"
	ErrorRequiredWasmHash         = "WasmHash is required"
	ErrorRequiredClient           = "Client is required"
	ErrorRequiredKeyPair          = "Key pair is required"
	ErrorRequiredSalt             = "Salt is required"
	ErrorWasmCodeNeedsRestore     = "Wasm code has no ttl, requires a restore"
	ErrorContractNeedsRestore     = "Contract has no ttl, requires a restore"
	ErrorContractDataNeedsRestore = "Contract data has no ttl, requires a restore"
	ErrorInvokeRequiresFunction   = "Function is required"
)

// NewContract returns a Contract builder that can install, deploy and invoke
//
// Example:
//
//	contract := soroban.NewContract().
//		Wasm(contractWasm).
//		Client(&sorobanClient).
//		Salt(salt).
//		SourceAccount(account).
//		KeyPair(pair).
func NewContract() *Contract {
	return &Contract{}
}

// Wasm sets the compiled wasm file of the Contract
func (c *Contract) Wasm(wasm []byte) *Contract {
	c.wasm = wasm
	c.wasmHash = sha256.Sum256(wasm)
	return c
}

// WasmHash sets the compiled wasm hash of the Contract
func (c *Contract) WasmHash(wasmHash [32]byte) *Contract {
	c.wasmHash = wasmHash
	return c
}

// Salt hashes and sets the salt of the contract
// Unique field required to predict the ID of the contract
func (c *Contract) Salt(salt string) *Contract {
	c.salt = sha256.Sum256([]byte(salt))
	return c
}

// Client sets the client to use to connect to the network
func (c *Contract) Client(client *Client) *Contract {
	c.client = client
	return c
}

// SourceAccount sets the account who will call the network
func (c *Contract) SourceAccount(source txnbuild.Account) *Contract {
	c.source = source
	return c
}

// KeyPair sets the key pair to sign transactions
func (c *Contract) KeyPair(kp *keypair.Full) *Contract {
	c.kp = kp
	return c
}

// Address sets the contract address
func (c *Contract) Address(address xdr.ScAddress) *Contract {
	c.address = &address
	return c
}

func (c *Contract) getContractIdPreimage() (xdr.ContractIdPreimage, error) {
	sourceAccountID, err := xdr.AddressToAccountId(c.source.GetAccountID())
	if err != nil {
		return xdr.ContractIdPreimage{}, err
	}

	return xdr.ContractIdPreimage{
		Type: xdr.ContractIdPreimageTypeContractIdPreimageFromAddress,
		FromAddress: &xdr.ContractIdPreimageFromAddress{
			Address: xdr.ScAddress{
				Type:      xdr.ScAddressTypeScAddressTypeAccount,
				AccountId: &sourceAccountID,
			},
			Salt: c.salt,
		},
	}, nil
}

// GetAddress returns the Address as xdr.ScAddress,
// If address is not set its created
//
//	Requires SourceAddress, Client, Salt
func (c *Contract) GetAddress() (*xdr.ScAddress, error) {
	if c.address != nil {
		return c.address, nil
	}
	switch {
	case c.source == nil:
		return nil, errors.New(ErrorRequiredSource)
	case c.client == nil:
		return nil, errors.New(ErrorRequiredClient)
	case len(c.salt) == 0:
		return nil, errors.New(ErrorRequiredSalt)
	}
	contractIdPreimage, err := c.getContractIdPreimage()
	if err != nil {
		return nil, err
	}
	contractId := &xdr.HashIdPreimageContractId{
		NetworkId:          sha256.Sum256([]byte(c.client.PassPhrase)),
		ContractIdPreimage: contractIdPreimage,
	}
	preImage := xdr.HashIdPreimage{
		Type:       xdr.EnvelopeTypeEnvelopeTypeContractId,
		ContractId: contractId,
	}
	xdrPreImageBytes, err := preImage.MarshalBinary()
	if err != nil {
		return nil, err
	}
	contractHash := xdr.Hash(sha256.Sum256(xdrPreImageBytes))
	c.address = &xdr.ScAddress{
		Type:       xdr.ScAddressTypeScAddressTypeContract,
		ContractId: &contractHash,
	}
	return c.address, nil
}

// GetCodeKey returns LedgerKey of ContractCode aka wasm file
//
//	Requires wasm or wasmHash
func (c *Contract) GetCodeKey() (xdr.LedgerKey, error) {
	if len(c.wasmHash) == 0 {
		return xdr.LedgerKey{}, errors.New(ErrorRequiredWasmHash)
	}
	ledgerKey := xdr.LedgerKey{
		Type: xdr.LedgerEntryTypeContractCode,
		ContractCode: &xdr.LedgerKeyContractCode{
			Hash: c.wasmHash,
		},
	}
	return ledgerKey, nil
}

// GetFootprint returns LedgerKey of ContractData aka contract instance
//
//	Requires wasm or wasmHash, SourceAddress, Client, Salt
func (c *Contract) GetFootprint() (xdr.LedgerKey, error) {
	if len(c.wasmHash) == 0 {
		return xdr.LedgerKey{}, errors.New(ErrorRequiredWasmHash)
	}
	contractAddress, err := c.GetAddress()
	if err != nil {
		return xdr.LedgerKey{}, nil
	}
	ledgerKey := xdr.LedgerKey{
		Type: xdr.LedgerEntryTypeContractData,
		ContractData: &xdr.LedgerKeyContractData{
			Contract: *contractAddress,
			Key: xdr.ScVal{
				Type: xdr.ScValTypeScvLedgerKeyContractInstance,
			},
			Durability: xdr.ContractDataDurabilityPersistent,
		},
	}
	return ledgerKey, nil
}

// IsCodeAlive returns if the contract code ttl is > 0 (liveUntilLedger >= current ledger),
// and the ledger entry of the ContractCode.
//
//	Requires wasm or wasmHash, Client
func (c *Contract) IsCodeAlive() (bool, *GetLedgerEntriesResult, error) {
	if c.client == nil {
		return false, nil, errors.New(ErrorRequiredClient)
	}
	ledgerKey, err := c.GetCodeKey()
	if err != nil {
		return false, nil, err
	}
	base64, err := ledgerKey.MarshalBinaryBase64()
	if err != nil {
		return false, nil, err
	}
	res, err := c.client.GetLedgerEntries(base64)
	if err != nil {
		return false, nil, err
	}
	return res.Entries[0].LiveUntilLedgerSeq >= res.LatestLedger, res, nil
}

// IsInstanceAlive returns if the contract data ttl is > 0 (liveUntilLedger >= current ledger),
// and the ledger entry of the ContractData.
//
//	Requires wasm or wasmHash, SourceAddress, Client, Salt
func (c *Contract) IsInstanceAlive() (bool, *GetLedgerEntriesResult, error) {
	ledgerKey, err := c.GetFootprint()
	if err != nil {
		return false, nil, err
	}
	base64, err := ledgerKey.MarshalBinaryBase64()
	if err != nil {
		return false, nil, err
	}
	res, err := c.client.GetLedgerEntries(base64)
	if err != nil {
		return false, nil, err
	}
	if len(res.Entries) == 0 {
		return false, res, nil
	}
	return res.Entries[0].LiveUntilLedgerSeq >= res.LatestLedger, res, nil
}

// IsAlive checks if contract code and instance are alive
//
//	Requires wasm or wasmHash, SourceAddress, Client, Salt
func (c *Contract) IsAlive() (bool, error) {
	code, _, err := c.IsCodeAlive()
	if err != nil {
		return false, err
	}
	instance, _, err := c.IsInstanceAlive()
	if err != nil {
		return false, err
	}
	return code && instance, nil
}

// Install sends the transaction to install the compiled contract wasm file
// The result status can be PENDING, DUPLICATE, TRY_AGAIN_LATER, ERROR
// It will NOT check if it was accepted, it will need to be check
// using RPC call to getTransaction with the transaction hash
//
//	Requires wasm, client, sourceAccount, keyPair
//
//	Example:
//	 res, err := soroban.NewContract().
//		Wasm(contractWasm).
//		Client(&sorobanClient).
//		Salt(salt).
//		SourceAccount(account).
//		KeyPair(pair).
//		Install()
func (c *Contract) Install() (*SendTransactionResult, error) {
	switch {
	case c.client == nil:
		return nil, errors.New(ErrorRequiredClient)
	case c.source == nil:
		return nil, errors.New(ErrorRequiredSource)
	case c.kp == nil:
		return nil, errors.New(ErrorRequiredKeyPair)
	}
	installOp := txnbuild.InvokeHostFunction{
		HostFunction: xdr.HostFunction{
			Type: xdr.HostFunctionTypeHostFunctionTypeUploadContractWasm,
			Wasm: &c.wasm,
		},
		SourceAccount: c.source.GetAccountID(),
	}
	return c.simulateSubmitHostFunction(installOp)
}

// Deploy sends the transaction to create a new instance of the compiled contract wasm file.
// It will return an error if the wasm code is not installed or has no time to live left.
// The result status can be PENDING, DUPLICATE, TRY_AGAIN_LATER, ERROR.
// It will NOT check if it was accepted, it will need to be check
// using RPC call to getTransaction with the transaction hash
//
//	Requires wasm, client, sourceAccount, keyPair
//
//	Example:
//	 res, err := soroban.NewContract().
//		Wasm(contractWasm).
//		Client(&sorobanClient).
//		Salt(salt).
//		SourceAccount(account).
//		KeyPair(pair).
//		Deploy()
func (c *Contract) Deploy() (*SendTransactionResult, error) {
	switch {
	case c.client == nil:
		return nil, errors.New(ErrorRequiredClient)
	case c.source == nil:
		return nil, errors.New(ErrorRequiredSource)
	case c.kp == nil:
		return nil, errors.New(ErrorRequiredKeyPair)
	}
	isCodeAlive, _, err := c.IsCodeAlive()
	if err != nil {
		return nil, err
	}
	if !isCodeAlive {
		return nil, errors.New(ErrorWasmCodeNeedsRestore)
	}

	contractIdPreimage, err := c.getContractIdPreimage()
	if err != nil {
		return nil, err
	}
	createContract := &xdr.CreateContractArgs{
		ContractIdPreimage: contractIdPreimage,
		Executable: xdr.ContractExecutable{
			Type:     xdr.ContractExecutableTypeContractExecutableWasm,
			WasmHash: (*xdr.Hash)(&c.wasmHash),
		},
	}
	createOp := txnbuild.InvokeHostFunction{
		HostFunction: xdr.HostFunction{
			Type:           xdr.HostFunctionTypeHostFunctionTypeCreateContract,
			CreateContract: createContract,
		},
		SourceAccount: c.source.GetAccountID(),
	}
	return c.simulateSubmitHostFunction(createOp)
}

// Invoke inits the building of an invoketion transaction of a function.
// It will return a inokeBuilder where function name, and parameter can be added.
//
//	Example:
//	 res, err := contract.
//		Invoke().
//		Function("hello").
//		Symbol("world").
//		Send()
func (c *Contract) Invoke() *invokeBuilder {
	return &invokeBuilder{
		contract: c,
		build: &invokeBuild{
			prams: make([]xdr.ScVal, 0),
		},
	}
}

// Function sets function name to be invoked
func (c *invokeBuilder) Function(function string) *invokeBuilder {
	c.build.function = function
	return c
}

// Params appends a list of xdr.ScVal to the params
func (c *invokeBuilder) Params(params ...xdr.ScVal) *invokeBuilder {
	c.build.prams = append(c.build.prams, params...)
	return c
}

// Bool appends a bool xdr.ScVal to the params
func (c *invokeBuilder) Bool(b bool) *invokeBuilder {
	c.build.prams = append(c.build.prams, xdr.ScVal{Type: xdr.ScValTypeScvBool, B: &b})
	return c
}

// Int32 appends a int32 xdr.ScVal to the params
func (c *invokeBuilder) Int32(i int32) *invokeBuilder {
	c.build.prams = append(c.build.prams, xdr.ScVal{Type: xdr.ScValTypeScvI32, I32: (*xdr.Int32)(&i)})
	return c
}

// Int64 appends a int64 xdr.ScVal to the params
func (c *invokeBuilder) Int64(i int64) *invokeBuilder {
	c.build.prams = append(c.build.prams, xdr.ScVal{Type: xdr.ScValTypeScvI64, I64: (*xdr.Int64)(&i)})
	return c
}

// Uint32 appends a uint64 xdr.ScVal to the params
func (c *invokeBuilder) Uint32(i uint32) *invokeBuilder {
	c.build.prams = append(c.build.prams, xdr.ScVal{Type: xdr.ScValTypeScvU32, U32: (*xdr.Uint32)(&i)})
	return c
}

// Uint64 appends a uint64 xdr.ScVal to the params
func (c *invokeBuilder) Uint64(i uint64) *invokeBuilder {
	c.build.prams = append(c.build.prams, xdr.ScVal{Type: xdr.ScValTypeScvU64, U64: (*xdr.Uint64)(&i)})
	return c
}

// String appends a string xdr.ScVal to the params
func (c *invokeBuilder) String(s string) *invokeBuilder {
	c.build.prams = append(c.build.prams, xdr.ScVal{Type: xdr.ScValTypeScvString, Str: (*xdr.ScString)(&s)})
	return c
}

// Symbol appends a symbol xdr.ScVal to the params
func (c *invokeBuilder) Symbol(s string) *invokeBuilder {
	c.build.prams = append(c.build.prams, xdr.ScVal{Type: xdr.ScValTypeScvSymbol, Sym: (*xdr.ScSymbol)(&s)})
	return c
}

// Send sends the transaction to invoke the contract function with the parameters set.
// It will return an error if the wasm code is not installed or has no time to live left.
// It will return an error if the contract instance has no time to live left.
// The result status can be PENDING, DUPLICATE, TRY_AGAIN_LATER, ERROR
// It will NOT check if it was accepted, it will need to be check
// using RPC call to getTransaction with the transaction hash
//
//	Requires wasm, client, sourceAccount, keyPair, salt, function
func (c *invokeBuilder) Send() (*SendTransactionResult, error) {
	if c.build.function == "" {
		return nil, errors.New(ErrorInvokeRequiresFunction)
	}
	isAlive, err := c.contract.IsAlive()
	if err != nil {
		return nil, err
	}
	if !isAlive {
		return nil, errors.New(ErrorContractNeedsRestore)
	}
	return c.contract.invoke(c.build, false)
}

// RestoreAndSend if the contract has no ttl left, it will retore it before sending the transaction.
// The result status can be PENDING, DUPLICATE, TRY_AGAIN_LATER, ERROR
// It will NOT check if it was accepted, it will need to be check
// using RPC call to getTransaction with the transaction hash
//
//	Requires wasm, client, sourceAccount, keyPair, salt, function
func (c *invokeBuilder) RestoreAndSend() (*SendTransactionResult, error) {
	if c.build.function == "" {
		return nil, errors.New(ErrorInvokeRequiresFunction)
	}
	isAlive, err := c.contract.IsAlive()
	if err != nil {
		return nil, err
	}
	if !isAlive {
		res, err := c.contract.Restore()
		if err != nil {
			return nil, err
		}
		c.contract.client.waitCompletedTransaction(res.Hash)
	}
	return c.contract.invoke(c.build, true)
}

func (c *Contract) invoke(build *invokeBuild, restore bool) (*SendTransactionResult, error) {
	contractAddress, err := c.GetAddress()
	if err != nil {
		return nil, err
	}
	invokeHostFunctionOp := txnbuild.InvokeHostFunction{
		HostFunction: xdr.HostFunction{
			Type: xdr.HostFunctionTypeHostFunctionTypeInvokeContract,
			InvokeContract: &xdr.InvokeContractArgs{
				ContractAddress: *contractAddress,
				FunctionName:    xdr.ScSymbol(build.function),
				Args:            (xdr.ScVec)(build.prams),
			},
		},
		SourceAccount: c.source.GetAccountID(),
	}
	transaction := NewTransctionBuilder().
		Client(c.client).
		SourceAccount(c.source).
		Signer(c.kp).
		Operation(&invokeHostFunctionOp).
		TimeBounds(txnbuild.NewTimeout(30))
	res, err := transaction.Simulate()
	if err != nil {
		return nil, err
	}
	if res.RestorePreamble.MinResourceFee != 0 {
		if !restore {
			return nil, errors.New(ErrorContractDataNeedsRestore)
		}
		var transactionData xdr.SorobanTransactionData
		err := xdr.SafeUnmarshalBase64(res.TransactionData, &transactionData)
		if err != nil {
			return nil, err
		}
		t := NewTransctionBuilder().
			Client(c.client).
			SourceAccount(c.source).
			Signer(c.kp).
			Operation(&txnbuild.RestoreFootprint{SourceAccount: c.source.GetAccountID()}).
			TimeBounds(txnbuild.NewTimeout(30)).
			SorobanData(transactionData).
			BaseFee(res.RestorePreamble.MinResourceFee + txnbuild.MinBaseFee)
		res, err := t.Send()
		if err != nil {
			return nil, err
		}
		c.client.waitCompletedTransaction(res.Hash)
	}
	return transaction.Send()
}

func (c *Contract) simulateSubmitHostFunction(op txnbuild.InvokeHostFunction) (*SendTransactionResult, error) {
	transaction := NewTransctionBuilder().
		Client(c.client).
		SourceAccount(c.source).
		Signer(c.kp).
		Operation(&op).
		TimeBounds(txnbuild.NewTimeout(30))
	_, err := transaction.Simulate()
	if err != nil {
		return nil, err
	}
	return transaction.Send()
}

// Restore restores the contract wasm code and instace if neededd
// Docs: https://developers.stellar.org/docs/learn/encyclopedia/storage/state-archival
//
//	Requires wasm, client, sourceAccount, keyPair, salt, function
func (c *Contract) Restore() (*SendTransactionResult, error) {
	var readWrite []xdr.LedgerKey
	codeKey, err := c.GetCodeKey()
	if err != nil {
		return nil, err
	}
	readWrite = append(readWrite, codeKey)
	instanceKey, err := c.GetFootprint()
	if err != nil {
		return nil, err
	}
	readWrite = append(readWrite, instanceKey)
	transaction := NewTransctionBuilder().
		Client(c.client).
		SourceAccount(c.source).
		Signer(c.kp).
		Operation(&txnbuild.RestoreFootprint{SourceAccount: c.source.GetAccountID()}).
		TimeBounds(txnbuild.NewTimeout(30)).
		SorobanData(xdr.SorobanTransactionData{
			Resources: xdr.SorobanResources{
				Footprint: xdr.LedgerFootprint{
					ReadWrite: readWrite,
				},
			},
		})
	_, err = transaction.Simulate()
	if err != nil {
		return nil, err
	}
	return transaction.Send()
}

func (c *Client) waitCompletedTransaction(hash string) (*GetTransactionResult, error) {
	for i := 0; i < 5; i++ {
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
