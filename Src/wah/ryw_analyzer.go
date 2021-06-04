package wah

import (
	"WAH_prototype_go-master/Src/analysisGraph/cfg"
	"WAH_prototype_go-master/Src/analysisGraph/vfg"
	"WAH_prototype_go-master/Src/icg"
	"WAH_prototype_go-master/Src/icg/symbolTable"
	"fmt"
	"github.com/mitchellh/go-z3"
	"io/ioutil"
	"log"
	"os/exec"
	"strconv"
	"strings"
)

type RYWAnalyzer struct {
	analysisFile  string
	defList       []int
	putStateLine  []int
	chain         vfg.DUChain
	codeList      []icg.CodeInfo
	analysisCount int
	ctx           *z3.Context
	solver        *z3.Solver
	symbolList    map[string]*SMTSymbol
	literalSymbol map[string]*z3.AST
	opStack       []string

	keyFormula  string
	putStateKey string

	litTable *symbolTable.LiteralTable
}
type SMTSymbol struct {
	symbolName string
	symbolType icg.SilType
}

func (analyzer *RYWAnalyzer) Init(analysisFile string, chain vfg.DUChain, codeList []icg.CodeInfo, litTable *symbolTable.LiteralTable) {
	analyzer.analysisFile = analysisFile
	analyzer.chain = chain
	analyzer.codeList = codeList
	analyzer.analysisCount = 0
	conf := z3.NewConfig()
	analyzer.ctx = z3.NewContext(conf)
	//conf.Close()
	//defer analyzer.ctx.Close()

	analyzer.solver = analyzer.ctx.NewSolver()
	//defer analyzer.solver.Close()

	analyzer.symbolList = make(map[string]*SMTSymbol)
	analyzer.literalSymbol = make(map[string]*z3.AST)
	analyzer.keyFormula = ""
	analyzer.litTable = litTable

}

func (analyzer *RYWAnalyzer) FindKeyVar(call icg.CodeInfo) icg.CodeInfo {
	reverseCodeList := sliceReverse(analyzer.codeList)
	defCodeIndex := findSILIndex(reverseCodeList, call.GetLine())

	analysisRange := reverseCodeList[defCodeIndex:]
	sp := -analysisRange[0].GetPopParameterNum() //pop이 소모자, push가 판매
	var res []icg.CodeInfo
	if isDebug {
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Printf("\tKey Parameter Analysis %s \n", analysisRange[0])
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

		sp = sp + code.GetPushParameterNum() - code.GetPopParameterNum()
		if isDebug {
			fmt.Printf("\tSatisfaction value for reverse analysis (if 0, analysis ends) : %d\n\n", sp)
		}

		if code.Opcode() == icg.Lod || code.Opcode() == icg.Lda {
			res = append(res, code)
		}

		if sp == 0 {
			if isDebug {
				fmt.Printf("Inverse analysis completed!\n")
				fmt.Printf("\t Key parameter : %s\n\n", fmt.Sprint(res[len(res)-1]))
				fmt.Println("--------------------------------------------------------------------------------\n")
			}
			break
		}
	}

	return res[len(res)-1]

}
func (analyzer *RYWAnalyzer) printAlarm(line int) {
	var ccw CCW = READ_YOUR_WRITE
	if isFirstDetect {
		fmt.Printf("chaincode weakness detected:\n")
		isFirstDetect = false
	}
	fmt.Printf("\t CCW-%03d : %s\n", ccw, ccw.String())
	fmt.Printf("\t %s : %d\n", analyzer.analysisFile, analyzer.putStateLine[len(analyzer.putStateLine)-1])
	fmt.Printf("\t %s : %d\n\n", analyzer.analysisFile, line)
	analyzer.analysisCount++
}
func (analyzer *RYWAnalyzer) RYWAnalysis(block cfg.CFGBlock) int {
	switch b := block.(type) {
	case *cfg.CallBlock:
		opcode := b.CodeList()[0]
		if callOp, ok := opcode.(*icg.ControlOpcode); ok {
			funcName := fmt.Sprint(callOp.Params().Front().Value)
			// 함수가 putstate 함수인 경우
			if strings.Contains(funcName, ".PutState") {
				keyOpcode := analyzer.FindKeyVar(callOp)
				keyParam := keyOpcode.(*icg.StackOpcode)
				offset := fmt.Sprint(keyParam.Params().Front().Next().Value)
				definition, _ := analyzer.chain.LookUpDefOfUse(offset, keyParam.GetLine())
				analyzer.defList = append(analyzer.defList, definition)
				analyzer.putStateLine = append(analyzer.putStateLine, callOp.GetSourceLine())
			} else if strings.Contains(funcName, ".GetState") {
				// putstate가 호출되엇는지
				if len(analyzer.defList) > 0 {
					keyOpcode := analyzer.FindKeyVar(callOp)
					keyParam := keyOpcode.(*icg.StackOpcode)
					offset := fmt.Sprint(keyParam.Params().Front().Next().Value)
					definition, _ := analyzer.chain.LookUpDefOfUse(offset, keyParam.GetLine())

					for i, def := range analyzer.defList {
						if def == definition {
							var ccw CCW = READ_YOUR_WRITE
							if isFirstDetect {
								fmt.Printf("chaincode weakness detected:\n")
								isFirstDetect = false
							}
							fmt.Printf("\t CCW-%03d : %s\n", ccw, ccw.String())
							fmt.Printf("\t %s : %d\n", analyzer.analysisFile, keyParam.GetSourceLine())
							fmt.Printf("\t %s : %d\n\n", analyzer.analysisFile, analyzer.putStateLine[i])
							analyzer.analysisCount++
						}
					}
				}
			}
		}
	}
	return analyzer.analysisCount
}
func (analyzer *RYWAnalyzer) IsBinaryOp(op icg.ArithmeticOpcode) bool {
	res := true
	switch op.Opcode() {
	case icg.Add:

	case icg.Sub:

	case icg.Mul:

	case icg.Div:
		// go z3 not implements div
	case icg.Mod:
		// go z3 not implements mod
	case icg.Eq:

	case icg.Ne:

	case icg.Ge:

	case icg.Gt:

	case icg.Le:

	case icg.Lt:

	case icg.Band:
		// go z3 not implements Bit operation
	case icg.Bor:
		// go z3 not implements Bit operation
	case icg.Bxor:
		// go z3 not implements Bit operation
	case icg.Shl:
		// go z3 not implements Shift operation
	case icg.Shr:
		// go z3 not implements Shift operation
	case icg.Ushr:
		// go z3 not implements Shift operation
	case icg.And:

	case icg.Or:

	default:
		res = false
	}

	return res
}
func (analyzer *RYWAnalyzer) makeBinaryExpression(op icg.ArithmeticOpcode, sym1 string, sym2 string) string {
	var formula string = ""
	switch op.Opcode() {
	case icg.Add:
		formula = fmt.Sprintf("(str.++ %s %s)", sym1, sym2)
	case icg.Sub:
		formula = fmt.Sprintf("(- %s %s)", sym1, sym2)
	case icg.Mul:
		formula = fmt.Sprintf("(* %s %s)", sym1, sym2)
	case icg.Div:
		formula = fmt.Sprintf("(/ %s %s)", sym1, sym2)
	case icg.Mod:
		formula = fmt.Sprintf("(% %s %s)", sym1, sym2)
	case icg.Eq:
		formula = fmt.Sprintf("(= %s %s)", sym1, sym2)
	case icg.Ne:
		formula = fmt.Sprintf("(not (= %s %s))", sym1, sym2)
	case icg.Ge:
		formula = fmt.Sprintf("(>= %s %s)", sym1, sym2)
	case icg.Gt:
		formula = fmt.Sprintf("(> %s %s)", sym1, sym2)
	case icg.Le:
		formula = fmt.Sprintf("(<= %s %s)", sym1, sym2)
	case icg.Lt:
		formula = fmt.Sprintf("(< %s %s)", sym1, sym2)
	case icg.Band:
		// go z3 not implements Bit operation
	case icg.Bor:
		// go z3 not implements Bit operation
	case icg.Bxor:
		// go z3 not implements Bit operation
	case icg.Shl:
		// go z3 not implements Shift operation
	case icg.Shr:
		// go z3 not implements Shift operation
	case icg.Ushr:
		// go z3 not implements Shift operation
	case icg.And:
		formula = fmt.Sprintf("(and %s %s)", sym1, sym2)
	case icg.Or:
		formula = fmt.Sprintf("(or %s %s)", sym1, sym2)
	}

	return formula
}
func execProgram(program string, args ...string) []byte{
	cmd := exec.Command(program, args...)

	res, _ := cmd.Output()
	return res
}
func (analyzer *RYWAnalyzer) makeSymbol(opcode icg.StackOpcode) *SMTSymbol {

	if opcode.Opcode() == icg.Lod || opcode.Opcode() == icg.Str {
		symName := "offset" + fmt.Sprint(opcode.Params().Front().Next().Value)

		return &SMTSymbol{symbolName: symName, symbolType: opcode.Type()}
	}
	return nil
}

func (analyzer *RYWAnalyzer) makeSMTFormula(genRange []icg.CodeInfo) string {
	var res string = ""
	var formulaList []string

	for _, code := range genRange {
		switch c := code.(type) {
		case *icg.StackOpcode:
			if c.Opcode() == icg.Lod {
				offset := fmt.Sprint(c.Params().Front().Next().Value)
				genRange := analyzer.GetFormulaGenRange(c, offset)

				if _, ok := analyzer.symbolList[offset]; !ok {
					lodFormula := analyzer.makeSMTFormula(genRange)
					formulaList = append(formulaList, lodFormula)
				}

				analyzer.opStack = append(analyzer.opStack, analyzer.symbolList[offset].symbolName)

			} else if c.Opcode() == icg.Str {
				offset := fmt.Sprint(c.Params().Front().Next().Value)
				if _, ok := analyzer.symbolList[offset]; !ok {
					smtSym := analyzer.makeSymbol(*c)
					analyzer.symbolList[offset] = smtSym
					res = analyzer.symbolList[offset].symbolName
				}

				if len(analyzer.opStack) >= 1 {
					sym1 := analyzer.opStack[len(analyzer.opStack)-1]
					analyzer.opStack = analyzer.opStack[0 : len(analyzer.opStack)-1]

					sym2 := analyzer.symbolList[offset].symbolName

					formula := "(= " + sym1 + " " + sym2 + ")"

					formulaList = append(formulaList, formula)
				}
			} else if c.Opcode() == icg.Lda {
				if c.Type() == icg.Sp {
					literalOffset := fmt.Sprint(c.Params().Front().Next().Value)
					literalOffset = literalOffset[1:]
					literalAdd,err := strconv.Atoi(literalOffset)
					if err != nil {
						log.Fatal("error")
					}

					lit,ok := analyzer.litTable.GetLiteral(literalAdd)
					if !ok {
						log.Fatal("error : not found literal")
					}

					analyzer.opStack = append(analyzer.opStack, lit)
				}
			}

		case *icg.ArithmeticOpcode:
			if analyzer.IsBinaryOp(*c) {
				sym1 := analyzer.opStack[len(analyzer.opStack)-1]
				analyzer.opStack = analyzer.opStack[0 : len(analyzer.opStack)-1]

				sym2 := analyzer.opStack[len(analyzer.opStack)-1]
				analyzer.opStack = analyzer.opStack[0 : len(analyzer.opStack)-1]
				analyzer.opStack = append(analyzer.opStack, analyzer.makeBinaryExpression(*c, sym2, sym1))
			}
		}
	}
	if len(formulaList) > 0 {
		res = formulaList[0]

		for i := 1; i < len(formulaList); i++ {
			res = fmt.Sprintf("(and %s %s)", res, formulaList[i])
		}
	}

	return res
}

func (analyzer *RYWAnalyzer) GetFormulaGenRange(variable icg.CodeInfo, offset string) []icg.CodeInfo {
	reverseCodeList := sliceReverse(analyzer.codeList)
	defLine := 0

	if variable.Opcode() == icg.Lod {
		lod := variable.(*icg.StackOpcode)

		defLine, _ = analyzer.chain.LookUpDefOfUse(offset, lod.GetLine())
	} else if variable.Opcode() == icg.Ldi {
		ldi := variable.(*icg.StackOpcode)
		defLine, _ = analyzer.chain.LookUpDefOfUse(offset, ldi.GetLine())
	} else if variable.Opcode() == icg.Str {
		defLine = variable.GetLine()
	} else {
		defLine = variable.GetLine()
	}

	defCodeIndex := findSILIndex(reverseCodeList, defLine)

	analysisRange := reverseCodeList[defCodeIndex:]
	sp := -analysisRange[0].GetPopParameterNum() //pop이 소모자, push가 판매
	var res []icg.CodeInfo
	if isDebug {
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Printf("\tFind variable used to assign variable %s \n", analysisRange[0])
		fmt.Println("--------------------------------------------------------------------------------")
	}

	for i, code := range analysisRange {
		if isDebug {
			fmt.Printf("code : %s \n\tpush : %d \tpop : %d\n\n", code.String(), code.GetPushParameterNum(), code.GetPopParameterNum())
		}
		if i == 0 {
			continue
		}

		if code.Opcode() != icg.Call {
			sp = sp + code.GetPushParameterNum() - code.GetPopParameterNum()
		} else {
			sp = sp + code.GetPushParameterNum()
		}

		if sp >= 0 {
			reverseRange := analysisRange[0 : i+1]
			res = sliceReverse(reverseRange)
			break
		}
	}

	return res
}
func GenliteralFormula(literalList []*z3.AST, target int) []*z3.AST {
	t := target
	var literalFormulaList []*z3.AST
	for i := t; i < len(literalList)-1; i++ {
		literalFormulaList = append(literalFormulaList, literalList[t].Eq(literalList[i+1]).Not())
	}

	return literalFormulaList
}
func (analyzer *RYWAnalyzer) GenliteralFormulaList() []*z3.AST {
	var literalFormulaList []*z3.AST
	var literalList []*z3.AST

	for _, literal := range analyzer.literalSymbol {
		literalList = append(literalList, literal)
	}

	for i := 0; i < len(literalList)-1; i++ {
		literalFormulaList = append(literalFormulaList, GenliteralFormula(literalList, i)...)
	}

	return literalFormulaList
}
func SMTSymTypetoString(silType icg.SilType) string {
	res := ""
	switch silType {
	case icg.Sp:
		res = "String"
	}

	return res
}
func (analyzer *RYWAnalyzer) RYWAnalysisUsedZ3(block cfg.CFGBlock) {
	switch b := block.(type) {
	case *cfg.CallBlock:
		opcode := b.CodeList()[0]
		if callOp, ok := opcode.(*icg.ControlOpcode); ok {
			funcName := fmt.Sprint(callOp.Params().Front().Value)

			// 함수가 putstate 함수인 경우
			if strings.Contains(funcName, ".PutState") {
				keyOpcode := analyzer.FindKeyVar(callOp)
				keyParam := keyOpcode.(*icg.StackOpcode)
				offset := fmt.Sprint(keyParam.Params().Front().Next().Value)

				formulaRange := analyzer.GetFormulaGenRange(keyParam, offset)

				formula := analyzer.makeSMTFormula(formulaRange)

				analyzer.keyFormula = formula
				analyzer.putStateKey = offset
				analyzer.putStateLine = append(analyzer.putStateLine, callOp.GetSourceLine())
				//analyzer.solver.Assert(formula)

			} else if strings.Contains(funcName, ".GetState") {
				if analyzer.keyFormula != "" {
					keyOpcode := analyzer.FindKeyVar(callOp)
					keyParam := keyOpcode.(*icg.StackOpcode)
					offset := fmt.Sprint(keyParam.Params().Front().Next().Value)
					formulaRange := analyzer.GetFormulaGenRange(keyParam, offset)

					formula := analyzer.makeSMTFormula(formulaRange)
					//analyzer.solver.Assert(formula)
					weaknessFormula := fmt.Sprintf("(and %s %s)", analyzer.keyFormula, formula)

					fmt.Println("formulaGenRange")
					for _,sil := range formulaRange {
						fmt.Println(sil.String())
					}

					keyEq := fmt.Sprintf("(= %s %s)", analyzer.symbolList[analyzer.putStateKey].symbolName, analyzer.symbolList[offset].symbolName)

					//analyzer.solver.Assert(keyEq)
					weaknessFormula = fmt.Sprintf("(and %s %s)", weaknessFormula, keyEq)

					weaknessFormula = fmt.Sprintf("(assert %s)\n", weaknessFormula)
					for _,sym := range analyzer.symbolList {
						weaknessFormula = fmt.Sprintf("(declare-const %s %s)\n",sym.symbolName,SMTSymTypetoString( sym.symbolType)) +weaknessFormula
					}
					weaknessFormula += "(check-sat)\n"


					data := []byte(weaknessFormula)
					ioutil.WriteFile("./ryw.smt",data,0644)

					res := execProgram("./z3" , "-smt2","./ryw.smt")
					analysisRes := string(res)
					analysisRes = analysisRes[:len(analysisRes)-1]
					//fmt.Println(string(res))
					if analysisRes ==  "sat"{
						analyzer.printAlarm(keyParam.GetSourceLine())
					}else {
						fmt.Println("RYW not exist")
					}

				}
			}
		}
	}
}

// func (analyzer *RYWAnalyzer) RYWAnalysis(block cfg.CFGBlock) {

// 	switch b := block.(type) {
// 	case *cfg.BasicBlock:
// 		var addressOffset []string
// 		for i, sil := range b.CodeList() {
// 			if sil.Opcode() == icg.Ldp {
// 				analyzer.ldpCount++
// 				if i == len(b.CodeList())-1 {
// 					analyzer.ldpCount = 0
// 				}
// 				continue
// 			}

// 			if analyzer.IsLodParams() {
// 				switch sil.Opcode() {
// 				case icg.Lod:
// 					if analyzer.IsGF(sil) {
// 						if b.LinkedBlock() != nil {
// 							analyzer.GFDeclAnalysis(b.LinkedBlock())
// 						}
// 					}
// 				case icg.Lda:
// 					arg, _ := sil.(*icg.StackOpcode)
// 					offsetStr := fmt.Sprint(arg.Params().Front().Next().Value)
// 					addressOffset = append(addressOffset, offsetStr)
// 				case icg.Ldi:
// 					offsetStr := addressOffset[len(addressOffset)-1]
// 					def, _ := analyzer.chain.LookUpDefOfUse(offsetStr, sil.GetLine())
// 					targets := FindRhsList(def, analyzer.codeList)

// 					for _, target := range targets {
// 						//현재 블록에 있는 def가 global인가
// 						if analyzer.IsGF(target) {
// 							//analyzer.gflst = append(analyzer.gflst, offsetStr)
// 							if b.LinkedBlock() != nil {
// 								analyzer.GFDeclAnalysis(b.LinkedBlock())
// 							}
// 							analyzer.gflst = nil
// 						}
// 					}

// 				case icg.Sti:
// 					addressOffset = addressOffset[0 : len(addressOffset)-1]
// 				}
// 			}
// 		}
// 		analyzer.gflst = nil
// 	case *cfg.CallBlock:
// 		analyzer.ldpCount--

// 		opcode := b.CodeList()[0]
// 		if callOp, ok := opcode.(*icg.ControlOpcode); ok {
// 			funcName := fmt.Sprint(callOp.Params().Front().Value)
// 			if strings.Contains(funcName, ".PutState") || strings.Contains(funcName, ".GetState") {
// 				fmt.Println("--------------------------------------------------------------------------------------")
// 				fmt.Println("  Do not use global variables or fields receiver as parameters \n  on the PutState or GetState functions.")
// 				fmt.Println("--------------------------------------------------------------------------------------")

// 				fmt.Println("\t * Code information with security weakness. ")
// 				fmt.Printf("\t\tSource code line  %d : %s\n", callOp.GetSourceLine(), callOp.String())
// 				fmt.Printf("\t * The offset of global variable or receiver field used as parameter: %s \n", analyzer.gflst[len(analyzer.gflst)-1])
// 				if len(analyzer.gflst) != 1 {
// 					fmt.Printf("\t\t")
// 					for i := len(analyzer.gflst) - 1; i >= 0; i-- {
// 						fmt.Printf("%s", analyzer.gflst[i])
// 						if i != 0 {
// 							fmt.Printf(" <- ")
// 						} else {
// 							fmt.Println()
// 						}
// 					}
// 				}

// 				fmt.Printf("\t * The source code line of global variable or receiver field used as parameter: %d \n", analyzer.gfSourceLine[len(analyzer.gflst)-1])
// 				if len(analyzer.gflst) != 1 {
// 					fmt.Printf("\t\t")
// 					for i := len(analyzer.gflst) - 1; i >= 0; i-- {
// 						fmt.Printf("%d", analyzer.gfSourceLine[i])
// 						if i != 0 {
// 							fmt.Printf(" <- ")
// 						} else {
// 							fmt.Println()
// 						}
// 					}
// 				}
// 				fmt.Println("--------------------------------------------------------------------------------------")

// 			}
// 		}
// 		if analyzer.IsLodParams() {
// 			if b.TargetBlock() != nil {
// 				analyzer.GFDeclAnalysis(b.TargetBlock())
// 			}
// 			if b.UjpBlock() != nil {
// 				analyzer.GFDeclAnalysis(b.UjpBlock())
// 			}
// 		}
// 	}
// }
