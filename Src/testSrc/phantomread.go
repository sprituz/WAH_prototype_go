package main

import (
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/peer"
)

type BadChaincode struct {
}

func (t BadChaincode) Invoke(stub shim.ChaincodeStubInterface) peer.Response {
	iterator, _ := stub.GetHistoryForKey("key")
	data, _ := iterator.Next()
	err := stub.PutState("key", data.Value)

	if err != nil {
		return shim.Error("could not write new data")
	}

	return shim.Success([]byte("stored"))
}
