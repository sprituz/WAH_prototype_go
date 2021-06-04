package wah

import (
	"fmt"
	"strings"

	"WAH_prototype_go-master/Src/analysisGraph/cfg"
	"WAH_prototype_go-master/Src/analysisGraph/vfg"
	"WAH_prototype_go-master/Src/icg"
)

type UEAnalyzer struct {
	analysisFile  string
	chain         vfg.DUChain
	codeList      []icg.CodeInfo
	errTable      map[string]int //funcName, error return num
	analysisCount int
	detectedLint  []int
	handledPoint map[string]int
}

func (analyzer *UEAnalyzer) Init(analysisFile string, chain vfg.DUChain, codeList []icg.CodeInfo) {
	analyzer.analysisFile = analysisFile
	analyzer.chain = chain
	analyzer.codeList = codeList
	analyzer.analysisCount = 0
	analyzer.handledPoint = make(map[string]int)

}
func (analyzer *UEAnalyzer) SetErrTable(errTable map[string]int) {
	analyzer.errTable = errTable
}

func (analyzer *UEAnalyzer) IsErrorVar(variable icg.CodeInfo, offset string) bool {
	reverseCodeList := sliceReverse(analyzer.codeList)
	defLine := 0
	res := false
	if variable.Opcode() == icg.Lod {
		lod := variable.(*icg.StackOpcode)

		defLine, _ = analyzer.chain.LookUpDefOfUse(offset, lod.GetLine())
	} else if variable.Opcode() == icg.Ldi {
		ldi := variable.(*icg.StackOpcode)
		defLine, _ = analyzer.chain.LookUpDefOfUse(offset, ldi.GetLine())
	} else if variable.Opcode() == icg.Str {
		defLine = variable.GetLine()
	} else {
		return res
	}

	defCodeIndex := findSILIndex(reverseCodeList, defLine)

	analysisRange := reverseCodeList[defCodeIndex:]
	sp := -analysisRange[0].GetPopParameterNum() //pop이 소모자, push가 판매

	if isDebug {
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Printf("\t Error Init Variable Analysis %s \n", variable)
		fmt.Println("--------------------------------------------------------------------------------")
	}
	for i, code := range analysisRange {
		if isDebug {
			fmt.Printf("code : %s \n\tpush : %d \tpop : %d\n", code.String(), code.GetPushParameterNum(), code.GetPopParameterNum())

		}
		if i == 0 {
			if isDebug {
				fmt.Printf("\tSatisfaction value for reverse analysis (if 0, analysis ends) : %d\n\n", sp)
			}
			continue
		}

		isFunction := false
		if code.Opcode() != icg.Call {
			sp = sp + code.GetPushParameterNum() - code.GetPopParameterNum()
		} else {
			sp = sp + code.GetPushParameterNum()
			isFunction = true
		}

		if isDebug {
			fmt.Printf("\tSatisfaction value for reverse analysis (if 0, analysis ends) : %d\n\n", sp)
		}

		if code.Opcode() == icg.Call && sp >= 0 {
			call := code.(*icg.ControlOpcode)
			funcName := fmt.Sprint(call.Params().Front().Value)

			if err, ok := analyzer.errTable[funcName]; ok {

				errLocation := call.GetPushParameterNum() - (sp + 1)
				if err == errLocation {
					res = true
					if isDebug {
						fmt.Printf("Inverse analysis completed!\n")
						fmt.Printf("\t error code : %s\n \t target code : %s\n\n", analysisRange[0].String(), code.String())
						fmt.Println("--------------------------------------------------------------------------------\n")
					}
				}

			}

			break
		}

		if isFunction {
			sp = sp - code.GetPopParameterNum()
			isFunction = false
		}

		if sp == 0 {
			res = false
			break
		}

	}

	return res
}
func (analyzer *UEAnalyzer) isDetected(line int) bool {
	res := false
	for _, dline := range analyzer.detectedLint {
		if dline == line {
			res = true
		}
	}
	return res
}
func (analyzer *UEAnalyzer) printAlarm(linenum int) {
	var ccw CCW = UNHANDLED_ERROR
	if isFirstDetect {
		fmt.Printf("chaincode weakness detected:\n")
		isFirstDetect = false
	}

	if !analyzer.isDetected(linenum) {
		analyzer.analysisCount++
		analyzer.detectedLint = append(analyzer.detectedLint,linenum)
		fmt.Printf("\t CCW-%03d : %s\n", ccw, ccw.String())
		fmt.Printf("\t %s : %d\n\n", analyzer.analysisFile, linenum)
	}
}
func (analyzer *UEAnalyzer) IsEmptyHandling(block cfg.CFGBlock, endblock cfg.CFGBlock) bool{
	res := true
	if block.BlockNumber() != endblock.BlockNumber() {
		switch b := block.(type) {
		case *cfg.BasicBlock:
			res = analyzer.IsEmptyHandling(b.LinkedBlock(), endblock)
		case *cfg.CallBlock:
			opcode := b.CodeList()[0].(*icg.ControlOpcode)
			funcName := fmt.Sprint(opcode.Params().Front().Value)
			if strings.Contains(funcName, "shim.Error") || strings.Contains(funcName, "log.Fatal") {
				res = false
				return res
			} else {
				res = analyzer.IsEmptyHandling(b.UjpBlock(), endblock)
			}

		case *cfg.BranchBlock:
			ujpRes := true
			if b.UjpBlock() != nil {
				ujpRes = analyzer.IsEmptyHandling(b.UjpBlock(), endblock)
			}

			targetRes := true
			if b.TargetBlock() != nil {
				targetRes = analyzer.IsEmptyHandling(b.UjpBlock(), endblock)
			}
			if !ujpRes && !targetRes {
				res = false
			}
		case *cfg.ReturnBlock:
			res = false
		}
	}
	return res
}
func (analyzer *UEAnalyzer) UEAnalysis(block cfg.CFGBlock) int {
	switch b := block.(type) {
	case *cfg.BasicBlock:
		var addressOffset []string
		for _, sil := range b.CodeList() {

			switch sil.Opcode() {
			case icg.Str:
				arg, _ := sil.(*icg.StackOpcode)
				offsetStr := fmt.Sprint(arg.Params().Front().Next().Value)

				if analyzer.IsErrorVar(sil, offsetStr) {
					// err handling이 있는지 DU체인 분석
					if useList, ok := analyzer.chain.LookUpUseOfDef(offsetStr, sil.GetLine()); ok {
						// error 변수인데 사용된 곳이 없다면 핸들링 되지 않은 것
						if len(useList) == 0 {
							analyzer.printAlarm(sil.GetSourceLine())
						} else {
							// error handling 하는 if문에서 사용되었는지
							for _, use := range useList {
								indx := findSILIndex(analyzer.codeList, use)
								useSIL := analyzer.codeList[indx]
								errSIL := useSIL.(*icg.StackOpcode)
								offset := fmt.Sprint(errSIL.Params().Front().Next().Value)

								if useSIL.ParentStmt() != icg.IfStmt {
									if p,ok := analyzer.handledPoint[offset]; ok {
										if p > useSIL.GetLine() {
											analyzer.printAlarm(useSIL.GetSourceLine())
										}
									}

								}else { //if문일 때 empty handling이 아닌지
									if bBlock , ok := b.LinkedBlock().(*cfg.BranchBlock); ok {
										// empty handling을 잡아내기 위해 if문 조건을 만족하는 블록만 검사
										if bBlock.UjpBlock().BlockNumber() == bBlock.TargetBlock().BlockNumber() {
											analyzer.printAlarm(useSIL.GetSourceLine())
										}else if bBlock.BranchType() ==cfg.FalseBranch  &&  bBlock.UjpBlock() != nil{
											if analyzer.IsEmptyHandling(bBlock.UjpBlock(),bBlock.TargetBlock()) {
												analyzer.printAlarm(useSIL.GetSourceLine())
											}else {
												analyzer.handledPoint[offset] = useSIL.GetLine()
											}
										}else if bBlock.BranchType() == cfg.TrueBranch && bBlock.TargetBlock() != nil {
											if analyzer.IsEmptyHandling(bBlock.TargetBlock(),bBlock.TargetBlock()) {
												analyzer.printAlarm(useSIL.GetSourceLine())
											}else {
												analyzer.handledPoint[offset] = useSIL.GetLine()
											}
										}
									}

								}
							}
						}
					}
				}

			case icg.Lda:
				arg, _ := sil.(*icg.StackOpcode)
				offsetStr := fmt.Sprint(arg.Params().Front().Next().Value)
				addressOffset = append(addressOffset, offsetStr)
			case icg.Ldi:
				offsetStr := addressOffset[len(addressOffset)-1]
				analyzer.IsErrorVar(sil, offsetStr)

			case icg.Sti:
				addressOffset = addressOffset[0 : len(addressOffset)-1]
			}

		}
	case *cfg.BranchBlock:



	}

	return analyzer.analysisCount
}
