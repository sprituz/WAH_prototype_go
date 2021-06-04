package main

import (
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/peer"
)

type BadChaincode struct {
	field string
}

func (t BadChaincode) Invoke(stub shim.ChaincodeStubInterface) peer.Response {
	fn, args := stub.GetFunctionAndParameters()
	t.field = args[0]
	if fn == "Set" {
		stub.PutState("key", []byte(t.field))
		return shim.Success([]byte("Success"))
	}

	return shim.Success([]byte("default"))
}
