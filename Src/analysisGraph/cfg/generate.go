package cfg

import (
	"fmt"
	"strconv"

	"WAH_prototype_go-master/Src/icg"
)

//Generate ...
func Generate(table *icg.SILTable) map[int]CFGBlock {

	//CFG Gen
	generator := cfgGenerator{}
	generator.Init()
	roots := generator.cfgGen(table)

	return roots
}

type cfgGenerator struct {
	labelMap       map[int]int
	blockNumber    int
	funcBlockTable map[int][]CFGBlock
	silcodeTable   *icg.SILTable
}

func (generator *cfgGenerator) Init() {
	generator.labelMap = make(map[int]int)
	generator.funcBlockTable = make(map[int][]CFGBlock)
	generator.blockNumber = 0
}

func (generator *cfgGenerator) cfgGen(table *icg.SILTable) map[int]CFGBlock {
	generator.silcodeTable = table

	funcTble := table.FunctionCodeTable()

	var blockList []CFGBlock

	for k, v := range funcTble {
		blockList = generator.blockGen(v)
		generator.funcBlockTable[k] = blockList
	}

	var roots map[int]CFGBlock = make(map[int]CFGBlock)
	for k := range generator.funcBlockTable {
		roots[k] = generator.linkBlock(k)
	}

	return roots
}
func (generator *cfgGenerator) blockGen(codeList []icg.CodeInfo) []CFGBlock {
	var list []CFGBlock
	var block CFGBlock = new(BasicBlock)
	var befBlock CFGBlock = nil

	for _, sil := range codeList {
		if controlOp, ok := sil.(*icg.ControlOpcode); ok {
			if (icg.Label <= controlOp.Opcode() && controlOp.Opcode() <= icg.Retmv) || (icg.Call == controlOp.Opcode()) {
				if block != nil {
					block.(*BasicBlock)._blockNumber = generator.blockNumber
					block.SetIsPrint(false)
					generator.blockNumber++

					list = append(list, block)
				}

				//branch block build
				opcode := controlOp.Opcode()
				switch opcode {
				case icg.Call:
					block = new(CallBlock)
					block.(*CallBlock)._blockNumber = generator.blockNumber
					generator.blockNumber++

					block.(*CallBlock)._codeList = append(block.(*CallBlock)._codeList, controlOp)
					block.(*CallBlock)._targetBlock = nil
					block.SetIsPrint(false)
					list = append(list, block)
					befBlock = block
					block = nil

					// target := fmt.Sprint(controlOp.Params().Front().Value)
					// targetFK := generator.silcodeTable.StringPool().LookupSymbolNumber(target)

					// if generator.silcodeTable.IsExist(targetFK) {
					// 	funcCodeList := generator.silcodeTable.FunctionCodeTable()[targetFK]
					// 	generator.funcBlockTable[targetFK] = generator.blockGen(funcCodeList)
					// }
				case icg.Ujp:
					block = new(BranchBlock)
					block.(*BranchBlock)._branchType = UnconditionBranch
					block.(*BranchBlock)._blockNumber = generator.blockNumber
					generator.blockNumber++

					block.(*BranchBlock)._codeList = append(block.(*BranchBlock)._codeList, controlOp)
					block.SetIsPrint(false)
					list = append(list, block)
					befBlock = block
					block = nil
				case icg.Fjp:
					block = new(BranchBlock)
					block.(*BranchBlock)._branchType = FalseBranch
					block.(*BranchBlock)._blockNumber = generator.blockNumber
					generator.blockNumber++

					block.(*BranchBlock)._codeList = append(block.(*BranchBlock)._codeList, controlOp)
					block.SetIsPrint(false)
					list = append(list, block)
					befBlock = block
					block = nil
				case icg.Tjp:
					block = new(BranchBlock)
					block.(*BranchBlock)._branchType = TrueBranch
					block.(*BranchBlock)._blockNumber = generator.blockNumber
					generator.blockNumber++

					block.(*BranchBlock)._codeList = append(block.(*BranchBlock)._codeList, controlOp)
					block.SetIsPrint(false)
					list = append(list, block)
					befBlock = block
					block = nil
				case icg.Label:
					target := fmt.Sprint(controlOp.Params().Front().Value)
					labelNum, _ := strconv.Atoi(target)
					if block != nil {
						switch b := block.(type) {
						case *BasicBlock:
							generator.labelMap[labelNum] = b._blockNumber + 1
						case *BranchBlock:
							generator.labelMap[labelNum] = b._blockNumber + 1
						case *CallBlock:
							generator.labelMap[labelNum] = b._blockNumber + 1

						case *ReturnBlock:
							generator.labelMap[labelNum] = b._blockNumber + 1
						}

					} else {
						switch b := befBlock.(type) {
						case *BasicBlock:
							generator.labelMap[labelNum] = b._blockNumber + 1
						case *BranchBlock:
							generator.labelMap[labelNum] = b._blockNumber + 1
						case *CallBlock:
							generator.labelMap[labelNum] = b._blockNumber + 1

						case *ReturnBlock:
							generator.labelMap[labelNum] = b._blockNumber + 1
						}
					}
					block = nil
				}

				if icg.Ret <= controlOp.Opcode() && controlOp.Opcode() <= icg.Retmv {
					block = new(ReturnBlock)
					block.(*ReturnBlock)._codeList = append(block.(*ReturnBlock)._codeList, controlOp)
					block.(*ReturnBlock)._blockNumber = generator.blockNumber
					generator.blockNumber++
					block.SetIsPrint(false)

					list = append(list, block)
					befBlock = block
					block = nil
				}
			} else {
				if block == nil {
					block = new(BasicBlock)
				}
				block.SetIsPrint(false)
				block.(*BasicBlock)._codeList = append(block.(*BasicBlock)._codeList, sil)
			}

		} else {
			if block == nil {
				block = new(BasicBlock)
			}
			block.SetIsPrint(false)
			block.(*BasicBlock)._codeList = append(block.(*BasicBlock)._codeList, sil)
		}
	}

	return list
}

func (generator *cfgGenerator) findBlock(blockNumber int) CFGBlock {
	var res CFGBlock = nil
	for _, blockList := range generator.funcBlockTable {
		for _, block := range blockList {
			switch b := block.(type) {
			case *BasicBlock:
				if b.BlockNumber() == blockNumber {
					res = block
				}
			case *BranchBlock:
				if b.BlockNumber() == blockNumber {
					res = block
				}
			case *CallBlock:
				if b.BlockNumber() == blockNumber {
					res = block
				}

			case *ReturnBlock:
				if b.BlockNumber() == blockNumber {
					res = block
				}
			}

			if res != nil {
				break
			}

		}
	}

	return res
}

func (generator *cfgGenerator) linkBlock(fKey int) CFGBlock {
	blockList := generator.funcBlockTable[fKey]

	for i, block := range blockList {
		if _, ok := block.(*CallBlock); ok {
			// target := fmt.Sprint(callBlock._codeList[0].(*icg.ControlOpcode).Params().Front().Value)
			// targetFK := generator.silcodeTable.StringPool().LookupSymbolNumber(target)
			// if generator.silcodeTable.IsExist(targetFK) {
			// 	generator.linkBlock(targetFK)
			// 	blockList[i].(*CallBlock)._targetBlock = generator.funcBlockTable[targetFK][0]
			// }
			blockList[i].(*CallBlock)._ujpBlock = blockList[i+1]
		} else if branchBlock, ok := block.(*BranchBlock); ok {
			target := fmt.Sprint(branchBlock._codeList[0].(*icg.ControlOpcode).Params().Front().Value)
			targetLabel, _ := strconv.Atoi(target)
			blockList[i].(*BranchBlock)._targetBlock = generator.findBlock(generator.labelMap[targetLabel])

			if branchBlock.BranchType() == FalseBranch || branchBlock.BranchType() == TrueBranch {
				blockList[i].(*BranchBlock)._ujpBlock = blockList[i+1]
			}
		} else if _, ok := block.(*ReturnBlock); ok {

		} else {
			blockList[i].(*BasicBlock)._linkedBlock = blockList[i+1]
		}

	}

	return blockList[0]
}
