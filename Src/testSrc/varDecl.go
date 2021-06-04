package main

import (
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/peer"
)

type SimpleAsset struct {
}

func (t *SimpleAsset) Invoke(stub shim.ChaincodeStubInterface) peer.Response {
	var myMap = map[int]int{
		1: 10,
		2: 20,
		3: 30,
	}
	result := 0
	for i, ii := range myMap {
		result = result + i + ii
	}

	return shim.Success([]byte("Result : " + string(result)))
}
