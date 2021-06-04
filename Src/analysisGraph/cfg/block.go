package cfg

import (
	"strconv"

	"WAH_prototype_go-master/Src/icg"
)

type BranchType int

const (
	TrueBranch        = 0
	FalseBranch       = 1
	UnconditionBranch = 2
	CallBranch        = 3
)

type CFGBlock interface {
	String() string
	SetIsPrint(isPrint bool)
	BlockNumber() int
	IsPrint() bool
}

type BasicBlock struct {
	_blockNumber int
	_codeList    []icg.CodeInfo
	_linkedBlock CFGBlock
	_isPrint     bool
}

func (block *BasicBlock) BlockNumber() int {
	return block._blockNumber
}
func (block *BasicBlock) SetIsPrint(isPrint bool) {
	block._isPrint = isPrint
}
func (block *BasicBlock) IsPrint() bool {
	return block._isPrint
}

func (block *BasicBlock) LinkedBlock() CFGBlock {
	return block._linkedBlock
}

func (block *BasicBlock) CodeList() []icg.CodeInfo {
	return block._codeList
}

func (block *BasicBlock) String() string {
	var buffer string
	for _, codeInfo := range block._codeList {
		buffer += strconv.Itoa(codeInfo.GetLine()) + " : " + codeInfo.String() + "\n"
	}

	return buffer
}

type BranchBlock struct {
	_branchType  BranchType
	_targetBlock CFGBlock
	_ujpBlock    CFGBlock
	BasicBlock
}

func (block *BranchBlock) BranchType() BranchType {
	return block._branchType
}
func (block *BranchBlock) TargetBlock() CFGBlock {
	return block._targetBlock
}
func (block *BranchBlock) UjpBlock() CFGBlock {
	return block._ujpBlock
}

func (block *BranchBlock) String() string {
	var buffer string
	for _, codeInfo := range block._codeList {
		buffer += strconv.Itoa(codeInfo.GetLine()) + " : " + codeInfo.String() + "\n"
	}

	return buffer
}

type CallBlock struct {
	BranchBlock
}

type ReturnBlock struct {
	BasicBlock
}
