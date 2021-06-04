package main

import (
"github.com/hyperledger/fabric/core/chaincode/shim"
"github.com/hyperledger/fabric/protos/peer"
)

type SimpleAsset struct {
}

func writeToLedger(stub shim.ChaincodeStubInterface, key string, data string) peer.Response {
	err := stub.PutState(key, []byte(data))
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success([]byte(key))
}

func (t *SimpleAsset) Invoke(stub shim.ChaincodeStubInterface) peer.Response {

	go writeToLedger(stub,"key1", "data1")
	go writeToLedger(stub,"key1", "data2")

	return shim.Success([]byte("key1"))
}


