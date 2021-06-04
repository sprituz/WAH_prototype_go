package main

import (
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/peer"
)

type BadChaincode struct {
}

func (t BadChaincode) Invoke(stub shim.ChaincodeStubInterface) peer.Response {
	key := "key"
	data := "data"
	key2 := key

	stub.PutState(key2, []byte(data))

	res, _ := stub.GetState(key)

	return shim.Success(res)
}
