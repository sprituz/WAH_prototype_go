package vfg

import (
	"fmt"
	"strconv"

	"WAH_prototype_go-master/Src/icg"
	"WAH_prototype_go-master/Src/analysisGraph/cfg"
)

func Generate(cfgs map[int]cfg.CFGBlock) map[int]DUChain {
	vfgGen := new(vfgGenerator)

	duchains := make(map[int]DUChain)
	for k, root := range cfgs {
		vfgGen.Init()
		duChain := vfgGen.Generate(root)
		duchains[k] = *duChain

	}

	return duchains
}

type vfgGenerator struct {
	duChain         *DUChain
	duSet           *DUSet
	curDef          map[string]int
	isLoop          bool
	addressOffset   []string
	addressBase     []int
	aliasStack      []bool
	tmpDU           DUChain
	isStructOrArray map[string]bool
}

func (g *vfgGenerator) Init() {
	g.duChain = &DUChain{make(map[string][]*DUSet)}
	g.duSet = &DUSet{}
	g.curDef = make(map[string]int)
	g.isLoop = false
	g.isStructOrArray = make(map[string]bool)
}

func (g *vfgGenerator) defGen(sil icg.CodeInfo, offset string) {
	g.curDef[offset] = sil.GetLine()
	if _, ok := g.duChain.LookUpUseOfDef(offset, sil.GetLine()); !ok {
		g.duSet = &DUSet{}
		g.duSet.Definition = sil.GetLine()

		g.duChain.Insert(offset, g.duSet)
	}
}
func (g *vfgGenerator) useGen(sil icg.CodeInfo, base int, offset string) {
	setList := g.duChain.LookUpFull(offset)
	if _, ok := g.isStructOrArray[offset]; !ok {
		existDef := false
		for _, set := range setList {
			if set.Definition == g.curDef[offset] {
				existDef = true
				isDuplicate := false
				for _, useList := range set.Use {
					if useList == sil.GetLine() {
						isDuplicate = true
					}
				}
				if !isDuplicate {
					set.Use = append(set.Use, sil.GetLine())
				}
			}
		}

		if !existDef && base == 0 {
			g.duSet = &DUSet{}
			g.curDef[offset] = -1
			g.duSet.Definition = -1
			g.duSet.Use = append(g.duSet.Use, sil.GetLine())
			g.duChain.Insert(offset, g.duSet)
		}
	} else {
		if g.isStructOrArray[offset] {
			for _, set := range setList {
				isDuplicate := false
				for _, useList := range set.Use {
					if useList == sil.GetLine() {
						isDuplicate = true
					}
				}
				if !isDuplicate {
					set.Use = append(set.Use, sil.GetLine())
				}

			}
		}
	}

}
func (g *vfgGenerator) Generate(block cfg.CFGBlock) *DUChain {
	switch b := block.(type) {
	case *cfg.BasicBlock:
		if g.isLoop {
			g.tmpDU = g.duChain.Copy()
			//g.isLoop = false
		}

		for _, sil := range b.CodeList() {
			switch sil.Opcode() {
			case icg.Str:

				strSil, _ := sil.(*icg.StackOpcode)
				offset := fmt.Sprint(strSil.Params().Front().Next().Value)
				g.defGen(sil, offset)
			case icg.Lod:
				lodSil, _ := sil.(*icg.StackOpcode)
				base, _ := strconv.Atoi(fmt.Sprint(lodSil.Params().Front().Value))

				offset := fmt.Sprint(lodSil.Params().Front().Next().Value)

				g.useGen(sil, base, offset)
			case icg.Lda:
				ldaSil := sil.(*icg.StackOpcode)
				addOffset := fmt.Sprint(ldaSil.Params().Front().Next().Value)
				g.addressOffset = append(g.addressOffset, addOffset)
				g.aliasStack = append(g.aliasStack, ldaSil.IsAlias())
				addressBase, _ := strconv.Atoi(fmt.Sprint(ldaSil.Params().Front().Value))
				g.addressBase = append(g.addressBase, addressBase)
				if !ldaSil.IsAlias() {
					g.isStructOrArray[addOffset] = true
				} else {
					g.isStructOrArray[addOffset] = false
				}

			case icg.Sti:
				addressOffset := g.addressOffset[len(g.addressOffset)-1]
				g.addressOffset = g.addressOffset[0 : len(g.addressOffset)-1]

				//addressBase := g.addressBase[len(g.addressBase)-1]
				g.addressBase = g.addressBase[0 : len(g.addressBase)-1]

				isAlias := g.aliasStack[len(g.aliasStack)-1]
				g.aliasStack = g.aliasStack[0 : len(g.aliasStack)-1]

				if !isAlias {
					g.defGen(sil, addressOffset)
				}
			case icg.Ldi:
				addressOffset := g.addressOffset[len(g.addressOffset)-1]
				g.addressOffset = g.addressOffset[0 : len(g.addressOffset)-1]

				addressBase := g.addressBase[len(g.addressBase)-1]
				g.addressBase = g.addressBase[0 : len(g.addressBase)-1]

				isAlias := g.aliasStack[len(g.aliasStack)-1]
				g.aliasStack = g.aliasStack[0 : len(g.aliasStack)-1]

				if !isAlias {

					g.useGen(sil, addressBase, addressOffset)

				}
			}

		}

		g.Generate(b.LinkedBlock())

	case *cfg.BranchBlock:

		tmpDef := make(map[string]int)
		for k, v := range g.curDef {
			tmpDef[k] = v
		}

		if b.UjpBlock() != nil {

			g.Generate(b.UjpBlock())
		}

		g.curDef = tmpDef
		if b.TargetBlock() != nil {
			target := b.TargetBlock()
			//loop인 경우
			tmp := g.isLoop
			isSame := false
			if target.BlockNumber() < b.BlockNumber() {
				if g.isLoop {
					isSame = IsSame(*g.duChain, g.tmpDU)
				} else {
					g.isLoop = true
				}
			}

			if !isSame {
				g.Generate(b.TargetBlock())
			}
			g.isLoop = tmp

		}

	case *cfg.CallBlock:

		if b.TargetBlock() != nil {
			g.Generate(b.TargetBlock())
		}

		if b.UjpBlock() != nil {
			g.Generate(b.UjpBlock())
		}
	case *cfg.ReturnBlock:
		if b.LinkedBlock() != nil {
			g.Generate(b.LinkedBlock())
		}
	}
	return g.duChain
}

func IsSame(c1 DUChain, c2 DUChain) bool {
	res := true
	if len(c1.chain) != len(c2.chain) {
		res = false
	}

	if res {
		for offset, c1SetList := range c1.chain {
			c2SetList := c2.chain[offset]
			if len(c2SetList) != len(c1SetList) {
				res = false
			}

			if res {
				for i, c1Set := range c1SetList {
					c2Set := c2SetList[i]

					if c1Set.Definition == c2Set.Definition {
						if len(c1Set.Use) != len(c2Set.Use) {
							res = false
						}

						if res {
							for j, c1Use := range c1Set.Use {
								c2Use := c2Set.Use[j]

								if c1Use != c2Use {
									res = false
								}

							}
						}
					} else {
						res = false
					}

				}
			}
		}
	}

	return res
}
