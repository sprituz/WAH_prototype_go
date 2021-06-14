package main
import (
"math/rand"
"strconv"

"github.com/hyperledger/fabric-chaincode-go/shim"
"github.com/hyperledger/fabric-protos-go/peer"
)

type BadChaincode struct {
}

func (t *BadChaincode) Invoke(stub shim.ChaincodeStubInterface) peer.Response {
	args := stub.GetStringArgs()
	data := "data"

	var ran int

	if args[0] == "random" {
		rand.Seed(1)
	}
	ran = rand.Intn(10)

	stub.PutState(strconv.Itoa(ran), []byte(data))
	return shim.Success([]byte("Success"))
}
