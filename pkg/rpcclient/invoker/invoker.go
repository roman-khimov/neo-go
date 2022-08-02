package invoker

import (
	"fmt"

	"github.com/nspcc-dev/neo-go/pkg/core/transaction"
	"github.com/nspcc-dev/neo-go/pkg/neorpc/result"
	"github.com/nspcc-dev/neo-go/pkg/smartcontract"
	"github.com/nspcc-dev/neo-go/pkg/util"
)

// RPCInvoke is a set of RPC methods needed to execute things at the current
// blockchain height.
type RPCInvoke interface {
	InvokeContractVerify(contract util.Uint160, params []smartcontract.Parameter, signers []transaction.Signer, witnesses ...transaction.Witness) (*result.Invoke, error)
	InvokeFunction(contract util.Uint160, operation string, params []smartcontract.Parameter, signers []transaction.Signer) (*result.Invoke, error)
	InvokeScript(script []byte, signers []transaction.Signer) (*result.Invoke, error)
}

// RPCInvokeHistoric is a set of RPC methods needed to execute things at some
// fixed point in blockchain's life.
type RPCInvokeHistoric interface {
	InvokeContractVerifyAtBlock(blockHash util.Uint256, contract util.Uint160, params []smartcontract.Parameter, signers []transaction.Signer, witnesses ...transaction.Witness) (*result.Invoke, error)
	InvokeContractVerifyAtHeight(height uint32, contract util.Uint160, params []smartcontract.Parameter, signers []transaction.Signer, witnesses ...transaction.Witness) (*result.Invoke, error)
	InvokeContractVerifyWithState(stateroot util.Uint256, contract util.Uint160, params []smartcontract.Parameter, signers []transaction.Signer, witnesses ...transaction.Witness) (*result.Invoke, error)
	InvokeFunctionAtBlock(blockHash util.Uint256, contract util.Uint160, operation string, params []smartcontract.Parameter, signers []transaction.Signer) (*result.Invoke, error)
	InvokeFunctionAtHeight(height uint32, contract util.Uint160, operation string, params []smartcontract.Parameter, signers []transaction.Signer) (*result.Invoke, error)
	InvokeFunctionWithState(stateroot util.Uint256, contract util.Uint160, operation string, params []smartcontract.Parameter, signers []transaction.Signer) (*result.Invoke, error)
	InvokeScriptAtBlock(blockHash util.Uint256, script []byte, signers []transaction.Signer) (*result.Invoke, error)
	InvokeScriptAtHeight(height uint32, script []byte, signers []transaction.Signer) (*result.Invoke, error)
	InvokeScriptWithState(stateroot util.Uint256, script []byte, signers []transaction.Signer) (*result.Invoke, error)
}

// Invoker allows to test-execute things using RPC client. Its API simplifies
// reusing the same signers list for a series of invocations and at the
// same time uses regular Go types for call parameters. It doesn't do anything with
// the result of invocation, that's left for upper (contract) layer to deal with.
// Invoker does not produce any transactions and does not change the state of the
// chain.
type Invoker struct {
	client  RPCInvoke
	signers []transaction.Signer
}

type historicConverter struct {
	client RPCInvokeHistoric
	block  *util.Uint256
	height *uint32
	root   *util.Uint256
}

// New creates an Invoker to test-execute things at the current blockchain height.
func New(client RPCInvoke, signers []transaction.Signer) *Invoker {
	return &Invoker{client, signers}
}

// NewHistoricAtBlock creates an Invoker to test-execute things at some given block.
func NewHistoricAtBlock(block util.Uint256, client RPCInvokeHistoric, signers []transaction.Signer) *Invoker {
	return New(&historicConverter{
		client: client,
		block:  &block,
	}, signers)
}

// NewHistoricAtHeight creates an Invoker to test-execute things at some given height.
func NewHistoricAtHeight(height uint32, client RPCInvokeHistoric, signers []transaction.Signer) *Invoker {
	return New(&historicConverter{
		client: client,
		height: &height,
	}, signers)
}

// NewHistoricWithState creates an Invoker to test-execute things with some given state.
func NewHistoricWithState(root util.Uint256, client RPCInvokeHistoric, signers []transaction.Signer) *Invoker {
	return New(&historicConverter{
		client: client,
		root:   &root,
	}, signers)
}

func (h *historicConverter) InvokeScript(script []byte, signers []transaction.Signer) (*result.Invoke, error) {
	if h.block != nil {
		return h.client.InvokeScriptAtBlock(*h.block, script, signers)
	}
	if h.height != nil {
		return h.client.InvokeScriptAtHeight(*h.height, script, signers)
	}
	if h.root != nil {
		return h.client.InvokeScriptWithState(*h.root, script, signers)
	}
	panic("uninitialized historicConverter")
}

func (h *historicConverter) InvokeFunction(contract util.Uint160, operation string, params []smartcontract.Parameter, signers []transaction.Signer) (*result.Invoke, error) {
	if h.block != nil {
		return h.client.InvokeFunctionAtBlock(*h.block, contract, operation, params, signers)
	}
	if h.height != nil {
		return h.client.InvokeFunctionAtHeight(*h.height, contract, operation, params, signers)
	}
	if h.root != nil {
		return h.client.InvokeFunctionWithState(*h.root, contract, operation, params, signers)
	}
	panic("uninitialized historicConverter")
}

func (h *historicConverter) InvokeContractVerify(contract util.Uint160, params []smartcontract.Parameter, signers []transaction.Signer, witnesses ...transaction.Witness) (*result.Invoke, error) {
	if h.block != nil {
		return h.client.InvokeContractVerifyAtBlock(*h.block, contract, params, signers, witnesses...)
	}
	if h.height != nil {
		return h.client.InvokeContractVerifyAtHeight(*h.height, contract, params, signers, witnesses...)
	}
	if h.root != nil {
		return h.client.InvokeContractVerifyWithState(*h.root, contract, params, signers, witnesses...)
	}
	panic("uninitialized historicConverter")
}

// Call invokes a method of the contract with the given parameters (and
// Invoker-specific list of signers) and returns the result as is.
func (v *Invoker) Call(contract util.Uint160, operation string, params ...interface{}) (*result.Invoke, error) {
	ps, err := smartcontract.NewParametersFromValues(params...)
	if err != nil {
		return nil, err
	}
	return v.client.InvokeFunction(contract, operation, ps, v.signers)
}

// CallAndExpandIterator creates a script containing a call of the specified method
// of a contract with given parameters (similar to how Call operates). But then this
// script contains additional code that expects that the result of the first call is
// an iterator. This iterator is traversed extracting values from it and adding them
// into an array until maxItems is reached or iterator has no more elements. The
// result of the whole script is an array containing up to maxResultItems elements
// from the iterator returned from the contract's method call. This script is executed
// using regular JSON-API (according to the way Iterator is set up).
func (v *Invoker) CallAndExpandIterator(contract util.Uint160, method string, maxItems int, params ...interface{}) (*result.Invoke, error) {
	bytes, err := smartcontract.CreateCallAndUnwrapIteratorScript(contract, method, maxItems, params...)
	if err != nil {
		return nil, fmt.Errorf("iterator unwrapper script: %w", err)
	}
	return v.Run(bytes)
}

// Verify invokes contract's verify method in the verification context with
// Invoker-specific signers and given witnesses and parameters.
func (v *Invoker) Verify(contract util.Uint160, witnesses []transaction.Witness, params ...interface{}) (*result.Invoke, error) {
	ps, err := smartcontract.NewParametersFromValues(params...)
	if err != nil {
		return nil, err
	}
	return v.client.InvokeContractVerify(contract, ps, v.signers, witnesses...)
}

// Run executes given bytecode with Invoker-specific list of signers.
func (v *Invoker) Run(script []byte) (*result.Invoke, error) {
	return v.client.InvokeScript(script, v.signers)
}
