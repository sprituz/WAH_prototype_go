// +build !sil_complete

package main

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"log"
	"strconv"
	"strings"

	"WAH_prototype_go-master/Src/astprinter"
	"WAH_prototype_go-master/Src/icg"
	"WAH_prototype_go-master/Src/icg/symbolTable"

	"os"

	"WAH_prototype_go-master/Src/analysisGraph/cfg"
	"WAH_prototype_go-master/Src/analysisGraph/vfg"
	"WAH_prototype_go-master/Src/wah"
)

func PrintDUChain(duChainofFunctions map[int]vfg.DUChain, strPool *symbolTable.StringPool) {
	for k, duChain := range duChainofFunctions {
		fmt.Println("==============================================")
		fmt.Printf("%s function\n", strPool.LookupSymbolName(k))
		fmt.Println("==============================================")
		fmt.Println("**********************************************")
		duChain.Print()
		fmt.Println("**********************************************\n")
	}
}
func DUChainFileGen(duChainofFunctions map[int]vfg.DUChain, strPool *symbolTable.StringPool, location string) {
	//printStr := strings.Builder{}
	for k, duChain := range duChainofFunctions {
		str := location + strPool.LookupSymbolName(k) + ".du"
		f, _ := os.Create(str)
		f.WriteString(duChain.String())
		f.Sync()
		f.Close()
	}
}

func CFGFileGen(controlFlowGraphs map[int]cfg.CFGBlock, strPool *symbolTable.StringPool, location string) {
	for k, root := range controlFlowGraphs {
		var f *os.File
		//fmt.Println("------------------------------------\n" + strPool.LookupSymbolName(k) + "\n------------------------------------\n")
		printStr := cfg.Print(root) + "}"
		cfg.InitPrinter()
		//fmt.Println(printStr)
		str := location + strPool.LookupSymbolName(k) + ".cfg"
		f, _ = os.Create(str)
		f.WriteString(printStr)
		f.Sync()
		f.Close()
	}
}

func SILFileGen(table *icg.SILTable, location string) {
	buffer := strings.Builder{}
	for k, v := range table.FunctionCodeTable() {
		for _, info := range v {

			if stackInfo, ok := info.(*icg.StackOpcode); ok {
				buffer.WriteString("\t" + stackInfo.String() + "\n")
			}
			if arithinfo, ok := info.(*icg.ArithmeticOpcode); ok {
				buffer.WriteString("\t" + arithinfo.String() + "\n")
			}
			if cinfo, ok := info.(*icg.ControlOpcode); ok {
				if cinfo.Opcode() == icg.Label {
					buffer.WriteString(cinfo.String() + "\n")
				} else {
					buffer.WriteString("\t" + cinfo.String() + "\n")
				}
			}

		}
		buffer.WriteString("\n")
		str := location + table.StringPool().LookupSymbolName(k) + ".sil"
		f, _ := os.Create(str)
		f.WriteString(buffer.String())
		f.Sync()
		f.Close()
		buffer = strings.Builder{}
	}

}

func PrintSIL(table *icg.SILTable) {
	buffer := strings.Builder{}
	for k, v := range table.FunctionCodeTable() {
		buffer.WriteString("------------------------------\n")
		buffer.WriteString(fmt.Sprintf("\tFunction : %s \n", table.StringPool().LookupSymbolName(k)))
		buffer.WriteString("------------------------------\n")

		for _, info := range v {

			if stackInfo, ok := info.(*icg.StackOpcode); ok {
				buffer.WriteString("\t" + strconv.Itoa(stackInfo.GetLine()) + ": " + stackInfo.String() + "\n")
			}
			if arithinfo, ok := info.(*icg.ArithmeticOpcode); ok {
				buffer.WriteString("\t" + strconv.Itoa(arithinfo.GetLine()) + ": " + arithinfo.String() + "\n")
			}
			if cinfo, ok := info.(*icg.ControlOpcode); ok {
				if cinfo.Opcode() == icg.Label {
					buffer.WriteString(strconv.Itoa(cinfo.GetLine()) + ": " + cinfo.String() + "\n")
				} else {
					buffer.WriteString("\t" + strconv.Itoa(cinfo.GetLine()) + ": " + cinfo.String() + "\n")
				}
			}

		}
		buffer.WriteString("\n")
	}

	fmt.Println(buffer.String())
}
func main() {
	var goFile string
	var fs *token.FileSet
	var f *ast.File
	var err error
	var conf types.Config
	var info *types.Info

	var strPool *symbolTable.StringPool

	var silTable *icg.SILTable
	var controlFlowGraphs map[int]cfg.CFGBlock
	var duChainofFunctions map[int]vfg.DUChain

	// console 입력으로 들어온 사용자 매개변수 처리 및 보안약점 분석을 위한 CFG, DUChain 생성
	if len(os.Args) >= 2 {
		for i, args := range os.Args {
			if i == 1 && args != "-h" {
				goFile = args
				fs = token.NewFileSet()

				f, err = parser.ParseFile(fs, goFile, nil, parser.AllErrors)
				if err != nil {
					log.Printf("could not parse %s: %v", goFile, err)
				}
				conf = types.Config{Importer: importer.ForCompiler(fs, "source", nil)}
				info = &types.Info{Types: make(map[ast.Expr]types.TypeAndValue)}
				if _, err := conf.Check("", fs, []*ast.File{f}, info); err != nil {
					log.Fatal(err) // type error
				}

				continue
			}

			switch args {
			// help option
			case "-h":
				fmt.Println("Usage : WAH.exe AnalysisTarget.go -option")
				fmt.Println("---------------------------------------------------------------------------------------")
				fmt.Println("\tOption")
				fmt.Println("\t  -h : Print usage and option list")
				fmt.Println("\t  -p : Print \"Ast\", \"DUChain\", \"CFG\" or \"SIL\" (CFG Print is not implementation yet)")
				fmt.Println("\t  -f : File Generate  \"SIL\", \"DUChain\" or \"CFG\" ")
				fmt.Println("---------------------------------------------------------------------------------------")
				os.Exit(0)
			// print option
			case "-p":
				if len(os.Args) <= i+1 {
					log.Fatal(fmt.Errorf("error : The -p option requires one of \"AST\", \"DUChain\", \"CFG\" or \"SIL\" arguments."))

				}

				for j := i + 1; j < len(os.Args); j++ {
					printTarget := os.Args[j]
					if strings.Contains(printTarget, "-") {
						if j == i+1 {
							log.Fatal(fmt.Errorf("error : The -p option requires one of \"AST\", \"DUChain\", \"CFG\" or \"SIL\" arguments."))

						}
						break
					}

					switch printTarget {
					case "AST":
						astprinter.PrintAst(f)
					case "DUChain":
						PrintDUChain(duChainofFunctions, strPool)
					case "CFG":
					case "SIL":
						PrintSIL(silTable)
					}
				}

			//file generate option
			case "-f":
				if len(os.Args) <= i+1 {
					log.Fatal(fmt.Errorf("error : The -f option requires one of \"SIL\", \"DUChain\", or \"CFG\" arguments."))

				}

				for j := i + 1; j < len(os.Args); j++ {
					genTarget := os.Args[j]
					if strings.Contains(genTarget, "-") {
						if j == i+1 {
							log.Fatal(fmt.Errorf("error : The -f option requires one of \"SIL\", \"DUChain\", or \"CFG\" arguments."))

						}
						break
					}

					switch genTarget {
					case "DUChain":
						//PrintDUChain(duChainofFunctions, strPool)
						location := "./DUChainResult/"
						DUChainFileGen(duChainofFunctions, strPool, location)
					case "CFG":
						location := "./CFGResult/"
						CFGFileGen(controlFlowGraphs, strPool, location)
					case "SIL":
						location := "./SILResult/"
						//PrintSIL(silTable)
						SILFileGen(silTable, location)

					}
				}
			}
		}
	} else {
		log.Fatal(fmt.Errorf("Please run it with reference to the -h option "))
	}

	//ast Analysis
	astAnalyzer := new(wah.ASTAnalyzer)
	astAnalyzer.Init(goFile, fs)
	errCount := astAnalyzer.Analysis(f, info)

	fmt.Printf("\t Total error count : %d",errCount)

}
