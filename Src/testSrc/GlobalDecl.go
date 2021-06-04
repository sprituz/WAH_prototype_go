package main

import (
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/peer"
)

type BadChaincode struct {
	field string
}
var global string

func (t BadChaincode) Invoke(stub shim.ChaincodeStubInterface) peer.Response {
	fn, args := stub.GetFunctionAndParameters()

	if len(args) > 1 {
		global = args[0]
		t.field = args[1]
	}

	if fn == "Set" {
		key := t.field
		val := global
		stub.PutState(key, []byte(val))
		return shim.Success([]byte("Success"))
	}

	return shim.Error("Setting Error")
}

