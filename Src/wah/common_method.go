package wah

// 모든 분석 항목에서 공동으로 사용되는 함수들을 모아놓은 파일
import (

	//"../../icg"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"WAH_prototype_go-master/Src/analysisGraph/cfg"
	"WAH_prototype_go-master/Src/analysisGraph/vfg"
	"WAH_prototype_go-master/Src/icg"
	"WAH_prototype_go-master/Src/icg/symbolTable"
)

// Analysis ... : 보안약점 항목을 검사할 때 외부에서 사용하는 함수
func Analysis(fs *token.FileSet, f *ast.File, info *types.Info, analysisFile string, block cfg.CFGBlock, chain vfg.DUChain, codeList []icg.CodeInfo,litTable *symbolTable.LiteralTable) {
	analyzer := new(ChainCodeAnalyzer)
	analyzer.Init(fs, analysisFile, chain, codeList,litTable)

	analyzer.WeaknessAnalysis(f, info, block)
	totalCount := analyzer.TotalCount()

	fmt.Printf("\t Total weakness count : %d\n",totalCount)
}

/* findRhsList ... : code list를 역해석 하여 lhs (definition)에 할당에 사용된 rhs (use)리스트를 찾는 함수
 *  ex ) a = b + c + 1 에서 a 할당에 사용된 b, c 를 찾아내는 함수
 */
func FindRhsList(defLine int, codeList []icg.CodeInfo) []icg.CodeInfo {
	reverseCodeList := sliceReverse(codeList)
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

		sp = sp + code.GetPushParameterNum() - code.GetPopParameterNum()

		if code.Opcode() == icg.Lod || code.Opcode() == icg.Lda {
			res = append(res, code)
		}

		if sp == 0 {
			if isDebug {
				fmt.Printf("Bottom up analysis completed!\n")
				fmt.Printf("\t definition code : %s\n \t target code list : %s\n\n", analysisRange[0].String(), fmt.Sprint(res))
				fmt.Println("--------------------------------------------------------------------------------\n")
			}
			break
		}
	}

	return res
}

func FindLhsList(defLine int, codeList []icg.CodeInfo) []icg.CodeInfo {

	defCodeIndex := findSILIndex(codeList, defLine)

	analysisRange := codeList[defCodeIndex:]
	sp := analysisRange[0].GetPushParameterNum() //pop이 소모자, push가 판매
	var res []icg.CodeInfo
	if isDebug {
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Printf("\tFind variable used to assign variable %s \n", analysisRange[0])
		fmt.Println("--------------------------------------------------------------------------------")
	}

	for i, code := range analysisRange {
		if isDebug {
			fmt.Printf("code : %s \n\tpush : %d \tpop : %d\n", code.String(), code.GetPushParameterNum(), code.GetPopParameterNum())
		}
		if i == 0 && sp != 0 {
			if isDebug {
				fmt.Printf("\tSatisfaction value for foward analysis (if 0, analysis ends) : %d\n\n", sp)
			}
			continue
		} else if i == 0 && sp == 0 {
			if isDebug {
				fmt.Printf("\tSatisfaction value for foward analysis (if 0, analysis ends) : %d\n\n", sp)
				fmt.Printf("analysis completed!\n")
				fmt.Printf("\t definition code : %s\n \t target code list : %s\n\n", analysisRange[0].String(), fmt.Sprint(res))
				fmt.Println("--------------------------------------------------------------------------------\n")
			}
			break
		}

		sp = sp + code.GetPushParameterNum() - code.GetPopParameterNum()
		if isDebug {
			fmt.Printf("\tSatisfaction value for foward analysis (if 0, analysis ends) : %d\n\n", sp)
		}

		if code.Opcode() == icg.Str || code.Opcode() == icg.Sti {
			res = append(res, code)
		}

		if sp == 0 {
			if isDebug {
				fmt.Printf("analysis completed!\n")
				fmt.Printf("\t definition code : %s\n \t target code list : %s\n\n", analysisRange[0].String(), fmt.Sprint(res))
				fmt.Println("--------------------------------------------------------------------------------\n")
			}
			break
		}
	}

	return res
}

func sliceReverse(codeInfo []icg.CodeInfo) []icg.CodeInfo {
	var reverseCode []icg.CodeInfo = make([]icg.CodeInfo, len(codeInfo))
	for i, sil := range codeInfo {
		reverseCode[(len(reverseCode)-1)-i] = sil
	}

	return reverseCode
}
func findSILIndex(codeInfo []icg.CodeInfo, line int) int {
	res := -1
	for i, sil := range codeInfo {
		if sil.GetLine() == line {
			res = i
			break

		}
	}
	return res
}
