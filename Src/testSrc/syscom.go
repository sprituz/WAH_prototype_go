package main

import (
"os/exec"
"github.com/hyperledger/fabric-chaincode-go/shim"
pb "github.com/hyperledger/fabric-protos-go/peer"
)

type SimpleChaincode struct {}

func (t *SimpleChaincode) example(stub shim.ChaincodeStubInterface, key string) pb.Response {
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
