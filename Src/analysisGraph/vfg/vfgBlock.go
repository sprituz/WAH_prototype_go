package vfg

import (
	"fmt"
	"strings"
)

// DU 체인의 경우 Definition에 대한 Use가 몇번 라인인지
// 즉 1번째 라인의 str 1 0 이 5 7 8번 라인에서 lod 된다면
// def 1 : { 5 , 7 , 8 } 이런식 ?
//definition에 대한 use들의 line number를 저장하는 집합

//DUSet ...
type DUSet struct {
	Definition int
	Use        []int
}

//DUChain ...
type DUChain struct {
	chain map[string][]*DUSet // offset, DUSet list
}

func (c *DUChain) Print() {
	for offset, setList := range c.chain {
		fmt.Printf(" \t* offset : %s * \n", offset)
		fmt.Println("\t--------------------------------")
		for _, set := range setList {
			fmt.Printf("\t  definition : %d Use :", set.Definition)
			fmt.Println(set.Use)
		}
		fmt.Println("\t--------------------------------\n")
	}
}
func (c *DUChain) String() string {
	builder := strings.Builder{}
	for offset, setList := range c.chain {
		builder.WriteString(fmt.Sprintf(" \t* offset : %s * \n", offset))
		builder.WriteString("\t--------------------------------\n")
		for _, set := range setList {
			builder.WriteString(fmt.Sprintf("\t  definition : %d Use :", set.Definition))
			builder.WriteString(fmt.Sprintln(set.Use))
		}
		builder.WriteString("\t--------------------------------\n")
	}

	return builder.String()
}
func (c *DUChain) Insert(offset string, set *DUSet) {
	c.chain[offset] = append(c.chain[offset], set)
}
func (c *DUChain) LookUpFull(offset string) []*DUSet {
	var res []*DUSet
	if chain, ok := c.chain[offset]; ok {
		res = chain
	}

	return res
}

func (c *DUChain) LookUpUseOfDef(offset string, def int) ([]int, bool) {
	var res []int
	isOk := false
	if chain, ok := c.chain[offset]; ok {
		for _, set := range chain {
			if set.Definition == def {
				res = set.Use
				isOk = true
				break
			}
		}
	}
	return res, isOk
}

func (c *DUChain) LookUpDefOfUse(offset string, use int) (int, bool) {
	var res int
	isOk := false
	if chain, ok := c.chain[offset]; ok {
		for _, set := range chain {
			for _, u := range set.Use {
				if u == use {
					res = set.Definition
					isOk = true
					break
				}
			}

			if isOk {
				break
			}
		}
	}
	return res, isOk
}

func (c *DUChain) Copy() DUChain {
	res := DUChain{make(map[string][]*DUSet)}

	for offset, setList := range c.chain {

		for _, set := range setList {
			duSet := &DUSet{}
			duSet.Definition = set.Definition
			for _, line := range set.Use {
				duSet.Use = append(duSet.Use, line)
			}
			res.chain[offset] = append(res.chain[offset], duSet)
		}

	}

	return res
}
