package main

import (
"os/exec"
"github.com/hyperledger/fabric-chaincode-go/shim"
"github.com/hyperledger/fabric-protos-go/peer"
)

type BadChaincode struct {}

func (t *BadChaincode) example(stub shim.ChaincodeStubInterface, key string) peer.Response {
	out, err:= exec.Command("date").Output()
	if err != nil {
	shim.Error("error")
	}
	err = stub.PutState(key, out)
	if err != nil {
		shim.Error("error")
	}
	return shim.Success(nil)
}
