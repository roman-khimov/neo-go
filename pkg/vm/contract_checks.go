package vm

import (
	"encoding/binary"

	"github.com/nspcc-dev/neo-go/pkg/vm/emit"
	"github.com/nspcc-dev/neo-go/pkg/vm/opcode"
)

var (
	verifyInteropID   = emit.InteropNameToID([]byte("Neo.Crypto.ECDsaVerify"))
	multisigInteropID = emit.InteropNameToID([]byte("Neo.Crypto.ECDsaCheckMultiSig"))
)

func getNumOfThingsFromInstr(instr opcode.Opcode, param []byte) (int, bool) {
	var nthings int

	switch instr {
	case opcode.PUSH1, opcode.PUSH2, opcode.PUSH3, opcode.PUSH4,
		opcode.PUSH5, opcode.PUSH6, opcode.PUSH7, opcode.PUSH8,
		opcode.PUSH9, opcode.PUSH10, opcode.PUSH11, opcode.PUSH12,
		opcode.PUSH13, opcode.PUSH14, opcode.PUSH15, opcode.PUSH16:
		nthings = int(instr-opcode.PUSH1) + 1
	case opcode.PUSHBYTES1:
		nthings = int(param[0])
	case opcode.PUSHBYTES2:
		nthings = int(binary.LittleEndian.Uint16(param))
	default:
		return 0, false
	}
	if nthings < 1 || nthings > MaxArraySize {
		return 0, false
	}
	return nthings, true
}

// IsMultiSigContract checks whether the passed script is a multi-signature
// contract.
func IsMultiSigContract(script []byte) bool {
	_, ok := ParseMultiSigContract(script)
	return ok
}

// ParseMultiSigContract returns list of public keys from the verification
// script of the contract.
func ParseMultiSigContract(script []byte) ([][]byte, bool) {
	var nsigs, nkeys int

	ctx := NewContext(script)
	instr, param, err := ctx.Next()
	if err != nil {
		return nil, false
	}
	nsigs, ok := getNumOfThingsFromInstr(instr, param)
	if !ok {
		return nil, false
	}
	var pubs [][]byte
	for {
		instr, param, err = ctx.Next()
		if err != nil {
			return nil, false
		}
		if instr != opcode.PUSHDATA1 {
			break
		}
		if len(param) < 33 {
			return nil, false
		}
		pubs = append(pubs, param)
		nkeys++
		if nkeys > MaxArraySize {
			return nil, false
		}
	}
	if nkeys < nsigs {
		return nil, false
	}
	nkeys2, ok := getNumOfThingsFromInstr(instr, param)
	if !ok {
		return nil, false
	}
	if nkeys2 != nkeys {
		return nil, false
	}
	instr, _, err = ctx.Next()
	if err != nil || instr != opcode.PUSHNULL {
		return nil, false
	}
	instr, param, err = ctx.Next()
	if err != nil || instr != opcode.SYSCALL || binary.LittleEndian.Uint32(param) != multisigInteropID {
		return nil, false
	}
	instr, _, err = ctx.Next()
	if err != nil || instr != opcode.RET || ctx.ip != len(script) {
		return nil, false
	}
	return pubs, true
}

// IsSignatureContract checks whether the passed script is a signature check
// contract.
func IsSignatureContract(script []byte) bool {
	if len(script) != 41 {
		return false
	}

	ctx := NewContext(script)
	instr, param, err := ctx.Next()
	if err != nil || instr != opcode.PUSHDATA1 || len(param) != 33 {
		return false
	}
	instr, _, err = ctx.Next()
	if err != nil || instr != opcode.PUSHNULL {
		return false
	}
	instr, param, err = ctx.Next()
	if err != nil || instr != opcode.SYSCALL || binary.LittleEndian.Uint32(param) != verifyInteropID {
		return false
	}
	return true
}

// IsStandardContract checks whether the passed script is a signature or
// multi-signature contract.
func IsStandardContract(script []byte) bool {
	return IsSignatureContract(script) || IsMultiSigContract(script)
}
