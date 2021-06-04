package wah

import (
	"go/ast"
	"go/token"
	"go/types"

	"WAH_prototype_go-master/Src/analysisGraph/cfg"
	"WAH_prototype_go-master/Src/analysisGraph/vfg"
	"WAH_prototype_go-master/Src/icg"
	"WAH_prototype_go-master/Src/icg/symbolTable"
)

type ChainCodeAnalyzer struct {
	astAnalyzer *ASTAnalyzer
	gfAnalyzer  *GFDeclAnalyzer
	uiaAnalyzer *UIAAnalyzer
	ueAnalyzer  *UEAnalyzer
	rywAnalyzer *RYWAnalyzer
	rngAnalyzer *RNGAnalyzer

	codeList            []icg.CodeInfo
	isFirstAnalysis     bool
	isFirstGraphAnalsis bool
	totalCount          int
}

func (cca *ChainCodeAnalyzer) Init(fs *token.FileSet, analysisFile string, chain vfg.DUChain, codeList []icg.CodeInfo,litTable *symbolTable.LiteralTable) {
	cca.astAnalyzer = new(ASTAnalyzer)
	cca.gfAnalyzer = new(GFDeclAnalyzer)
	cca.uiaAnalyzer = new(UIAAnalyzer)
	cca.ueAnalyzer = new(UEAnalyzer)
	cca.rywAnalyzer = new(RYWAnalyzer)
	cca.rngAnalyzer = new(RNGAnalyzer)

	cca.astAnalyzer.Init(analysisFile, fs)
	cca.gfAnalyzer.Init(analysisFile, chain, codeList)
	cca.uiaAnalyzer.Init(analysisFile, chain, codeList)
	cca.ueAnalyzer.Init(analysisFile, chain, codeList)
	cca.rywAnalyzer.Init(analysisFile, chain, codeList, litTable)
	cca.rngAnalyzer.Init(analysisFile)

	cca.codeList = codeList
	cca.isFirstAnalysis = true
	cca.isFirstGraphAnalsis = true
	cca.totalCount = 0
}
func (cca *ChainCodeAnalyzer) TotalCount() int {
	res := cca.astAnalyzer.analysisCount + cca.gfAnalyzer.analysisCount + cca.uiaAnalyzer.analysisCount + cca.ueAnalyzer.analysisCount + cca.rywAnalyzer.analysisCount + cca.rngAnalyzer.analysisCount
	return res
}
func (cca *ChainCodeAnalyzer) WeaknessAnalysis(f *ast.File, info *types.Info, block cfg.CFGBlock) {
	if cca.isFirstAnalysis {
		cca.astAnalyzer.Analysis(f, info)
		cca.isFirstAnalysis = false

		cca.ueAnalyzer.SetErrTable(cca.astAnalyzer.FuncRetTable)
	}

	switch b := block.(type) {
	case *cfg.BasicBlock:
		cca.gfAnalyzer.GFDeclAnalysis(b)
		cca.uiaAnalyzer.UIAAnalysis(b)
		cca.ueAnalyzer.UEAnalysis(b)

		if b.LinkedBlock() != nil {
			cca.WeaknessAnalysis(f, info, b.LinkedBlock())
		}
	case *cfg.BranchBlock:
		befRNGAnalyzer := cca.rngAnalyzer.Copy()
		if b.UjpBlock() != nil {

			cca.WeaknessAnalysis(f, info, b.UjpBlock())
		}

		cca.rngAnalyzer = befRNGAnalyzer
		if b.TargetBlock() != nil {
			cca.WeaknessAnalysis(f, info, b.TargetBlock())
		}
	case *cfg.CallBlock:
		cca.rywAnalyzer.RYWAnalysisUsedZ3(b)
		cca.rngAnalyzer.RNGAnalysis(b)

		if b.TargetBlock() != nil {
			cca.WeaknessAnalysis(f, info, b.TargetBlock())
		}

		if b.UjpBlock() != nil {
			cca.WeaknessAnalysis(f, info, b.UjpBlock())
		}
	case *cfg.ReturnBlock:
		if b.LinkedBlock() != nil {
			cca.WeaknessAnalysis(f, info, b.LinkedBlock())
		}
	}

	/*	if isFirstGraph {
		fmt.Println("... Graph analysis end ...")
	}*/
	//return totalCount
}
