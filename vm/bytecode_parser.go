package vm

import (
	"fmt"
	"github.com/goby-lang/goby/compiler/bytecode"
	"strconv"
	"strings"
)

// bytecodeParser is responsible for parsing bytecodes
type bytecodeParser struct {
	line       int
	labelTable map[labelType]map[string][]*instructionSet
	vm         *VM
	filename   filename
	blockTable map[string]*instructionSet
	program    *instructionSet
}

func (p *bytecodeParser) setLabel(is *instructionSet, name string) {
	var l *label
	var ln string
	var lt labelType

	if name == bytecode.Program {
		p.program = is
		return
	}

	ln = strings.Split(name, ":")[1]
	lt = labelType(strings.Split(name, ":")[0])

	l = &label{name: name, Type: lt}
	is.label = l

	if lt == bytecode.Block {
		p.blockTable[ln] = is
		return
	}

	p.labelTable[lt][ln] = append(p.labelTable[lt][ln], is)
}

// newBytecodeParser initializes bytecodeParser and its label table then returns it
func newBytecodeParser(file filename) *bytecodeParser {
	p := &bytecodeParser{filename: file}
	p.blockTable = make(map[string]*instructionSet)
	p.labelTable = map[labelType]map[string][]*instructionSet{
		bytecode.LabelDef:      make(map[string][]*instructionSet),
		bytecode.LabelDefClass: make(map[string][]*instructionSet),
	}

	return p
}

func (p *bytecodeParser) parseInstructionSets(sets []*bytecode.InstructionSet) []*instructionSet {
	iss := []*instructionSet{}
	count := 0

	for _, set := range sets {
		count++
		p.parseInstructionSet(iss, set)
	}

	return iss
}

func (p *bytecodeParser) parseInstructionSet(iss []*instructionSet, set *bytecode.InstructionSet) {
	is := &instructionSet{filename: p.filename}
	count := 0
	p.setLabel(is, set.LabelName())

	for _, i := range set.Instructions {
		count++
		p.convertInstruction(is, i)
	}

	iss = append(iss, is)
}

// convertInstruction transfer a bytecode.Instruction into an vm instruction and append it into given instruction set.
func (p *bytecodeParser) convertInstruction(is *instructionSet, i *bytecode.Instruction) {
	var params []interface{}
	act := operationType(i.Action)

	action := builtInActions[act]

	if action == nil {
		panic(fmt.Sprintf("Unknown command: %s. line: %d", act, i.Line()))
	} else {
		switch act {
		case bytecode.PutString:
			text := strings.Split(i.Params[0], "\"")[1]
			params = append(params, text)
		case bytecode.BranchUnless, bytecode.BranchIf, bytecode.Jump:
			line, err := i.AnchorLine()

			if err != nil {
				panic(err.Error())
			}

			params = append(params, line)
		default:
			for _, param := range i.Params {
				params = append(params, p.parseParam(param))
			}
		}
	}

	is.define(i.Line(), action, params...)
}

// parseBytecode parses given bytecodes and transfer them into a sequence of instruction set.
func (p *bytecodeParser) parseBytecode(bytecodes string) []*instructionSet {
	iss := []*instructionSet{}
	bytecodes = strings.TrimSpace(bytecodes)
	bytecodesByLine := strings.Split(bytecodes, "\n")

	defer func() {
		if p := recover(); p != nil {
			switch p.(type) {
			case errorMessage:
				return
			default:
				panic(p)
			}
		}
	}()

	p.parseBytecodeSection(iss, bytecodesByLine)

	return iss
}

func (p *bytecodeParser) parseBytecodeSection(iss []*instructionSet, bytecodesByLine []string) {
	is := &instructionSet{filename: p.filename}
	count := 0

	// First line is label
	p.parseBytecodeLabel(is, bytecodesByLine[0])

	for _, text := range bytecodesByLine[1:] {
		count++
		l := strings.TrimSpace(text)
		if strings.HasPrefix(l, "<") {
			p.parseBytecodeSection(iss, bytecodesByLine[count:])
			break
		} else {
			p.parseInstruction(is, l)
		}
	}

	iss = append(iss, is)
}

func (p *bytecodeParser) parseBytecodeLabel(is *instructionSet, line string) {
	line = strings.Trim(line, "<")
	line = strings.Trim(line, ">")
	p.setLabel(is, line)
}

// parseInstruction transfer a line of bytecode into an instruction and append it into given instruction set.
func (p *bytecodeParser) parseInstruction(is *instructionSet, line string) {
	var params []interface{}
	var rawParams []string

	tokens := strings.Split(line, " ")
	lineNum, act := tokens[0], tokens[1]
	ln, _ := strconv.ParseInt(lineNum, 0, 64)
	action := builtInActions[operationType(act)]

	if act == bytecode.PutString {
		text := strings.Split(line, "\"")[1]
		params = append(params, text)
	} else if len(tokens) > 2 {
		rawParams = tokens[2:]

		for _, param := range rawParams {
			params = append(params, p.parseParam(param))
		}
	} else if action == nil {
		panic(fmt.Sprintf("Unknown command: %s. line: %d", act, ln))
	}

	is.define(int(ln), action, params...)
}

func (p *bytecodeParser) parseParam(param string) interface{} {
	integer, e := strconv.ParseInt(param, 0, 64)
	if e != nil {
		return param
	}

	i := int(integer)

	return i
}
