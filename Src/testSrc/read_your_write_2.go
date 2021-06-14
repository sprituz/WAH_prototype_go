package main

import (
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/peer"
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
