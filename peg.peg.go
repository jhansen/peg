package main

import (
	/*"bytes"*/
	"fmt"
	"math"
	"sort"
	"strconv"
)

const END_SYMBOL rune = 4

/* The rule types inferred from the grammar are below. */
type Rule uint8

const (
	RuleUnknown Rule = iota
	RuleGrammar
	RuleDefinition
	RuleExpression
	RuleSequence
	RulePrefix
	RuleSuffix
	RulePrimary
	RuleIdentifier
	RuleIdentStart
	RuleIdentCont
	RuleLiteral
	RuleClass
	RuleRanges
	RuleDoubleRanges
	RuleRange
	RuleDoubleRange
	RuleChar
	RuleDoubleChar
	RuleEscape
	RuleLeftArrow
	RuleSlash
	RuleAnd
	RuleNot
	RuleQuestion
	RuleStar
	RulePlus
	RuleOpen
	RuleClose
	RuleDot
	RuleSpacing
	RuleComment
	RuleSpace
	RuleEndOfLine
	RuleEndOfFile
	RuleAction
	RuleActionInner
	RuleBegin
	RuleEnd
	RuleAction0
	RuleAction1
	RuleAction2
	RuleAction3
	RuleAction4
	RuleAction5
	RuleAction6
	RuleAction7
	RuleAction8
	RuleAction9
	RuleAction10
	RuleAction11
	RuleAction12
	RuleAction13
	RuleAction14
	RuleAction15
	RuleAction16
	RuleAction17
	RuleAction18
	RulePegText
	RuleAction19
	RuleAction20
	RuleAction21
	RuleAction22
	RuleAction23
	RuleAction24
	RuleAction25
	RuleAction26
	RuleAction27
	RuleAction28
	RuleAction29
	RuleAction30
	RuleAction31
	RuleAction32
	RuleAction33
	RuleAction34
	RuleAction35
	RuleAction36
	RuleAction37
	RuleAction38
	RuleAction39
	RuleAction40
	RuleAction41
	RuleAction42
	RuleAction43
	RuleAction44
	RuleAction45

	RulePre_
	Rule_In_
	Rule_Suf
)

var Rul3s = [...]string{
	"Unknown",
	"Grammar",
	"Definition",
	"Expression",
	"Sequence",
	"Prefix",
	"Suffix",
	"Primary",
	"Identifier",
	"IdentStart",
	"IdentCont",
	"Literal",
	"Class",
	"Ranges",
	"DoubleRanges",
	"Range",
	"DoubleRange",
	"Char",
	"DoubleChar",
	"Escape",
	"LeftArrow",
	"Slash",
	"And",
	"Not",
	"Question",
	"Star",
	"Plus",
	"Open",
	"Close",
	"Dot",
	"Spacing",
	"Comment",
	"Space",
	"EndOfLine",
	"EndOfFile",
	"Action",
	"ActionInner",
	"Begin",
	"End",
	"Action0",
	"Action1",
	"Action2",
	"Action3",
	"Action4",
	"Action5",
	"Action6",
	"Action7",
	"Action8",
	"Action9",
	"Action10",
	"Action11",
	"Action12",
	"Action13",
	"Action14",
	"Action15",
	"Action16",
	"Action17",
	"Action18",
	"PegText",
	"Action19",
	"Action20",
	"Action21",
	"Action22",
	"Action23",
	"Action24",
	"Action25",
	"Action26",
	"Action27",
	"Action28",
	"Action29",
	"Action30",
	"Action31",
	"Action32",
	"Action33",
	"Action34",
	"Action35",
	"Action36",
	"Action37",
	"Action38",
	"Action39",
	"Action40",
	"Action41",
	"Action42",
	"Action43",
	"Action44",
	"Action45",

	"Pre_",
	"_In_",
	"_Suf",
}

type TokenTree interface {
	Print()
	PrintSyntax()
	PrintSyntaxTree(buffer string)
	Add(rule Rule, begin, end, next, depth int)
	Expand(index int) TokenTree
	Tokens() <-chan token32
	Error() []token32
	trim(length int)
}

/* ${@} bit structure for abstract syntax tree */
type token16 struct {
	Rule
	begin, end, next int16
}

func (t *token16) isZero() bool {
	return t.Rule == RuleUnknown && t.begin == 0 && t.end == 0 && t.next == 0
}

func (t *token16) isParentOf(u token16) bool {
	return t.begin <= u.begin && t.end >= u.end && t.next > u.next
}

func (t *token16) GetToken32() token32 {
	return token32{Rule: t.Rule, begin: int32(t.begin), end: int32(t.end), next: int32(t.next)}
}

func (t *token16) String() string {
	return fmt.Sprintf("\x1B[34m%v\x1B[m %v %v %v", Rul3s[t.Rule], t.begin, t.end, t.next)
}

type tokens16 struct {
	tree    []token16
	ordered [][]token16
}

func (t *tokens16) trim(length int) {
	t.tree = t.tree[0:length]
}

func (t *tokens16) Print() {
	for _, token := range t.tree {
		fmt.Println(token.String())
	}
}

func (t *tokens16) Order() [][]token16 {
	if t.ordered != nil {
		return t.ordered
	}

	depths := make([]int16, 1, math.MaxInt16)
	for i, token := range t.tree {
		if token.Rule == RuleUnknown {
			t.tree = t.tree[:i]
			break
		}
		depth := int(token.next)
		if length := len(depths); depth >= length {
			depths = depths[:depth+1]
		}
		depths[depth]++
	}
	depths = append(depths, 0)

	ordered, pool := make([][]token16, len(depths)), make([]token16, len(t.tree)+len(depths))
	for i, depth := range depths {
		depth++
		ordered[i], pool, depths[i] = pool[:depth], pool[depth:], 0
	}

	for i, token := range t.tree {
		depth := token.next
		token.next = int16(i)
		ordered[depth][depths[depth]] = token
		depths[depth]++
	}
	t.ordered = ordered
	return ordered
}

type State16 struct {
	token16
	depths []int16
	leaf   bool
}

func (t *tokens16) PreOrder() (<-chan State16, [][]token16) {
	s, ordered := make(chan State16, 6), t.Order()
	go func() {
		var states [8]State16
		for i, _ := range states {
			states[i].depths = make([]int16, len(ordered))
		}
		depths, state, depth := make([]int16, len(ordered)), 0, 1
		write := func(t token16, leaf bool) {
			S := states[state]
			state, S.Rule, S.begin, S.end, S.next, S.leaf = (state+1)%8, t.Rule, t.begin, t.end, int16(depth), leaf
			copy(S.depths, depths)
			s <- S
		}

		states[state].token16 = ordered[0][0]
		depths[0]++
		state++
		a, b := ordered[depth-1][depths[depth-1]-1], ordered[depth][depths[depth]]
	depthFirstSearch:
		for {
			for {
				if i := depths[depth]; i > 0 {
					if c, j := ordered[depth][i-1], depths[depth-1]; a.isParentOf(c) &&
						(j < 2 || !ordered[depth-1][j-2].isParentOf(c)) {
						if c.end != b.begin {
							write(token16{Rule: Rule_In_, begin: c.end, end: b.begin}, true)
						}
						break
					}
				}

				if a.begin < b.begin {
					write(token16{Rule: RulePre_, begin: a.begin, end: b.begin}, true)
				}
				break
			}

			next := depth + 1
			if c := ordered[next][depths[next]]; c.Rule != RuleUnknown && b.isParentOf(c) {
				write(b, false)
				depths[depth]++
				depth, a, b = next, b, c
				continue
			}

			write(b, true)
			depths[depth]++
			c, parent := ordered[depth][depths[depth]], true
			for {
				if c.Rule != RuleUnknown && a.isParentOf(c) {
					b = c
					continue depthFirstSearch
				} else if parent && b.end != a.end {
					write(token16{Rule: Rule_Suf, begin: b.end, end: a.end}, true)
				}

				depth--
				if depth > 0 {
					a, b, c = ordered[depth-1][depths[depth-1]-1], a, ordered[depth][depths[depth]]
					parent = a.isParentOf(b)
					continue
				}

				break depthFirstSearch
			}
		}

		close(s)
	}()
	return s, ordered
}

func (t *tokens16) PrintSyntax() {
	tokens, ordered := t.PreOrder()
	max := -1
	for token := range tokens {
		if !token.leaf {
			fmt.Printf("%v", token.begin)
			for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
				fmt.Printf(" \x1B[36m%v\x1B[m", Rul3s[ordered[i][depths[i]-1].Rule])
			}
			fmt.Printf(" \x1B[36m%v\x1B[m\n", Rul3s[token.Rule])
		} else if token.begin == token.end {
			fmt.Printf("%v", token.begin)
			for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
				fmt.Printf(" \x1B[31m%v\x1B[m", Rul3s[ordered[i][depths[i]-1].Rule])
			}
			fmt.Printf(" \x1B[31m%v\x1B[m\n", Rul3s[token.Rule])
		} else {
			for c, end := token.begin, token.end; c < end; c++ {
				if i := int(c); max+1 < i {
					for j := max; j < i; j++ {
						fmt.Printf("skip %v %v\n", j, token.String())
					}
					max = i
				} else if i := int(c); i <= max {
					for j := i; j <= max; j++ {
						fmt.Printf("dupe %v %v\n", j, token.String())
					}
				} else {
					max = int(c)
				}
				fmt.Printf("%v", c)
				for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
					fmt.Printf(" \x1B[34m%v\x1B[m", Rul3s[ordered[i][depths[i]-1].Rule])
				}
				fmt.Printf(" \x1B[34m%v\x1B[m\n", Rul3s[token.Rule])
			}
			fmt.Printf("\n")
		}
	}
}

func (t *tokens16) PrintSyntaxTree(buffer string) {
	tokens, _ := t.PreOrder()
	for token := range tokens {
		for c := 0; c < int(token.next); c++ {
			fmt.Printf(" ")
		}
		fmt.Printf("\x1B[34m%v\x1B[m %v\n", Rul3s[token.Rule], strconv.Quote(buffer[token.begin:token.end]))
	}
}

func (t *tokens16) Add(rule Rule, begin, end, depth, index int) {
	t.tree[index] = token16{Rule: rule, begin: int16(begin), end: int16(end), next: int16(depth)}
}

func (t *tokens16) Tokens() <-chan token32 {
	s := make(chan token32, 16)
	go func() {
		for _, v := range t.tree {
			s <- v.GetToken32()
		}
		close(s)
	}()
	return s
}

func (t *tokens16) Error() []token32 {
	ordered := t.Order()
	length := len(ordered)
	tokens, length := make([]token32, length), length-1
	for i, _ := range tokens {
		o := ordered[length-i]
		if len(o) > 1 {
			tokens[i] = o[len(o)-2].GetToken32()
		}
	}
	return tokens
}

/* ${@} bit structure for abstract syntax tree */
type token32 struct {
	Rule
	begin, end, next int32
}

func (t *token32) isZero() bool {
	return t.Rule == RuleUnknown && t.begin == 0 && t.end == 0 && t.next == 0
}

func (t *token32) isParentOf(u token32) bool {
	return t.begin <= u.begin && t.end >= u.end && t.next > u.next
}

func (t *token32) GetToken32() token32 {
	return token32{Rule: t.Rule, begin: int32(t.begin), end: int32(t.end), next: int32(t.next)}
}

func (t *token32) String() string {
	return fmt.Sprintf("\x1B[34m%v\x1B[m %v %v %v", Rul3s[t.Rule], t.begin, t.end, t.next)
}

type tokens32 struct {
	tree    []token32
	ordered [][]token32
}

func (t *tokens32) trim(length int) {
	t.tree = t.tree[0:length]
}

func (t *tokens32) Print() {
	for _, token := range t.tree {
		fmt.Println(token.String())
	}
}

func (t *tokens32) Order() [][]token32 {
	if t.ordered != nil {
		return t.ordered
	}

	depths := make([]int32, 1, math.MaxInt16)
	for i, token := range t.tree {
		if token.Rule == RuleUnknown {
			t.tree = t.tree[:i]
			break
		}
		depth := int(token.next)
		if length := len(depths); depth >= length {
			depths = depths[:depth+1]
		}
		depths[depth]++
	}
	depths = append(depths, 0)

	ordered, pool := make([][]token32, len(depths)), make([]token32, len(t.tree)+len(depths))
	for i, depth := range depths {
		depth++
		ordered[i], pool, depths[i] = pool[:depth], pool[depth:], 0
	}

	for i, token := range t.tree {
		depth := token.next
		token.next = int32(i)
		ordered[depth][depths[depth]] = token
		depths[depth]++
	}
	t.ordered = ordered
	return ordered
}

type State32 struct {
	token32
	depths []int32
	leaf   bool
}

func (t *tokens32) PreOrder() (<-chan State32, [][]token32) {
	s, ordered := make(chan State32, 6), t.Order()
	go func() {
		var states [8]State32
		for i, _ := range states {
			states[i].depths = make([]int32, len(ordered))
		}
		depths, state, depth := make([]int32, len(ordered)), 0, 1
		write := func(t token32, leaf bool) {
			S := states[state]
			state, S.Rule, S.begin, S.end, S.next, S.leaf = (state+1)%8, t.Rule, t.begin, t.end, int32(depth), leaf
			copy(S.depths, depths)
			s <- S
		}

		states[state].token32 = ordered[0][0]
		depths[0]++
		state++
		a, b := ordered[depth-1][depths[depth-1]-1], ordered[depth][depths[depth]]
	depthFirstSearch:
		for {
			for {
				if i := depths[depth]; i > 0 {
					if c, j := ordered[depth][i-1], depths[depth-1]; a.isParentOf(c) &&
						(j < 2 || !ordered[depth-1][j-2].isParentOf(c)) {
						if c.end != b.begin {
							write(token32{Rule: Rule_In_, begin: c.end, end: b.begin}, true)
						}
						break
					}
				}

				if a.begin < b.begin {
					write(token32{Rule: RulePre_, begin: a.begin, end: b.begin}, true)
				}
				break
			}

			next := depth + 1
			if c := ordered[next][depths[next]]; c.Rule != RuleUnknown && b.isParentOf(c) {
				write(b, false)
				depths[depth]++
				depth, a, b = next, b, c
				continue
			}

			write(b, true)
			depths[depth]++
			c, parent := ordered[depth][depths[depth]], true
			for {
				if c.Rule != RuleUnknown && a.isParentOf(c) {
					b = c
					continue depthFirstSearch
				} else if parent && b.end != a.end {
					write(token32{Rule: Rule_Suf, begin: b.end, end: a.end}, true)
				}

				depth--
				if depth > 0 {
					a, b, c = ordered[depth-1][depths[depth-1]-1], a, ordered[depth][depths[depth]]
					parent = a.isParentOf(b)
					continue
				}

				break depthFirstSearch
			}
		}

		close(s)
	}()
	return s, ordered
}

func (t *tokens32) PrintSyntax() {
	tokens, ordered := t.PreOrder()
	max := -1
	for token := range tokens {
		if !token.leaf {
			fmt.Printf("%v", token.begin)
			for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
				fmt.Printf(" \x1B[36m%v\x1B[m", Rul3s[ordered[i][depths[i]-1].Rule])
			}
			fmt.Printf(" \x1B[36m%v\x1B[m\n", Rul3s[token.Rule])
		} else if token.begin == token.end {
			fmt.Printf("%v", token.begin)
			for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
				fmt.Printf(" \x1B[31m%v\x1B[m", Rul3s[ordered[i][depths[i]-1].Rule])
			}
			fmt.Printf(" \x1B[31m%v\x1B[m\n", Rul3s[token.Rule])
		} else {
			for c, end := token.begin, token.end; c < end; c++ {
				if i := int(c); max+1 < i {
					for j := max; j < i; j++ {
						fmt.Printf("skip %v %v\n", j, token.String())
					}
					max = i
				} else if i := int(c); i <= max {
					for j := i; j <= max; j++ {
						fmt.Printf("dupe %v %v\n", j, token.String())
					}
				} else {
					max = int(c)
				}
				fmt.Printf("%v", c)
				for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
					fmt.Printf(" \x1B[34m%v\x1B[m", Rul3s[ordered[i][depths[i]-1].Rule])
				}
				fmt.Printf(" \x1B[34m%v\x1B[m\n", Rul3s[token.Rule])
			}
			fmt.Printf("\n")
		}
	}
}

func (t *tokens32) PrintSyntaxTree(buffer string) {
	tokens, _ := t.PreOrder()
	for token := range tokens {
		for c := 0; c < int(token.next); c++ {
			fmt.Printf(" ")
		}
		fmt.Printf("\x1B[34m%v\x1B[m %v\n", Rul3s[token.Rule], strconv.Quote(buffer[token.begin:token.end]))
	}
}

func (t *tokens32) Add(rule Rule, begin, end, depth, index int) {
	t.tree[index] = token32{Rule: rule, begin: int32(begin), end: int32(end), next: int32(depth)}
}

func (t *tokens32) Tokens() <-chan token32 {
	s := make(chan token32, 16)
	go func() {
		for _, v := range t.tree {
			s <- v.GetToken32()
		}
		close(s)
	}()
	return s
}

func (t *tokens32) Error() []token32 {
	ordered := t.Order()
	length := len(ordered)
	tokens, length := make([]token32, length), length-1
	for i, _ := range tokens {
		o := ordered[length-i]
		if len(o) > 1 {
			tokens[i] = o[len(o)-2].GetToken32()
		}
	}
	return tokens
}

func (t *tokens16) Expand(index int) TokenTree {
	tree := t.tree
	if index >= len(tree) {
		expanded := make([]token32, 2*len(tree))
		for i, v := range tree {
			expanded[i] = v.GetToken32()
		}
		return &tokens32{tree: expanded}
	}
	return nil
}

func (t *tokens32) Expand(index int) TokenTree {
	tree := t.tree
	if index >= len(tree) {
		expanded := make([]token32, 2*len(tree))
		copy(expanded, tree)
		t.tree = expanded
	}
	return nil
}

type Peg struct {
	*Tree

	Buffer string
	buffer []rune
	rules  [86]func() bool
	Parse  func(rule ...int) error
	Reset  func()
	TokenTree
}

type textPosition struct {
	line, symbol int
}

type textPositionMap map[int]textPosition

func translatePositions(buffer string, positions []int) textPositionMap {
	length, translations, j, line, symbol := len(positions), make(textPositionMap, len(positions)), 0, 1, 0
	sort.Ints(positions)

search:
	for i, c := range buffer[0:] {
		if c == '\n' {
			line, symbol = line+1, 0
		} else {
			symbol++
		}
		if i == positions[j] {
			translations[positions[j]] = textPosition{line, symbol}
			for j++; j < length; j++ {
				if i != positions[j] {
					continue search
				}
			}
			break search
		}
	}

	return translations
}

type parseError struct {
	p *Peg
}

func (e *parseError) Error() string {
	tokens, error := e.p.TokenTree.Error(), "\n"
	positions, p := make([]int, 2*len(tokens)), 0
	for _, token := range tokens {
		positions[p], p = int(token.begin), p+1
		positions[p], p = int(token.end), p+1
	}
	translations := translatePositions(e.p.Buffer, positions)
	for _, token := range tokens {
		begin, end := int(token.begin), int(token.end)
		error += fmt.Sprintf("parse error near \x1B[34m%v\x1B[m (line %v symbol %v - line %v symbol %v):\n%v\n",
			Rul3s[token.Rule],
			translations[begin].line, translations[begin].symbol,
			translations[end].line, translations[end].symbol,
			/*strconv.Quote(*/ e.p.Buffer[begin:end] /*)*/)
	}

	return error
}

func (p *Peg) PrintSyntaxTree() {
	p.TokenTree.PrintSyntaxTree(p.Buffer)
}

func (p *Peg) Highlighter() {
	p.TokenTree.PrintSyntax()
}

func (p *Peg) Execute() {
	buffer, begin, end := p.Buffer, 0, 0
	for token := range p.TokenTree.Tokens() {
		switch token.Rule {
		case RulePegText:
			begin, end = int(token.begin), int(token.end)
		case RuleAction0:
			p.AddPackage(buffer[begin:end])
		case RuleAction1:
			p.AddPeg(buffer[begin:end])
		case RuleAction2:
			p.AddState(buffer[begin:end])
		case RuleAction3:
			p.AddRule(buffer[begin:end])
		case RuleAction4:
			p.AddExpression()
		case RuleAction5:
			p.AddAlternate()
		case RuleAction6:
			p.AddNil()
			p.AddAlternate()
		case RuleAction7:
			p.AddNil()
		case RuleAction8:
			p.AddSequence()
		case RuleAction9:
			p.AddPredicate(buffer[begin:end])
		case RuleAction10:
			p.AddPeekFor()
		case RuleAction11:
			p.AddPeekNot()
		case RuleAction12:
			p.AddQuery()
		case RuleAction13:
			p.AddStar()
		case RuleAction14:
			p.AddPlus()
		case RuleAction15:
			p.AddName(buffer[begin:end])
		case RuleAction16:
			p.AddDot()
		case RuleAction17:
			p.AddAction(buffer[begin:end])
		case RuleAction18:
			p.AddPush()
		case RuleAction19:
			p.AddSequence()
		case RuleAction20:
			p.AddSequence()
		case RuleAction21:
			p.AddPeekNot()
			p.AddDot()
			p.AddSequence()
		case RuleAction22:
			p.AddPeekNot()
			p.AddDot()
			p.AddSequence()
		case RuleAction23:
			p.AddAlternate()
		case RuleAction24:
			p.AddAlternate()
		case RuleAction25:
			p.AddRange()
		case RuleAction26:
			p.AddDoubleRange()
		case RuleAction27:
			p.AddCharacter(buffer[begin:end])
		case RuleAction28:
			p.AddDoubleCharacter(buffer[begin:end])
		case RuleAction29:
			p.AddCharacter(buffer[begin:end])
		case RuleAction30:
			p.AddCharacter("\a")
		case RuleAction31:
			p.AddCharacter("\b")
		case RuleAction32:
			p.AddCharacter("\x1B")
		case RuleAction33:
			p.AddCharacter("\f")
		case RuleAction34:
			p.AddCharacter("\n")
		case RuleAction35:
			p.AddCharacter("\r")
		case RuleAction36:
			p.AddCharacter("\t")
		case RuleAction37:
			p.AddCharacter("\v")
		case RuleAction38:
			p.AddCharacter("'")
		case RuleAction39:
			p.AddCharacter("\"")
		case RuleAction40:
			p.AddCharacter("[")
		case RuleAction41:
			p.AddCharacter("]")
		case RuleAction42:
			p.AddCharacter("-")
		case RuleAction43:
			p.AddOctalCharacter(buffer[begin:end])
		case RuleAction44:
			p.AddOctalCharacter(buffer[begin:end])
		case RuleAction45:
			p.AddCharacter("\\")

		}
	}
}

func (p *Peg) Init() {
	p.buffer = []rune(p.Buffer)
	if len(p.buffer) == 0 || p.buffer[len(p.buffer)-1] != END_SYMBOL {
		p.buffer = append(p.buffer, END_SYMBOL)
	}

	var tree TokenTree = &tokens16{tree: make([]token16, math.MaxInt16)}
	position, depth, tokenIndex, buffer, rules := 0, 0, 0, p.buffer, p.rules

	p.Parse = func(rule ...int) error {
		r := 1
		if len(rule) > 0 {
			r = rule[0]
		}
		matches := p.rules[r]()
		p.TokenTree = tree
		if matches {
			p.TokenTree.trim(tokenIndex)
			return nil
		}
		return &parseError{p}
	}

	p.Reset = func() {
		position, tokenIndex, depth = 0, 0, 0
	}

	add := func(rule Rule, begin int) {
		if t := tree.Expand(tokenIndex); t != nil {
			tree = t
		}
		tree.Add(rule, begin, position, depth, tokenIndex)
		tokenIndex++
	}

	matchDot := func() bool {
		if buffer[position] != END_SYMBOL {
			position++
			return true
		}
		return false
	}

	/*matchChar := func(c byte) bool {
		if buffer[position] == c {
			position++
			return true
		}
		return false
	}*/

	/*matchRange := func(lower byte, upper byte) bool {
		if c := buffer[position]; c >= lower && c <= upper {
			position++
			return true
		}
		return false
	}*/

	rules = [...]func() bool{
		nil,
		/* 0 Grammar <- <(Spacing ('p' 'a' 'c' 'k' 'a' 'g' 'e') Spacing Identifier Action0 ('t' 'y' 'p' 'e') Spacing Identifier Action1 ('P' 'e' 'g') Spacing Action Action2 Definition+ EndOfFile)> */
		func() bool {
			position0, tokenIndex0, depth0 := position, tokenIndex, depth
			{
				position1 := position
				depth++
				if !rules[RuleSpacing]() {
					goto l0
				}
				if buffer[position] != rune('p') {
					goto l0
				}
				position++
				if buffer[position] != rune('a') {
					goto l0
				}
				position++
				if buffer[position] != rune('c') {
					goto l0
				}
				position++
				if buffer[position] != rune('k') {
					goto l0
				}
				position++
				if buffer[position] != rune('a') {
					goto l0
				}
				position++
				if buffer[position] != rune('g') {
					goto l0
				}
				position++
				if buffer[position] != rune('e') {
					goto l0
				}
				position++
				if !rules[RuleSpacing]() {
					goto l0
				}
				if !rules[RuleIdentifier]() {
					goto l0
				}
				if !rules[RuleAction0]() {
					goto l0
				}
				if buffer[position] != rune('t') {
					goto l0
				}
				position++
				if buffer[position] != rune('y') {
					goto l0
				}
				position++
				if buffer[position] != rune('p') {
					goto l0
				}
				position++
				if buffer[position] != rune('e') {
					goto l0
				}
				position++
				if !rules[RuleSpacing]() {
					goto l0
				}
				if !rules[RuleIdentifier]() {
					goto l0
				}
				if !rules[RuleAction1]() {
					goto l0
				}
				if buffer[position] != rune('P') {
					goto l0
				}
				position++
				if buffer[position] != rune('e') {
					goto l0
				}
				position++
				if buffer[position] != rune('g') {
					goto l0
				}
				position++
				if !rules[RuleSpacing]() {
					goto l0
				}
				if !rules[RuleAction]() {
					goto l0
				}
				if !rules[RuleAction2]() {
					goto l0
				}
				if !rules[RuleDefinition]() {
					goto l0
				}
			l2:
				{
					position3, tokenIndex3, depth3 := position, tokenIndex, depth
					if !rules[RuleDefinition]() {
						goto l3
					}
					goto l2
				l3:
					position, tokenIndex, depth = position3, tokenIndex3, depth3
				}
				if !rules[RuleEndOfFile]() {
					goto l0
				}
				depth--
				add(RuleGrammar, position1)
			}
			return true
		l0:
			position, tokenIndex, depth = position0, tokenIndex0, depth0
			return false
		},
		/* 1 Definition <- <(Identifier Action3 LeftArrow Expression Action4 &((Identifier LeftArrow) / !.))> */
		func() bool {
			position4, tokenIndex4, depth4 := position, tokenIndex, depth
			{
				position5 := position
				depth++
				if !rules[RuleIdentifier]() {
					goto l4
				}
				if !rules[RuleAction3]() {
					goto l4
				}
				if !rules[RuleLeftArrow]() {
					goto l4
				}
				if !rules[RuleExpression]() {
					goto l4
				}
				if !rules[RuleAction4]() {
					goto l4
				}
				{
					position6, tokenIndex6, depth6 := position, tokenIndex, depth
					{
						position7, tokenIndex7, depth7 := position, tokenIndex, depth
						if !rules[RuleIdentifier]() {
							goto l8
						}
						if !rules[RuleLeftArrow]() {
							goto l8
						}
						goto l7
					l8:
						position, tokenIndex, depth = position7, tokenIndex7, depth7
						{
							position9, tokenIndex9, depth9 := position, tokenIndex, depth
							if !matchDot() {
								goto l9
							}
							goto l4
						l9:
							position, tokenIndex, depth = position9, tokenIndex9, depth9
						}
					}
				l7:
					position, tokenIndex, depth = position6, tokenIndex6, depth6
				}
				depth--
				add(RuleDefinition, position5)
			}
			return true
		l4:
			position, tokenIndex, depth = position4, tokenIndex4, depth4
			return false
		},
		/* 2 Expression <- <((Sequence (Slash Sequence Action5)* (Slash Action6)?) / Action7)> */
		func() bool {
			position10, tokenIndex10, depth10 := position, tokenIndex, depth
			{
				position11 := position
				depth++
				{
					position12, tokenIndex12, depth12 := position, tokenIndex, depth
					if !rules[RuleSequence]() {
						goto l13
					}
				l14:
					{
						position15, tokenIndex15, depth15 := position, tokenIndex, depth
						if !rules[RuleSlash]() {
							goto l15
						}
						if !rules[RuleSequence]() {
							goto l15
						}
						if !rules[RuleAction5]() {
							goto l15
						}
						goto l14
					l15:
						position, tokenIndex, depth = position15, tokenIndex15, depth15
					}
					{
						position16, tokenIndex16, depth16 := position, tokenIndex, depth
						if !rules[RuleSlash]() {
							goto l16
						}
						if !rules[RuleAction6]() {
							goto l16
						}
						goto l17
					l16:
						position, tokenIndex, depth = position16, tokenIndex16, depth16
					}
				l17:
					goto l12
				l13:
					position, tokenIndex, depth = position12, tokenIndex12, depth12
					if !rules[RuleAction7]() {
						goto l10
					}
				}
			l12:
				depth--
				add(RuleExpression, position11)
			}
			return true
		l10:
			position, tokenIndex, depth = position10, tokenIndex10, depth10
			return false
		},
		/* 3 Sequence <- <(Prefix (Prefix Action8)*)> */
		func() bool {
			position18, tokenIndex18, depth18 := position, tokenIndex, depth
			{
				position19 := position
				depth++
				if !rules[RulePrefix]() {
					goto l18
				}
			l20:
				{
					position21, tokenIndex21, depth21 := position, tokenIndex, depth
					if !rules[RulePrefix]() {
						goto l21
					}
					if !rules[RuleAction8]() {
						goto l21
					}
					goto l20
				l21:
					position, tokenIndex, depth = position21, tokenIndex21, depth21
				}
				depth--
				add(RuleSequence, position19)
			}
			return true
		l18:
			position, tokenIndex, depth = position18, tokenIndex18, depth18
			return false
		},
		/* 4 Prefix <- <((And Action Action9) / (And Suffix Action10) / (Not Suffix Action11) / Suffix)> */
		func() bool {
			position22, tokenIndex22, depth22 := position, tokenIndex, depth
			{
				position23 := position
				depth++
				{
					position24, tokenIndex24, depth24 := position, tokenIndex, depth
					if !rules[RuleAnd]() {
						goto l25
					}
					if !rules[RuleAction]() {
						goto l25
					}
					if !rules[RuleAction9]() {
						goto l25
					}
					goto l24
				l25:
					position, tokenIndex, depth = position24, tokenIndex24, depth24
					if !rules[RuleAnd]() {
						goto l26
					}
					if !rules[RuleSuffix]() {
						goto l26
					}
					if !rules[RuleAction10]() {
						goto l26
					}
					goto l24
				l26:
					position, tokenIndex, depth = position24, tokenIndex24, depth24
					if !rules[RuleNot]() {
						goto l27
					}
					if !rules[RuleSuffix]() {
						goto l27
					}
					if !rules[RuleAction11]() {
						goto l27
					}
					goto l24
				l27:
					position, tokenIndex, depth = position24, tokenIndex24, depth24
					if !rules[RuleSuffix]() {
						goto l22
					}
				}
			l24:
				depth--
				add(RulePrefix, position23)
			}
			return true
		l22:
			position, tokenIndex, depth = position22, tokenIndex22, depth22
			return false
		},
		/* 5 Suffix <- <(Primary ((Question Action12) / (Star Action13) / (Plus Action14))?)> */
		func() bool {
			position28, tokenIndex28, depth28 := position, tokenIndex, depth
			{
				position29 := position
				depth++
				if !rules[RulePrimary]() {
					goto l28
				}
				{
					position30, tokenIndex30, depth30 := position, tokenIndex, depth
					{
						position32, tokenIndex32, depth32 := position, tokenIndex, depth
						if !rules[RuleQuestion]() {
							goto l33
						}
						if !rules[RuleAction12]() {
							goto l33
						}
						goto l32
					l33:
						position, tokenIndex, depth = position32, tokenIndex32, depth32
						if !rules[RuleStar]() {
							goto l34
						}
						if !rules[RuleAction13]() {
							goto l34
						}
						goto l32
					l34:
						position, tokenIndex, depth = position32, tokenIndex32, depth32
						if !rules[RulePlus]() {
							goto l30
						}
						if !rules[RuleAction14]() {
							goto l30
						}
					}
				l32:
					goto l31
				l30:
					position, tokenIndex, depth = position30, tokenIndex30, depth30
				}
			l31:
				depth--
				add(RuleSuffix, position29)
			}
			return true
		l28:
			position, tokenIndex, depth = position28, tokenIndex28, depth28
			return false
		},
		/* 6 Primary <- <((Identifier !LeftArrow Action15) / (Open Expression Close) / Literal / Class / (Dot Action16) / (Action Action17) / (Begin Expression End Action18))> */
		func() bool {
			position35, tokenIndex35, depth35 := position, tokenIndex, depth
			{
				position36 := position
				depth++
				{
					position37, tokenIndex37, depth37 := position, tokenIndex, depth
					if !rules[RuleIdentifier]() {
						goto l38
					}
					{
						position39, tokenIndex39, depth39 := position, tokenIndex, depth
						if !rules[RuleLeftArrow]() {
							goto l39
						}
						goto l38
					l39:
						position, tokenIndex, depth = position39, tokenIndex39, depth39
					}
					if !rules[RuleAction15]() {
						goto l38
					}
					goto l37
				l38:
					position, tokenIndex, depth = position37, tokenIndex37, depth37
					if !rules[RuleOpen]() {
						goto l40
					}
					if !rules[RuleExpression]() {
						goto l40
					}
					if !rules[RuleClose]() {
						goto l40
					}
					goto l37
				l40:
					position, tokenIndex, depth = position37, tokenIndex37, depth37
					if !rules[RuleLiteral]() {
						goto l41
					}
					goto l37
				l41:
					position, tokenIndex, depth = position37, tokenIndex37, depth37
					if !rules[RuleClass]() {
						goto l42
					}
					goto l37
				l42:
					position, tokenIndex, depth = position37, tokenIndex37, depth37
					if !rules[RuleDot]() {
						goto l43
					}
					if !rules[RuleAction16]() {
						goto l43
					}
					goto l37
				l43:
					position, tokenIndex, depth = position37, tokenIndex37, depth37
					if !rules[RuleAction]() {
						goto l44
					}
					if !rules[RuleAction17]() {
						goto l44
					}
					goto l37
				l44:
					position, tokenIndex, depth = position37, tokenIndex37, depth37
					if !rules[RuleBegin]() {
						goto l35
					}
					if !rules[RuleExpression]() {
						goto l35
					}
					if !rules[RuleEnd]() {
						goto l35
					}
					if !rules[RuleAction18]() {
						goto l35
					}
				}
			l37:
				depth--
				add(RulePrimary, position36)
			}
			return true
		l35:
			position, tokenIndex, depth = position35, tokenIndex35, depth35
			return false
		},
		/* 7 Identifier <- <(<(IdentStart IdentCont*)> Spacing)> */
		func() bool {
			position45, tokenIndex45, depth45 := position, tokenIndex, depth
			{
				position46 := position
				depth++
				{
					position47 := position
					depth++
					if !rules[RuleIdentStart]() {
						goto l45
					}
				l48:
					{
						position49, tokenIndex49, depth49 := position, tokenIndex, depth
						if !rules[RuleIdentCont]() {
							goto l49
						}
						goto l48
					l49:
						position, tokenIndex, depth = position49, tokenIndex49, depth49
					}
					depth--
					add(RulePegText, position47)
				}
				if !rules[RuleSpacing]() {
					goto l45
				}
				depth--
				add(RuleIdentifier, position46)
			}
			return true
		l45:
			position, tokenIndex, depth = position45, tokenIndex45, depth45
			return false
		},
		/* 8 IdentStart <- <([a-z] / [A-Z] / '_')> */
		func() bool {
			position50, tokenIndex50, depth50 := position, tokenIndex, depth
			{
				position51 := position
				depth++
				{
					position52, tokenIndex52, depth52 := position, tokenIndex, depth
					if c := buffer[position]; c < rune('a') || c > rune('z') {
						goto l53
					}
					position++
					goto l52
				l53:
					position, tokenIndex, depth = position52, tokenIndex52, depth52
					if c := buffer[position]; c < rune('A') || c > rune('Z') {
						goto l54
					}
					position++
					goto l52
				l54:
					position, tokenIndex, depth = position52, tokenIndex52, depth52
					if buffer[position] != rune('_') {
						goto l50
					}
					position++
				}
			l52:
				depth--
				add(RuleIdentStart, position51)
			}
			return true
		l50:
			position, tokenIndex, depth = position50, tokenIndex50, depth50
			return false
		},
		/* 9 IdentCont <- <(IdentStart / [0-9])> */
		func() bool {
			position55, tokenIndex55, depth55 := position, tokenIndex, depth
			{
				position56 := position
				depth++
				{
					position57, tokenIndex57, depth57 := position, tokenIndex, depth
					if !rules[RuleIdentStart]() {
						goto l58
					}
					goto l57
				l58:
					position, tokenIndex, depth = position57, tokenIndex57, depth57
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l55
					}
					position++
				}
			l57:
				depth--
				add(RuleIdentCont, position56)
			}
			return true
		l55:
			position, tokenIndex, depth = position55, tokenIndex55, depth55
			return false
		},
		/* 10 Literal <- <(('\'' (!'\'' Char)? (!'\'' Char Action19)* '\'' Spacing) / ('"' (!'"' DoubleChar)? (!'"' DoubleChar Action20)* '"' Spacing))> */
		func() bool {
			position59, tokenIndex59, depth59 := position, tokenIndex, depth
			{
				position60 := position
				depth++
				{
					position61, tokenIndex61, depth61 := position, tokenIndex, depth
					if buffer[position] != rune('\'') {
						goto l62
					}
					position++
					{
						position63, tokenIndex63, depth63 := position, tokenIndex, depth
						{
							position65, tokenIndex65, depth65 := position, tokenIndex, depth
							if buffer[position] != rune('\'') {
								goto l65
							}
							position++
							goto l63
						l65:
							position, tokenIndex, depth = position65, tokenIndex65, depth65
						}
						if !rules[RuleChar]() {
							goto l63
						}
						goto l64
					l63:
						position, tokenIndex, depth = position63, tokenIndex63, depth63
					}
				l64:
				l66:
					{
						position67, tokenIndex67, depth67 := position, tokenIndex, depth
						{
							position68, tokenIndex68, depth68 := position, tokenIndex, depth
							if buffer[position] != rune('\'') {
								goto l68
							}
							position++
							goto l67
						l68:
							position, tokenIndex, depth = position68, tokenIndex68, depth68
						}
						if !rules[RuleChar]() {
							goto l67
						}
						if !rules[RuleAction19]() {
							goto l67
						}
						goto l66
					l67:
						position, tokenIndex, depth = position67, tokenIndex67, depth67
					}
					if buffer[position] != rune('\'') {
						goto l62
					}
					position++
					if !rules[RuleSpacing]() {
						goto l62
					}
					goto l61
				l62:
					position, tokenIndex, depth = position61, tokenIndex61, depth61
					if buffer[position] != rune('"') {
						goto l59
					}
					position++
					{
						position69, tokenIndex69, depth69 := position, tokenIndex, depth
						{
							position71, tokenIndex71, depth71 := position, tokenIndex, depth
							if buffer[position] != rune('"') {
								goto l71
							}
							position++
							goto l69
						l71:
							position, tokenIndex, depth = position71, tokenIndex71, depth71
						}
						if !rules[RuleDoubleChar]() {
							goto l69
						}
						goto l70
					l69:
						position, tokenIndex, depth = position69, tokenIndex69, depth69
					}
				l70:
				l72:
					{
						position73, tokenIndex73, depth73 := position, tokenIndex, depth
						{
							position74, tokenIndex74, depth74 := position, tokenIndex, depth
							if buffer[position] != rune('"') {
								goto l74
							}
							position++
							goto l73
						l74:
							position, tokenIndex, depth = position74, tokenIndex74, depth74
						}
						if !rules[RuleDoubleChar]() {
							goto l73
						}
						if !rules[RuleAction20]() {
							goto l73
						}
						goto l72
					l73:
						position, tokenIndex, depth = position73, tokenIndex73, depth73
					}
					if buffer[position] != rune('"') {
						goto l59
					}
					position++
					if !rules[RuleSpacing]() {
						goto l59
					}
				}
			l61:
				depth--
				add(RuleLiteral, position60)
			}
			return true
		l59:
			position, tokenIndex, depth = position59, tokenIndex59, depth59
			return false
		},
		/* 11 Class <- <((('[' '[' (('^' DoubleRanges Action21) / DoubleRanges)? (']' ']')) / ('[' (('^' Ranges Action22) / Ranges)? ']')) Spacing)> */
		func() bool {
			position75, tokenIndex75, depth75 := position, tokenIndex, depth
			{
				position76 := position
				depth++
				{
					position77, tokenIndex77, depth77 := position, tokenIndex, depth
					if buffer[position] != rune('[') {
						goto l78
					}
					position++
					if buffer[position] != rune('[') {
						goto l78
					}
					position++
					{
						position79, tokenIndex79, depth79 := position, tokenIndex, depth
						{
							position81, tokenIndex81, depth81 := position, tokenIndex, depth
							if buffer[position] != rune('^') {
								goto l82
							}
							position++
							if !rules[RuleDoubleRanges]() {
								goto l82
							}
							if !rules[RuleAction21]() {
								goto l82
							}
							goto l81
						l82:
							position, tokenIndex, depth = position81, tokenIndex81, depth81
							if !rules[RuleDoubleRanges]() {
								goto l79
							}
						}
					l81:
						goto l80
					l79:
						position, tokenIndex, depth = position79, tokenIndex79, depth79
					}
				l80:
					if buffer[position] != rune(']') {
						goto l78
					}
					position++
					if buffer[position] != rune(']') {
						goto l78
					}
					position++
					goto l77
				l78:
					position, tokenIndex, depth = position77, tokenIndex77, depth77
					if buffer[position] != rune('[') {
						goto l75
					}
					position++
					{
						position83, tokenIndex83, depth83 := position, tokenIndex, depth
						{
							position85, tokenIndex85, depth85 := position, tokenIndex, depth
							if buffer[position] != rune('^') {
								goto l86
							}
							position++
							if !rules[RuleRanges]() {
								goto l86
							}
							if !rules[RuleAction22]() {
								goto l86
							}
							goto l85
						l86:
							position, tokenIndex, depth = position85, tokenIndex85, depth85
							if !rules[RuleRanges]() {
								goto l83
							}
						}
					l85:
						goto l84
					l83:
						position, tokenIndex, depth = position83, tokenIndex83, depth83
					}
				l84:
					if buffer[position] != rune(']') {
						goto l75
					}
					position++
				}
			l77:
				if !rules[RuleSpacing]() {
					goto l75
				}
				depth--
				add(RuleClass, position76)
			}
			return true
		l75:
			position, tokenIndex, depth = position75, tokenIndex75, depth75
			return false
		},
		/* 12 Ranges <- <(!']' Range (!']' Range Action23)*)> */
		func() bool {
			position87, tokenIndex87, depth87 := position, tokenIndex, depth
			{
				position88 := position
				depth++
				{
					position89, tokenIndex89, depth89 := position, tokenIndex, depth
					if buffer[position] != rune(']') {
						goto l89
					}
					position++
					goto l87
				l89:
					position, tokenIndex, depth = position89, tokenIndex89, depth89
				}
				if !rules[RuleRange]() {
					goto l87
				}
			l90:
				{
					position91, tokenIndex91, depth91 := position, tokenIndex, depth
					{
						position92, tokenIndex92, depth92 := position, tokenIndex, depth
						if buffer[position] != rune(']') {
							goto l92
						}
						position++
						goto l91
					l92:
						position, tokenIndex, depth = position92, tokenIndex92, depth92
					}
					if !rules[RuleRange]() {
						goto l91
					}
					if !rules[RuleAction23]() {
						goto l91
					}
					goto l90
				l91:
					position, tokenIndex, depth = position91, tokenIndex91, depth91
				}
				depth--
				add(RuleRanges, position88)
			}
			return true
		l87:
			position, tokenIndex, depth = position87, tokenIndex87, depth87
			return false
		},
		/* 13 DoubleRanges <- <(!(']' ']') DoubleRange (!(']' ']') DoubleRange Action24)*)> */
		func() bool {
			position93, tokenIndex93, depth93 := position, tokenIndex, depth
			{
				position94 := position
				depth++
				{
					position95, tokenIndex95, depth95 := position, tokenIndex, depth
					if buffer[position] != rune(']') {
						goto l95
					}
					position++
					if buffer[position] != rune(']') {
						goto l95
					}
					position++
					goto l93
				l95:
					position, tokenIndex, depth = position95, tokenIndex95, depth95
				}
				if !rules[RuleDoubleRange]() {
					goto l93
				}
			l96:
				{
					position97, tokenIndex97, depth97 := position, tokenIndex, depth
					{
						position98, tokenIndex98, depth98 := position, tokenIndex, depth
						if buffer[position] != rune(']') {
							goto l98
						}
						position++
						if buffer[position] != rune(']') {
							goto l98
						}
						position++
						goto l97
					l98:
						position, tokenIndex, depth = position98, tokenIndex98, depth98
					}
					if !rules[RuleDoubleRange]() {
						goto l97
					}
					if !rules[RuleAction24]() {
						goto l97
					}
					goto l96
				l97:
					position, tokenIndex, depth = position97, tokenIndex97, depth97
				}
				depth--
				add(RuleDoubleRanges, position94)
			}
			return true
		l93:
			position, tokenIndex, depth = position93, tokenIndex93, depth93
			return false
		},
		/* 14 Range <- <((Char '-' Char Action25) / Char)> */
		func() bool {
			position99, tokenIndex99, depth99 := position, tokenIndex, depth
			{
				position100 := position
				depth++
				{
					position101, tokenIndex101, depth101 := position, tokenIndex, depth
					if !rules[RuleChar]() {
						goto l102
					}
					if buffer[position] != rune('-') {
						goto l102
					}
					position++
					if !rules[RuleChar]() {
						goto l102
					}
					if !rules[RuleAction25]() {
						goto l102
					}
					goto l101
				l102:
					position, tokenIndex, depth = position101, tokenIndex101, depth101
					if !rules[RuleChar]() {
						goto l99
					}
				}
			l101:
				depth--
				add(RuleRange, position100)
			}
			return true
		l99:
			position, tokenIndex, depth = position99, tokenIndex99, depth99
			return false
		},
		/* 15 DoubleRange <- <((Char '-' Char Action26) / DoubleChar)> */
		func() bool {
			position103, tokenIndex103, depth103 := position, tokenIndex, depth
			{
				position104 := position
				depth++
				{
					position105, tokenIndex105, depth105 := position, tokenIndex, depth
					if !rules[RuleChar]() {
						goto l106
					}
					if buffer[position] != rune('-') {
						goto l106
					}
					position++
					if !rules[RuleChar]() {
						goto l106
					}
					if !rules[RuleAction26]() {
						goto l106
					}
					goto l105
				l106:
					position, tokenIndex, depth = position105, tokenIndex105, depth105
					if !rules[RuleDoubleChar]() {
						goto l103
					}
				}
			l105:
				depth--
				add(RuleDoubleRange, position104)
			}
			return true
		l103:
			position, tokenIndex, depth = position103, tokenIndex103, depth103
			return false
		},
		/* 16 Char <- <(Escape / (!'\\' <.> Action27))> */
		func() bool {
			position107, tokenIndex107, depth107 := position, tokenIndex, depth
			{
				position108 := position
				depth++
				{
					position109, tokenIndex109, depth109 := position, tokenIndex, depth
					if !rules[RuleEscape]() {
						goto l110
					}
					goto l109
				l110:
					position, tokenIndex, depth = position109, tokenIndex109, depth109
					{
						position111, tokenIndex111, depth111 := position, tokenIndex, depth
						if buffer[position] != rune('\\') {
							goto l111
						}
						position++
						goto l107
					l111:
						position, tokenIndex, depth = position111, tokenIndex111, depth111
					}
					{
						position112 := position
						depth++
						if !matchDot() {
							goto l107
						}
						depth--
						add(RulePegText, position112)
					}
					if !rules[RuleAction27]() {
						goto l107
					}
				}
			l109:
				depth--
				add(RuleChar, position108)
			}
			return true
		l107:
			position, tokenIndex, depth = position107, tokenIndex107, depth107
			return false
		},
		/* 17 DoubleChar <- <(Escape / (<([a-z] / [A-Z])> Action28) / (!'\\' <.> Action29))> */
		func() bool {
			position113, tokenIndex113, depth113 := position, tokenIndex, depth
			{
				position114 := position
				depth++
				{
					position115, tokenIndex115, depth115 := position, tokenIndex, depth
					if !rules[RuleEscape]() {
						goto l116
					}
					goto l115
				l116:
					position, tokenIndex, depth = position115, tokenIndex115, depth115
					{
						position118 := position
						depth++
						{
							position119, tokenIndex119, depth119 := position, tokenIndex, depth
							if c := buffer[position]; c < rune('a') || c > rune('z') {
								goto l120
							}
							position++
							goto l119
						l120:
							position, tokenIndex, depth = position119, tokenIndex119, depth119
							if c := buffer[position]; c < rune('A') || c > rune('Z') {
								goto l117
							}
							position++
						}
					l119:
						depth--
						add(RulePegText, position118)
					}
					if !rules[RuleAction28]() {
						goto l117
					}
					goto l115
				l117:
					position, tokenIndex, depth = position115, tokenIndex115, depth115
					{
						position121, tokenIndex121, depth121 := position, tokenIndex, depth
						if buffer[position] != rune('\\') {
							goto l121
						}
						position++
						goto l113
					l121:
						position, tokenIndex, depth = position121, tokenIndex121, depth121
					}
					{
						position122 := position
						depth++
						if !matchDot() {
							goto l113
						}
						depth--
						add(RulePegText, position122)
					}
					if !rules[RuleAction29]() {
						goto l113
					}
				}
			l115:
				depth--
				add(RuleDoubleChar, position114)
			}
			return true
		l113:
			position, tokenIndex, depth = position113, tokenIndex113, depth113
			return false
		},
		/* 18 Escape <- <(('\\' ('a' / 'A') Action30) / ('\\' ('b' / 'B') Action31) / ('\\' ('e' / 'E') Action32) / ('\\' ('f' / 'F') Action33) / ('\\' ('n' / 'N') Action34) / ('\\' ('r' / 'R') Action35) / ('\\' ('t' / 'T') Action36) / ('\\' ('v' / 'V') Action37) / ('\\' '\'' Action38) / ('\\' '"' Action39) / ('\\' '[' Action40) / ('\\' ']' Action41) / ('\\' '-' Action42) / ('\\' <([0-3] [0-7] [0-7])> Action43) / ('\\' <([0-7] [0-7]?)> Action44) / ('\\' '\\' Action45))> */
		func() bool {
			position123, tokenIndex123, depth123 := position, tokenIndex, depth
			{
				position124 := position
				depth++
				{
					position125, tokenIndex125, depth125 := position, tokenIndex, depth
					if buffer[position] != rune('\\') {
						goto l126
					}
					position++
					{
						position127, tokenIndex127, depth127 := position, tokenIndex, depth
						if buffer[position] != rune('a') {
							goto l128
						}
						position++
						goto l127
					l128:
						position, tokenIndex, depth = position127, tokenIndex127, depth127
						if buffer[position] != rune('A') {
							goto l126
						}
						position++
					}
				l127:
					if !rules[RuleAction30]() {
						goto l126
					}
					goto l125
				l126:
					position, tokenIndex, depth = position125, tokenIndex125, depth125
					if buffer[position] != rune('\\') {
						goto l129
					}
					position++
					{
						position130, tokenIndex130, depth130 := position, tokenIndex, depth
						if buffer[position] != rune('b') {
							goto l131
						}
						position++
						goto l130
					l131:
						position, tokenIndex, depth = position130, tokenIndex130, depth130
						if buffer[position] != rune('B') {
							goto l129
						}
						position++
					}
				l130:
					if !rules[RuleAction31]() {
						goto l129
					}
					goto l125
				l129:
					position, tokenIndex, depth = position125, tokenIndex125, depth125
					if buffer[position] != rune('\\') {
						goto l132
					}
					position++
					{
						position133, tokenIndex133, depth133 := position, tokenIndex, depth
						if buffer[position] != rune('e') {
							goto l134
						}
						position++
						goto l133
					l134:
						position, tokenIndex, depth = position133, tokenIndex133, depth133
						if buffer[position] != rune('E') {
							goto l132
						}
						position++
					}
				l133:
					if !rules[RuleAction32]() {
						goto l132
					}
					goto l125
				l132:
					position, tokenIndex, depth = position125, tokenIndex125, depth125
					if buffer[position] != rune('\\') {
						goto l135
					}
					position++
					{
						position136, tokenIndex136, depth136 := position, tokenIndex, depth
						if buffer[position] != rune('f') {
							goto l137
						}
						position++
						goto l136
					l137:
						position, tokenIndex, depth = position136, tokenIndex136, depth136
						if buffer[position] != rune('F') {
							goto l135
						}
						position++
					}
				l136:
					if !rules[RuleAction33]() {
						goto l135
					}
					goto l125
				l135:
					position, tokenIndex, depth = position125, tokenIndex125, depth125
					if buffer[position] != rune('\\') {
						goto l138
					}
					position++
					{
						position139, tokenIndex139, depth139 := position, tokenIndex, depth
						if buffer[position] != rune('n') {
							goto l140
						}
						position++
						goto l139
					l140:
						position, tokenIndex, depth = position139, tokenIndex139, depth139
						if buffer[position] != rune('N') {
							goto l138
						}
						position++
					}
				l139:
					if !rules[RuleAction34]() {
						goto l138
					}
					goto l125
				l138:
					position, tokenIndex, depth = position125, tokenIndex125, depth125
					if buffer[position] != rune('\\') {
						goto l141
					}
					position++
					{
						position142, tokenIndex142, depth142 := position, tokenIndex, depth
						if buffer[position] != rune('r') {
							goto l143
						}
						position++
						goto l142
					l143:
						position, tokenIndex, depth = position142, tokenIndex142, depth142
						if buffer[position] != rune('R') {
							goto l141
						}
						position++
					}
				l142:
					if !rules[RuleAction35]() {
						goto l141
					}
					goto l125
				l141:
					position, tokenIndex, depth = position125, tokenIndex125, depth125
					if buffer[position] != rune('\\') {
						goto l144
					}
					position++
					{
						position145, tokenIndex145, depth145 := position, tokenIndex, depth
						if buffer[position] != rune('t') {
							goto l146
						}
						position++
						goto l145
					l146:
						position, tokenIndex, depth = position145, tokenIndex145, depth145
						if buffer[position] != rune('T') {
							goto l144
						}
						position++
					}
				l145:
					if !rules[RuleAction36]() {
						goto l144
					}
					goto l125
				l144:
					position, tokenIndex, depth = position125, tokenIndex125, depth125
					if buffer[position] != rune('\\') {
						goto l147
					}
					position++
					{
						position148, tokenIndex148, depth148 := position, tokenIndex, depth
						if buffer[position] != rune('v') {
							goto l149
						}
						position++
						goto l148
					l149:
						position, tokenIndex, depth = position148, tokenIndex148, depth148
						if buffer[position] != rune('V') {
							goto l147
						}
						position++
					}
				l148:
					if !rules[RuleAction37]() {
						goto l147
					}
					goto l125
				l147:
					position, tokenIndex, depth = position125, tokenIndex125, depth125
					if buffer[position] != rune('\\') {
						goto l150
					}
					position++
					if buffer[position] != rune('\'') {
						goto l150
					}
					position++
					if !rules[RuleAction38]() {
						goto l150
					}
					goto l125
				l150:
					position, tokenIndex, depth = position125, tokenIndex125, depth125
					if buffer[position] != rune('\\') {
						goto l151
					}
					position++
					if buffer[position] != rune('"') {
						goto l151
					}
					position++
					if !rules[RuleAction39]() {
						goto l151
					}
					goto l125
				l151:
					position, tokenIndex, depth = position125, tokenIndex125, depth125
					if buffer[position] != rune('\\') {
						goto l152
					}
					position++
					if buffer[position] != rune('[') {
						goto l152
					}
					position++
					if !rules[RuleAction40]() {
						goto l152
					}
					goto l125
				l152:
					position, tokenIndex, depth = position125, tokenIndex125, depth125
					if buffer[position] != rune('\\') {
						goto l153
					}
					position++
					if buffer[position] != rune(']') {
						goto l153
					}
					position++
					if !rules[RuleAction41]() {
						goto l153
					}
					goto l125
				l153:
					position, tokenIndex, depth = position125, tokenIndex125, depth125
					if buffer[position] != rune('\\') {
						goto l154
					}
					position++
					if buffer[position] != rune('-') {
						goto l154
					}
					position++
					if !rules[RuleAction42]() {
						goto l154
					}
					goto l125
				l154:
					position, tokenIndex, depth = position125, tokenIndex125, depth125
					if buffer[position] != rune('\\') {
						goto l155
					}
					position++
					{
						position156 := position
						depth++
						if c := buffer[position]; c < rune('0') || c > rune('3') {
							goto l155
						}
						position++
						if c := buffer[position]; c < rune('0') || c > rune('7') {
							goto l155
						}
						position++
						if c := buffer[position]; c < rune('0') || c > rune('7') {
							goto l155
						}
						position++
						depth--
						add(RulePegText, position156)
					}
					if !rules[RuleAction43]() {
						goto l155
					}
					goto l125
				l155:
					position, tokenIndex, depth = position125, tokenIndex125, depth125
					if buffer[position] != rune('\\') {
						goto l157
					}
					position++
					{
						position158 := position
						depth++
						if c := buffer[position]; c < rune('0') || c > rune('7') {
							goto l157
						}
						position++
						{
							position159, tokenIndex159, depth159 := position, tokenIndex, depth
							if c := buffer[position]; c < rune('0') || c > rune('7') {
								goto l159
							}
							position++
							goto l160
						l159:
							position, tokenIndex, depth = position159, tokenIndex159, depth159
						}
					l160:
						depth--
						add(RulePegText, position158)
					}
					if !rules[RuleAction44]() {
						goto l157
					}
					goto l125
				l157:
					position, tokenIndex, depth = position125, tokenIndex125, depth125
					if buffer[position] != rune('\\') {
						goto l123
					}
					position++
					if buffer[position] != rune('\\') {
						goto l123
					}
					position++
					if !rules[RuleAction45]() {
						goto l123
					}
				}
			l125:
				depth--
				add(RuleEscape, position124)
			}
			return true
		l123:
			position, tokenIndex, depth = position123, tokenIndex123, depth123
			return false
		},
		/* 19 LeftArrow <- <('<' '-' Spacing)> */
		func() bool {
			position161, tokenIndex161, depth161 := position, tokenIndex, depth
			{
				position162 := position
				depth++
				if buffer[position] != rune('<') {
					goto l161
				}
				position++
				if buffer[position] != rune('-') {
					goto l161
				}
				position++
				if !rules[RuleSpacing]() {
					goto l161
				}
				depth--
				add(RuleLeftArrow, position162)
			}
			return true
		l161:
			position, tokenIndex, depth = position161, tokenIndex161, depth161
			return false
		},
		/* 20 Slash <- <('/' Spacing)> */
		func() bool {
			position163, tokenIndex163, depth163 := position, tokenIndex, depth
			{
				position164 := position
				depth++
				if buffer[position] != rune('/') {
					goto l163
				}
				position++
				if !rules[RuleSpacing]() {
					goto l163
				}
				depth--
				add(RuleSlash, position164)
			}
			return true
		l163:
			position, tokenIndex, depth = position163, tokenIndex163, depth163
			return false
		},
		/* 21 And <- <('&' Spacing)> */
		func() bool {
			position165, tokenIndex165, depth165 := position, tokenIndex, depth
			{
				position166 := position
				depth++
				if buffer[position] != rune('&') {
					goto l165
				}
				position++
				if !rules[RuleSpacing]() {
					goto l165
				}
				depth--
				add(RuleAnd, position166)
			}
			return true
		l165:
			position, tokenIndex, depth = position165, tokenIndex165, depth165
			return false
		},
		/* 22 Not <- <('!' Spacing)> */
		func() bool {
			position167, tokenIndex167, depth167 := position, tokenIndex, depth
			{
				position168 := position
				depth++
				if buffer[position] != rune('!') {
					goto l167
				}
				position++
				if !rules[RuleSpacing]() {
					goto l167
				}
				depth--
				add(RuleNot, position168)
			}
			return true
		l167:
			position, tokenIndex, depth = position167, tokenIndex167, depth167
			return false
		},
		/* 23 Question <- <('?' Spacing)> */
		func() bool {
			position169, tokenIndex169, depth169 := position, tokenIndex, depth
			{
				position170 := position
				depth++
				if buffer[position] != rune('?') {
					goto l169
				}
				position++
				if !rules[RuleSpacing]() {
					goto l169
				}
				depth--
				add(RuleQuestion, position170)
			}
			return true
		l169:
			position, tokenIndex, depth = position169, tokenIndex169, depth169
			return false
		},
		/* 24 Star <- <('*' Spacing)> */
		func() bool {
			position171, tokenIndex171, depth171 := position, tokenIndex, depth
			{
				position172 := position
				depth++
				if buffer[position] != rune('*') {
					goto l171
				}
				position++
				if !rules[RuleSpacing]() {
					goto l171
				}
				depth--
				add(RuleStar, position172)
			}
			return true
		l171:
			position, tokenIndex, depth = position171, tokenIndex171, depth171
			return false
		},
		/* 25 Plus <- <('+' Spacing)> */
		func() bool {
			position173, tokenIndex173, depth173 := position, tokenIndex, depth
			{
				position174 := position
				depth++
				if buffer[position] != rune('+') {
					goto l173
				}
				position++
				if !rules[RuleSpacing]() {
					goto l173
				}
				depth--
				add(RulePlus, position174)
			}
			return true
		l173:
			position, tokenIndex, depth = position173, tokenIndex173, depth173
			return false
		},
		/* 26 Open <- <('(' Spacing)> */
		func() bool {
			position175, tokenIndex175, depth175 := position, tokenIndex, depth
			{
				position176 := position
				depth++
				if buffer[position] != rune('(') {
					goto l175
				}
				position++
				if !rules[RuleSpacing]() {
					goto l175
				}
				depth--
				add(RuleOpen, position176)
			}
			return true
		l175:
			position, tokenIndex, depth = position175, tokenIndex175, depth175
			return false
		},
		/* 27 Close <- <(')' Spacing)> */
		func() bool {
			position177, tokenIndex177, depth177 := position, tokenIndex, depth
			{
				position178 := position
				depth++
				if buffer[position] != rune(')') {
					goto l177
				}
				position++
				if !rules[RuleSpacing]() {
					goto l177
				}
				depth--
				add(RuleClose, position178)
			}
			return true
		l177:
			position, tokenIndex, depth = position177, tokenIndex177, depth177
			return false
		},
		/* 28 Dot <- <('.' Spacing)> */
		func() bool {
			position179, tokenIndex179, depth179 := position, tokenIndex, depth
			{
				position180 := position
				depth++
				if buffer[position] != rune('.') {
					goto l179
				}
				position++
				if !rules[RuleSpacing]() {
					goto l179
				}
				depth--
				add(RuleDot, position180)
			}
			return true
		l179:
			position, tokenIndex, depth = position179, tokenIndex179, depth179
			return false
		},
		/* 29 Spacing <- <(Space / Comment)*> */
		func() bool {
			{
				position182 := position
				depth++
			l183:
				{
					position184, tokenIndex184, depth184 := position, tokenIndex, depth
					{
						position185, tokenIndex185, depth185 := position, tokenIndex, depth
						if !rules[RuleSpace]() {
							goto l186
						}
						goto l185
					l186:
						position, tokenIndex, depth = position185, tokenIndex185, depth185
						if !rules[RuleComment]() {
							goto l184
						}
					}
				l185:
					goto l183
				l184:
					position, tokenIndex, depth = position184, tokenIndex184, depth184
				}
				depth--
				add(RuleSpacing, position182)
			}
			return true
		},
		/* 30 Comment <- <('#' (!EndOfLine .)* EndOfLine)> */
		func() bool {
			position187, tokenIndex187, depth187 := position, tokenIndex, depth
			{
				position188 := position
				depth++
				if buffer[position] != rune('#') {
					goto l187
				}
				position++
			l189:
				{
					position190, tokenIndex190, depth190 := position, tokenIndex, depth
					{
						position191, tokenIndex191, depth191 := position, tokenIndex, depth
						if !rules[RuleEndOfLine]() {
							goto l191
						}
						goto l190
					l191:
						position, tokenIndex, depth = position191, tokenIndex191, depth191
					}
					if !matchDot() {
						goto l190
					}
					goto l189
				l190:
					position, tokenIndex, depth = position190, tokenIndex190, depth190
				}
				if !rules[RuleEndOfLine]() {
					goto l187
				}
				depth--
				add(RuleComment, position188)
			}
			return true
		l187:
			position, tokenIndex, depth = position187, tokenIndex187, depth187
			return false
		},
		/* 31 Space <- <(' ' / '\t' / EndOfLine)> */
		func() bool {
			position192, tokenIndex192, depth192 := position, tokenIndex, depth
			{
				position193 := position
				depth++
				{
					position194, tokenIndex194, depth194 := position, tokenIndex, depth
					if buffer[position] != rune(' ') {
						goto l195
					}
					position++
					goto l194
				l195:
					position, tokenIndex, depth = position194, tokenIndex194, depth194
					if buffer[position] != rune('\t') {
						goto l196
					}
					position++
					goto l194
				l196:
					position, tokenIndex, depth = position194, tokenIndex194, depth194
					if !rules[RuleEndOfLine]() {
						goto l192
					}
				}
			l194:
				depth--
				add(RuleSpace, position193)
			}
			return true
		l192:
			position, tokenIndex, depth = position192, tokenIndex192, depth192
			return false
		},
		/* 32 EndOfLine <- <(('\r' '\n') / '\n' / '\r')> */
		func() bool {
			position197, tokenIndex197, depth197 := position, tokenIndex, depth
			{
				position198 := position
				depth++
				{
					position199, tokenIndex199, depth199 := position, tokenIndex, depth
					if buffer[position] != rune('\r') {
						goto l200
					}
					position++
					if buffer[position] != rune('\n') {
						goto l200
					}
					position++
					goto l199
				l200:
					position, tokenIndex, depth = position199, tokenIndex199, depth199
					if buffer[position] != rune('\n') {
						goto l201
					}
					position++
					goto l199
				l201:
					position, tokenIndex, depth = position199, tokenIndex199, depth199
					if buffer[position] != rune('\r') {
						goto l197
					}
					position++
				}
			l199:
				depth--
				add(RuleEndOfLine, position198)
			}
			return true
		l197:
			position, tokenIndex, depth = position197, tokenIndex197, depth197
			return false
		},
		/* 33 EndOfFile <- <!.> */
		func() bool {
			position202, tokenIndex202, depth202 := position, tokenIndex, depth
			{
				position203 := position
				depth++
				{
					position204, tokenIndex204, depth204 := position, tokenIndex, depth
					if !matchDot() {
						goto l204
					}
					goto l202
				l204:
					position, tokenIndex, depth = position204, tokenIndex204, depth204
				}
				depth--
				add(RuleEndOfFile, position203)
			}
			return true
		l202:
			position, tokenIndex, depth = position202, tokenIndex202, depth202
			return false
		},
		/* 34 Action <- <('{' <ActionInner> '}' Spacing)> */
		func() bool {
			position205, tokenIndex205, depth205 := position, tokenIndex, depth
			{
				position206 := position
				depth++
				if buffer[position] != rune('{') {
					goto l205
				}
				position++
				{
					position207 := position
					depth++
					if !rules[RuleActionInner]() {
						goto l205
					}
					depth--
					add(RulePegText, position207)
				}
				if buffer[position] != rune('}') {
					goto l205
				}
				position++
				if !rules[RuleSpacing]() {
					goto l205
				}
				depth--
				add(RuleAction, position206)
			}
			return true
		l205:
			position, tokenIndex, depth = position205, tokenIndex205, depth205
			return false
		},
		/* 35 ActionInner <- <((!('{' / '}') .)* ('{' ActionInner '}' (!('{' / '}') .)*)*)> */
		func() bool {
			{
				position209 := position
				depth++
			l210:
				{
					position211, tokenIndex211, depth211 := position, tokenIndex, depth
					{
						position212, tokenIndex212, depth212 := position, tokenIndex, depth
						{
							position213, tokenIndex213, depth213 := position, tokenIndex, depth
							if buffer[position] != rune('{') {
								goto l214
							}
							position++
							goto l213
						l214:
							position, tokenIndex, depth = position213, tokenIndex213, depth213
							if buffer[position] != rune('}') {
								goto l212
							}
							position++
						}
					l213:
						goto l211
					l212:
						position, tokenIndex, depth = position212, tokenIndex212, depth212
					}
					if !matchDot() {
						goto l211
					}
					goto l210
				l211:
					position, tokenIndex, depth = position211, tokenIndex211, depth211
				}
			l215:
				{
					position216, tokenIndex216, depth216 := position, tokenIndex, depth
					if buffer[position] != rune('{') {
						goto l216
					}
					position++
					if !rules[RuleActionInner]() {
						goto l216
					}
					if buffer[position] != rune('}') {
						goto l216
					}
					position++
				l217:
					{
						position218, tokenIndex218, depth218 := position, tokenIndex, depth
						{
							position219, tokenIndex219, depth219 := position, tokenIndex, depth
							{
								position220, tokenIndex220, depth220 := position, tokenIndex, depth
								if buffer[position] != rune('{') {
									goto l221
								}
								position++
								goto l220
							l221:
								position, tokenIndex, depth = position220, tokenIndex220, depth220
								if buffer[position] != rune('}') {
									goto l219
								}
								position++
							}
						l220:
							goto l218
						l219:
							position, tokenIndex, depth = position219, tokenIndex219, depth219
						}
						if !matchDot() {
							goto l218
						}
						goto l217
					l218:
						position, tokenIndex, depth = position218, tokenIndex218, depth218
					}
					goto l215
				l216:
					position, tokenIndex, depth = position216, tokenIndex216, depth216
				}
				depth--
				add(RuleActionInner, position209)
			}
			return true
		},
		/* 36 Begin <- <('<' Spacing)> */
		func() bool {
			position222, tokenIndex222, depth222 := position, tokenIndex, depth
			{
				position223 := position
				depth++
				if buffer[position] != rune('<') {
					goto l222
				}
				position++
				if !rules[RuleSpacing]() {
					goto l222
				}
				depth--
				add(RuleBegin, position223)
			}
			return true
		l222:
			position, tokenIndex, depth = position222, tokenIndex222, depth222
			return false
		},
		/* 37 End <- <('>' Spacing)> */
		func() bool {
			position224, tokenIndex224, depth224 := position, tokenIndex, depth
			{
				position225 := position
				depth++
				if buffer[position] != rune('>') {
					goto l224
				}
				position++
				if !rules[RuleSpacing]() {
					goto l224
				}
				depth--
				add(RuleEnd, position225)
			}
			return true
		l224:
			position, tokenIndex, depth = position224, tokenIndex224, depth224
			return false
		},
		/* 39 Action0 <- <{ p.AddPackage(buffer[begin:end]) }> */
		func() bool {
			{
				add(RuleAction0, position)
			}
			return true
		},
		/* 40 Action1 <- <{ p.AddPeg(buffer[begin:end]) }> */
		func() bool {
			{
				add(RuleAction1, position)
			}
			return true
		},
		/* 41 Action2 <- <{ p.AddState(buffer[begin:end]) }> */
		func() bool {
			{
				add(RuleAction2, position)
			}
			return true
		},
		/* 42 Action3 <- <{ p.AddRule(buffer[begin:end]) }> */
		func() bool {
			{
				add(RuleAction3, position)
			}
			return true
		},
		/* 43 Action4 <- <{ p.AddExpression() }> */
		func() bool {
			{
				add(RuleAction4, position)
			}
			return true
		},
		/* 44 Action5 <- <{ p.AddAlternate() }> */
		func() bool {
			{
				add(RuleAction5, position)
			}
			return true
		},
		/* 45 Action6 <- <{ p.AddNil(); p.AddAlternate() }> */
		func() bool {
			{
				add(RuleAction6, position)
			}
			return true
		},
		/* 46 Action7 <- <{ p.AddNil() }> */
		func() bool {
			{
				add(RuleAction7, position)
			}
			return true
		},
		/* 47 Action8 <- <{ p.AddSequence() }> */
		func() bool {
			{
				add(RuleAction8, position)
			}
			return true
		},
		/* 48 Action9 <- <{ p.AddPredicate(buffer[begin:end]) }> */
		func() bool {
			{
				add(RuleAction9, position)
			}
			return true
		},
		/* 49 Action10 <- <{ p.AddPeekFor() }> */
		func() bool {
			{
				add(RuleAction10, position)
			}
			return true
		},
		/* 50 Action11 <- <{ p.AddPeekNot() }> */
		func() bool {
			{
				add(RuleAction11, position)
			}
			return true
		},
		/* 51 Action12 <- <{ p.AddQuery() }> */
		func() bool {
			{
				add(RuleAction12, position)
			}
			return true
		},
		/* 52 Action13 <- <{ p.AddStar() }> */
		func() bool {
			{
				add(RuleAction13, position)
			}
			return true
		},
		/* 53 Action14 <- <{ p.AddPlus() }> */
		func() bool {
			{
				add(RuleAction14, position)
			}
			return true
		},
		/* 54 Action15 <- <{ p.AddName(buffer[begin:end]) }> */
		func() bool {
			{
				add(RuleAction15, position)
			}
			return true
		},
		/* 55 Action16 <- <{ p.AddDot() }> */
		func() bool {
			{
				add(RuleAction16, position)
			}
			return true
		},
		/* 56 Action17 <- <{ p.AddAction(buffer[begin:end]) }> */
		func() bool {
			{
				add(RuleAction17, position)
			}
			return true
		},
		/* 57 Action18 <- <{ p.AddPush() }> */
		func() bool {
			{
				add(RuleAction18, position)
			}
			return true
		},
		nil,
		/* 59 Action19 <- <{ p.AddSequence() }> */
		func() bool {
			{
				add(RuleAction19, position)
			}
			return true
		},
		/* 60 Action20 <- <{ p.AddSequence() }> */
		func() bool {
			{
				add(RuleAction20, position)
			}
			return true
		},
		/* 61 Action21 <- <{ p.AddPeekNot(); p.AddDot(); p.AddSequence() }> */
		func() bool {
			{
				add(RuleAction21, position)
			}
			return true
		},
		/* 62 Action22 <- <{ p.AddPeekNot(); p.AddDot(); p.AddSequence() }> */
		func() bool {
			{
				add(RuleAction22, position)
			}
			return true
		},
		/* 63 Action23 <- <{ p.AddAlternate() }> */
		func() bool {
			{
				add(RuleAction23, position)
			}
			return true
		},
		/* 64 Action24 <- <{ p.AddAlternate() }> */
		func() bool {
			{
				add(RuleAction24, position)
			}
			return true
		},
		/* 65 Action25 <- <{ p.AddRange() }> */
		func() bool {
			{
				add(RuleAction25, position)
			}
			return true
		},
		/* 66 Action26 <- <{ p.AddDoubleRange() }> */
		func() bool {
			{
				add(RuleAction26, position)
			}
			return true
		},
		/* 67 Action27 <- <{ p.AddCharacter(buffer[begin:end]) }> */
		func() bool {
			{
				add(RuleAction27, position)
			}
			return true
		},
		/* 68 Action28 <- <{ p.AddDoubleCharacter(buffer[begin:end]) }> */
		func() bool {
			{
				add(RuleAction28, position)
			}
			return true
		},
		/* 69 Action29 <- <{ p.AddCharacter(buffer[begin:end]) }> */
		func() bool {
			{
				add(RuleAction29, position)
			}
			return true
		},
		/* 70 Action30 <- <{ p.AddCharacter("\a") }> */
		func() bool {
			{
				add(RuleAction30, position)
			}
			return true
		},
		/* 71 Action31 <- <{ p.AddCharacter("\b") }> */
		func() bool {
			{
				add(RuleAction31, position)
			}
			return true
		},
		/* 72 Action32 <- <{ p.AddCharacter("\x1B") }> */
		func() bool {
			{
				add(RuleAction32, position)
			}
			return true
		},
		/* 73 Action33 <- <{ p.AddCharacter("\f") }> */
		func() bool {
			{
				add(RuleAction33, position)
			}
			return true
		},
		/* 74 Action34 <- <{ p.AddCharacter("\n") }> */
		func() bool {
			{
				add(RuleAction34, position)
			}
			return true
		},
		/* 75 Action35 <- <{ p.AddCharacter("\r") }> */
		func() bool {
			{
				add(RuleAction35, position)
			}
			return true
		},
		/* 76 Action36 <- <{ p.AddCharacter("\t") }> */
		func() bool {
			{
				add(RuleAction36, position)
			}
			return true
		},
		/* 77 Action37 <- <{ p.AddCharacter("\v") }> */
		func() bool {
			{
				add(RuleAction37, position)
			}
			return true
		},
		/* 78 Action38 <- <{ p.AddCharacter("'") }> */
		func() bool {
			{
				add(RuleAction38, position)
			}
			return true
		},
		/* 79 Action39 <- <{ p.AddCharacter("\"") }> */
		func() bool {
			{
				add(RuleAction39, position)
			}
			return true
		},
		/* 80 Action40 <- <{ p.AddCharacter("[") }> */
		func() bool {
			{
				add(RuleAction40, position)
			}
			return true
		},
		/* 81 Action41 <- <{ p.AddCharacter("]") }> */
		func() bool {
			{
				add(RuleAction41, position)
			}
			return true
		},
		/* 82 Action42 <- <{ p.AddCharacter("-") }> */
		func() bool {
			{
				add(RuleAction42, position)
			}
			return true
		},
		/* 83 Action43 <- <{ p.AddOctalCharacter(buffer[begin:end]) }> */
		func() bool {
			{
				add(RuleAction43, position)
			}
			return true
		},
		/* 84 Action44 <- <{ p.AddOctalCharacter(buffer[begin:end]) }> */
		func() bool {
			{
				add(RuleAction44, position)
			}
			return true
		},
		/* 85 Action45 <- <{ p.AddCharacter("\\") }> */
		func() bool {
			{
				add(RuleAction45, position)
			}
			return true
		},
	}
	p.rules = rules
}
