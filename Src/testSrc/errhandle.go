package main

import (
	"fmt"
	//"fmt"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/peer"
	//"log"
)

type BadChaincode struct {
}

func (t BadChaincode) Invoke(stub shim.ChaincodeStubInterface) peer.Response {
	key := "keystring"
	ret, err := stub.GetState(key)

	if err != nil {
		fmt.Println(err.Error())
	}
	if ret != nil {
		return shim.Success(ret)
	}

	return shim.Error("error : can't get state")
}


