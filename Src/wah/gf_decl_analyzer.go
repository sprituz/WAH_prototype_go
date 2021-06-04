package wah

import (
	"fmt"
	"strings"

	"WAH_prototype_go-master/Src/analysisGraph/cfg"
	"WAH_prototype_go-master/Src/analysisGraph/vfg"
	"WAH_prototype_go-master/Src/icg"
)

type GFDeclAnalyzer struct {
	analysisFile  string
	ldpCount      int
	gflst         []string
	gfSourceLine  []int
	chain         vfg.DUChain
	codeList      []icg.CodeInfo
	analysisCount int
}

func (analyzer *GFDeclAnalyzer) Init(analysisFile string, chain vfg.DUChain, codeList []icg.CodeInfo) {
	analyzer.analysisFile = analysisFile
	analyzer.ldpCount = 0
	analyzer.chain = chain
	analyzer.codeList = codeList
	analyzer.analysisCount = 0
}

func (analyer *GFDeclAnalyzer) IsGF(target icg.CodeInfo) bool {
	isGlobal := false
	upperDef := -1
	if target.Opcode() == icg.Lod {
		lod := target.(*icg.StackOpcode)
		offset := fmt.Sprint(lod.Params().Front().Next().Value)
		if len(offset) > 0 {
			startSym := string(offset[0])
			if startSym == "$" {
				isGlobal = true
			} else {
				upperDef, _ = analyer.chain.LookUpDefOfUse(offset, target.GetLine())
				targets := FindRhsList(upperDef, analyer.codeList)
				for _, target := range targets {
					isGlobal = analyer.IsGF(target)
					if isGlobal {
						break
					}
				}
			}
		}

		if isGlobal {
			analyer.gflst = append(analyer.gflst, offset)
			analyer.gfSourceLine = append(analyer.gfSourceLine, lod.GetSourceLine())
		}

	} else if target.Opcode() == icg.Lda {
		lda := target.(*icg.StackOpcode)
		offset := fmt.Sprint(lda.Params().Front().Next().Value)
		if len(offset) > 0 {
			startSym := string(offset[0])
			if startSym == "$" {
				isGlobal = true
			} else {
				if lda.IsReceiver() {
					offset += " (Receiver)"
					isGlobal = true
				}
			}

			if isGlobal {
				analyer.gflst = append(analyer.gflst, offset)
				analyer.gfSourceLine = append(analyer.gfSourceLine, lda.GetSourceLine())
			}
		}
	}

	return isGlobal
}
func (analyzer *GFDeclAnalyzer) IsLodParams() bool {
	res := false
	if analyzer.ldpCount > 0 {
		res = true
	}
	return res
}
func (analyzer *GFDeclAnalyzer) GFDeclAnalysis(block cfg.CFGBlock) int{

	switch b := block.(type) {
	case *cfg.BasicBlock:
		var addressOffset []string
		for i, sil := range b.CodeList() {
			if sil.Opcode() == icg.Ldp {
				analyzer.ldpCount++
				if i == len(b.CodeList())-1 {
					analyzer.ldpCount = 0
				}
				continue
			}

			if analyzer.IsLodParams() {
				switch sil.Opcode() {
				case icg.Lod:
					if analyzer.IsGF(sil) {
						if b.LinkedBlock() != nil {
							analyzer.GFDeclAnalysis(b.LinkedBlock())
						}
					}
				case icg.Lda:
					arg, _ := sil.(*icg.StackOpcode)
					offsetStr := fmt.Sprint(arg.Params().Front().Next().Value)
					addressOffset = append(addressOffset, offsetStr)
				case icg.Ldi:
					offsetStr := addressOffset[len(addressOffset)-1]
					def, _ := analyzer.chain.LookUpDefOfUse(offsetStr, sil.GetLine())
					targets := FindRhsList(def, analyzer.codeList)

					for _, target := range targets {
						//현재 블록에 있는 def가 global인가
						if analyzer.IsGF(target) {
							//analyzer.gflst = append(analyzer.gflst, offsetStr)
							if b.LinkedBlock() != nil {
								analyzer.GFDeclAnalysis(b.LinkedBlock())
							}
							analyzer.gflst = nil
						}
					}

				case icg.Sti:
					addressOffset = addressOffset[0 : len(addressOffset)-1]
				}
			}
		}
		analyzer.gflst = nil
	case *cfg.CallBlock:
		analyzer.ldpCount--

		opcode := b.CodeList()[0]
		if callOp, ok := opcode.(*icg.ControlOpcode); ok {
			funcName := fmt.Sprint(callOp.Params().Front().Value)
			if strings.Contains(funcName, ".PutState") || strings.Contains(funcName, ".GetState") {
				var ccw CCW = GF_DECLARATION
				analyzer.analysisCount ++
				if isFirstDetect {
					fmt.Printf("chaincode weakness detected:\n")
					isFirstDetect = false
				}
				fmt.Printf("\t CCW-%03d : %s\n", ccw, ccw.String())
				fmt.Printf("\t %s : %d\n", analyzer.analysisFile, callOp.GetSourceLine())

				fmt.Printf("  \t The offset of global variable or receiver field used as parameter: %s \n", analyzer.gflst[len(analyzer.gflst)-1])
				if len(analyzer.gflst) != 1 {
					fmt.Printf("  \t\t")
					for i := len(analyzer.gflst) - 1; i >= 0; i-- {
						fmt.Printf("%s", analyzer.gflst[i])
						if i != 0 {
							fmt.Printf(" <- ")
						} else {
							fmt.Println()
						}
					}
				}
				fmt.Println()
			}
		}
		if analyzer.IsLodParams() {
			if b.TargetBlock() != nil {
				analyzer.GFDeclAnalysis(b.TargetBlock())
			}
			if b.UjpBlock() != nil {
				analyzer.GFDeclAnalysis(b.UjpBlock())
			}
		}
	}
	return analyzer.analysisCount
}
