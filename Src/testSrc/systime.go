package main


import (
"encoding/json"
"time"
"github.com/hyperledger/fabric-chaincode-go/shim"
"github.com/hyperledger/fabric-protos-go/peer"
)

type BadChaincode struct {
}

func (t *BadChaincode) Init(stub shim.ChaincodeStubInterface) peer.Response {
		return shim.Success([]byte("success"))

	}


func (t *BadChaincode) Invoke(stub shim.ChaincodeStubInterface) peer.Response {
 tByte,err:= json.Marshal(time.Now())
   err = stub.PutState("key", tByte)
if err != nil {
 return shim.Error(err.Error())
}
return shim.Success([]byte("success"))
}
