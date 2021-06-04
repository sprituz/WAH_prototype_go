package main

import (
	"log"

	//"fmt"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/peer"
	//"log"
)

type BadChaincode struct {
}

func (t BadChaincode) Invoke(stub shim.ChaincodeStubInterface) peer.Response {
	key := "keystring"
	ret, err := stub.GetState(key)

	if err != nil {
		log.Fatal(err)
	}

	return shim.Success(ret)
}
