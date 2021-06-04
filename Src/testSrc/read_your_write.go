package main

import (
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/peer"
)

type BadChaincode struct {
}

func (t BadChaincode) Invoke(stub shim.ChaincodeStubInterface) peer.Response {
	key := "key2"
	data := "data"

	stub.PutState(key, []byte(data))

	key2 := "key" + "3"
	res, err := stub.GetState(key2)

	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(res)
}

