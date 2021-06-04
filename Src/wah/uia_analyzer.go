package wah

import (
	"fmt"

	"WAH_prototype_go-master/Src/analysisGraph/cfg"
	"WAH_prototype_go-master/Src/analysisGraph/vfg"
	"WAH_prototype_go-master/Src/icg"
)

type UIAAnalyzer struct {
	analysisFile  string
	chain         vfg.DUChain
	codeList      []icg.CodeInfo
	checkPoint    map[string]int // offset , line number
	analysisCount int
	taintList     []TaintInfo
	detectedLint  []int
}
type TaintInfo struct {
	offset   string
	codeInfo icg.CodeInfo
}

func (info *TaintInfo) Init(offset string, codeInfo icg.CodeInfo) {
	info.offset = offset
	info.codeInfo = codeInfo
}

func (analyzer *UIAAnalyzer) Init(analysisFile string, chain vfg.DUChain, codeList []icg.CodeInfo) {
	analyzer.analysisFile = analysisFile
	analyzer.chain = chain
	analyzer.codeList = codeList
	analyzer.checkPoint = make(map[string]int)
	analyzer.analysisCount = 0
}

func (analyzer *UIAAnalyzer) FindUncheckedVar(variable icg.CodeInfo) []icg.CodeInfo {

	if variable.Opcode() != icg.Call {
		return nil
	}

	defLine := variable.GetLine()
	defCodeIndex := findSILIndex(analyzer.codeList, defLine)
	endCodeIndex := defCodeIndex + variable.GetPushParameterNum() + 1
	analysisRange := analyzer.codeList[defCodeIndex:endCodeIndex]
	//sp := analysisRange[0].GetPushParameterNum() //pop이 소모자, push가 판매
	var res []icg.CodeInfo
	if isDebug {
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Printf("\tFind Unchecked Variables %s \n", analysisRange[0])
		fmt.Println("--------------------------------------------------------------------------------")
	}

	for _, code := range analysisRange {
		if isDebug {
			fmt.Printf("code : %s \n\tpush : %d \tpop : %d\n", code.String(), code.GetPushParameterNum(), code.GetPopParameterNum())
		}
		if code.Opcode() == icg.Str || code.Opcode() == icg.Sti {
			res = append(res, code)
		}
	}

	return res
}

func (analyzer *UIAAnalyzer) IsUncheckedVar(variable icg.CodeInfo, offset string) bool {
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
		fmt.Printf("\t Unchecked Variable Analysis : %s \n", variable)
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

		if code.Opcode() != icg.Call {
			sp = sp + code.GetPushParameterNum() - code.GetPopParameterNum()
		} else {
			sp = sp + code.GetPushParameterNum()
		}
		if isDebug {
			fmt.Printf("\tSatisfaction value for reverse analysis (if 0, analysis ends) : %d\n\n", sp)
		}

		if code.Opcode() == icg.Call && sp >= 0 {
			res = true

			if isDebug {
				fmt.Printf("Unchecked variable analysis completed!\n")
				fmt.Printf("\t definition code : %s\n \t target code : %s\n\n", analysisRange[0].String(), code.String())
				fmt.Println("--------------------------------------------------------------------------------\n")
			}
			break
		}

		if sp == 0 {
			res = false
			break
		}

	}

	return res
}

func (analyzer *UIAAnalyzer) isDetected(line int) bool{
	res := false
	for _, dline := range analyzer.detectedLint {
		if dline == line {
			res = true
		}
	}
	return res
}
func (analyzer *UIAAnalyzer) printAlarm(linenum int) {
	var ccw CCW = UNCHECKED_INPUT_ARGUMENTS
	if isFirstDetect {
		fmt.Printf("chaincode weakness detected:\n")
		isFirstDetect = false
	}
	if !analyzer.isDetected(linenum) {
		analyzer.detectedLint = append(analyzer.detectedLint,linenum)
		fmt.Printf("\t CCW-%03d : %s\n", ccw, ccw.String())
		fmt.Printf("\t %s : %d\n\n", analyzer.analysisFile, linenum)
		analyzer.analysisCount++
	}



}

func findAddressOffset(codeList []icg.CodeInfo, index int) string {
	sp := -codeList[index].GetPopParameterNum()
	res := ""
	for i := index + 1; i < len(codeList); i++ {
		code := codeList[i]
		sp = sp + code.GetPushParameterNum() - code.GetPopParameterNum()

		if sp == 0 {
			if code.Opcode() == icg.Lda {
				lda := code.(*icg.StackOpcode)
				offset := fmt.Sprint(lda.Params().Front().Next().Value)
				res = offset
			}
		}
	}
	return res
}
func (analyzer *UIAAnalyzer) IsTaintedVar(variable icg.CodeInfo, offset string) bool {
	reverseCodeList := sliceReverse(analyzer.codeList)
	defLine := 0
	res := false
	// 변수가 이미 오염된 변수인경우

	if variable.Opcode() == icg.Lod {
		lod := variable.(*icg.StackOpcode)

		defLine, _ = analyzer.chain.LookUpDefOfUse(offset, lod.GetLine())
		for _, taint := range analyzer.taintList {
			def := taint.codeInfo.GetLine()
			if defLine == def {
				res = true
				break
			}
		}
	} else if variable.Opcode() == icg.Ldi {
		ldi := variable.(*icg.StackOpcode)
		defLine, _ = analyzer.chain.LookUpDefOfUse(offset, ldi.GetLine())
		for _, taint := range analyzer.taintList {
			def := taint.codeInfo.GetLine()
			if defLine == def {
				res = true
				break
			}
		}
	} else if variable.Opcode() == icg.Str {
		defLine = variable.GetLine()
	} else {
		return res
	}

	if !res {

		defCodeIndex := findSILIndex(reverseCodeList, defLine)

		analysisRange := reverseCodeList[defCodeIndex:]
		sp := -analysisRange[0].GetPopParameterNum() //pop이 소모자, push가 판매

		if isDebug {
			fmt.Println("--------------------------------------------------------------------------------")
			fmt.Printf("\t Tainted Analysis : %s \n", variable)
			fmt.Println("--------------------------------------------------------------------------------")
		}

		for i, code := range analysisRange {
			if isDebug {
				fmt.Printf("code : %s \n\tpush : %d \tpop : %d\n\n", code.String(), code.GetPushParameterNum(), code.GetPopParameterNum())
			}
			if i == 0 {
				continue
			}

			sp = sp + code.GetPushParameterNum() - code.GetPopParameterNum()

			// rhs에 사용된 변수들인 경우
			if code.Opcode() == icg.Lod {
				lod := code.(*icg.StackOpcode)
				rhsOffset := fmt.Sprint(lod.Params().Front().Next().Value)

				rhsDef, _ := analyzer.chain.LookUpDefOfUse(rhsOffset, code.GetLine())

				for _, taint := range analyzer.taintList {
					taintedDef := taint.codeInfo.GetLine()
					if rhsDef == taintedDef {
						res = true
						if isDebug {
							fmt.Printf("The variable %s is tainted by %s\n\n", analysisRange[0].String(), code.String())
						}
						break
					}
				}
			} else if code.Opcode() == icg.Ldi {
				rhsOffset := findAddressOffset(analysisRange, i)
				rhsDef, _ := analyzer.chain.LookUpDefOfUse(rhsOffset, code.GetLine())

				for _, taint := range analyzer.taintList {
					taintedDef := taint.codeInfo.GetLine()
					if rhsDef == taintedDef {
						res = true
						if isDebug {
							fmt.Printf("The variable %s is tainted by %s\n", analysisRange[0].String(), code.String())
						}
						break
					}
				}
			}

			if sp == 0 {
				if isDebug {
					fmt.Printf("tainted analysis completed!\n")
					fmt.Printf("\t definition code : %s\n \t is tainted : %v\n\n", analysisRange[0].String(), res)
					fmt.Println("--------------------------------------------------------------------------------\n")
				}
				break
			}
		}
	}else {
		if isDebug {
			fmt.Println("--------------------------------------------------------------------------------")
			fmt.Printf("\t Tainted Analysis : %s \n", variable)
			fmt.Println("--------------------------------------------------------------------------------")

			fmt.Printf("The variable %s is already tainted \n", variable)
			fmt.Println("--------------------------------------------------------------------------------\n")

		}
	}
	return res
}
func (analyzer *UIAAnalyzer) UIAAnalysis(block cfg.CFGBlock) int {

	switch b := block.(type) {
	case *cfg.BasicBlock:
		var addressOffset []string
		for _, sil := range b.CodeList() {

			switch sil.Opcode() {
			case icg.Str:
				// 오염된 변수인지 확인
				arg, _ := sil.(*icg.StackOpcode)
				offsetStr := fmt.Sprint(arg.Params().Front().Next().Value)

				// 확인되지 않은 변수이면 오염된 변수 리스트에 추가
				if analyzer.IsUncheckedVar(sil, offsetStr) {
					taintInfo := new(TaintInfo)
					taintInfo.Init(offsetStr, sil)

					analyzer.taintList = append(analyzer.taintList, *taintInfo)
				}
			case icg.Lod:
				arg, _ := sil.(*icg.StackOpcode)
				offsetStr := fmt.Sprint(arg.Params().Front().Next().Value)

				/*if analyzer.IsUncheckedVar(sil, offsetStr) {
					if sil.ParentStmt() == icg.IfStmt {
						analyzer.checkPoint[offsetStr] = sil.GetLine()
					} else {
						if checkPoint, ok := analyzer.checkPoint[offsetStr]; ok {
							if checkPoint > sil.GetLine() {
								analyzer.printAlarm(sil.GetSourceLine())
							}
						} else {
							analyzer.printAlarm(sil.GetSourceLine())
						}
					}
				}else */
				if analyzer.IsTaintedVar(sil, offsetStr) { // 사용된 변수가 오염된 변수인 경우 검증하는지 검사
					if sil.ParentStmt() == icg.IfStmt {
						analyzer.checkPoint[offsetStr] = sil.GetLine()
					} else {
						if checkPoint, ok := analyzer.checkPoint[offsetStr]; ok {
							if checkPoint > sil.GetLine() {
								analyzer.printAlarm(sil.GetSourceLine())
							}
						} else {
							analyzer.printAlarm(sil.GetSourceLine())
						}
					}
				}

			case icg.Lda:
				arg, _ := sil.(*icg.StackOpcode)
				offsetStr := fmt.Sprint(arg.Params().Front().Next().Value)
				addressOffset = append(addressOffset, offsetStr)
			case icg.Ldi:
				offsetStr := addressOffset[len(addressOffset)-1]

				//현재 블록에 있는 def가 global인가
				if analyzer.IsTaintedVar(sil, offsetStr) {
					if sil.ParentStmt() == icg.IfStmt {
						analyzer.checkPoint[offsetStr] = sil.GetLine()
					} else {
						if checkPoint, ok := analyzer.checkPoint[offsetStr]; ok {
							if checkPoint > sil.GetLine() {
								analyzer.printAlarm(sil.GetSourceLine())
								//analyzer.analysisCount++
							}
						} else {
							analyzer.printAlarm(sil.GetSourceLine())
							//analyzer.analysisCount++
						}
					}

				}

			case icg.Sti:
				addressOffset = addressOffset[0 : len(addressOffset)-1]
			}

		}

	}

	return analyzer.analysisCount
}
