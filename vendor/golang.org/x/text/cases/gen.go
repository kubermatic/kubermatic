// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

// This program generates the trie for casing operations. The Unicode casing
// algorithm requires the lookup of various properties and mappings for each
// rune. The table generated by this generator combines several of the most
// frequently used of these into a single trie so that they can be accessed
// with a single lookup.
package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"reflect"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/internal/gen"
	"golang.org/x/text/internal/triegen"
	"golang.org/x/text/internal/ucd"
	"golang.org/x/text/unicode/norm"
)

func main() {
	gen.Init()
	genTables()
	genTablesTest()
	gen.Repackage("gen_trieval.go", "trieval.go", "cases")
}

// runeInfo contains all information for a rune that we care about for casing
// operations.
type runeInfo struct {
	Rune rune

	entry info // trie value for this rune.

	CaseMode info

	// Simple case mappings.
	Simple [1 + maxCaseMode][]rune

	// Special casing
	HasSpecial  bool
	Conditional bool
	Special     [1 + maxCaseMode][]rune

	// Folding
	FoldSimple  rune
	FoldSpecial rune
	FoldFull    []rune

	// TODO: FC_NFKC, or equivalent data.

	// Properties
	SoftDotted     bool
	CaseIgnorable  bool
	Cased          bool
	DecomposeGreek bool
	BreakType      string
	BreakCat       breakCategory

	// We care mostly about 0, Above, and IotaSubscript.
	CCC byte
}

type breakCategory int

const (
	breakBreak breakCategory = iota
	breakLetter
	breakIgnored
)

// mapping returns the case mapping for the given case type.
func (r *runeInfo) mapping(c info) string {
	if r.HasSpecial {
		return string(r.Special[c])
	}
	if len(r.Simple[c]) != 0 {
		return string(r.Simple[c])
	}
	return string(r.Rune)
}

func parse(file string, f func(p *ucd.Parser)) {
	ucd.Parse(gen.OpenUCDFile(file), f)
}

func parseUCD() []runeInfo {
	chars := make([]runeInfo, unicode.MaxRune)

	get := func(r rune) *runeInfo {
		c := &chars[r]
		c.Rune = r
		return c
	}

	parse("UnicodeData.txt", func(p *ucd.Parser) {
		ri := get(p.Rune(0))
		ri.CCC = byte(p.Int(ucd.CanonicalCombiningClass))
		ri.Simple[cLower] = p.Runes(ucd.SimpleLowercaseMapping)
		ri.Simple[cUpper] = p.Runes(ucd.SimpleUppercaseMapping)
		ri.Simple[cTitle] = p.Runes(ucd.SimpleTitlecaseMapping)
		if p.String(ucd.GeneralCategory) == "Lt" {
			ri.CaseMode = cTitle
		}
	})

	// <code>; <property>
	parse("PropList.txt", func(p *ucd.Parser) {
		if p.String(1) == "Soft_Dotted" {
			chars[p.Rune(0)].SoftDotted = true
		}
	})

	// <code>; <word break type>
	parse("DerivedCoreProperties.txt", func(p *ucd.Parser) {
		ri := get(p.Rune(0))
		switch p.String(1) {
		case "Case_Ignorable":
			ri.CaseIgnorable = true
		case "Cased":
			ri.Cased = true
		case "Lowercase":
			ri.CaseMode = cLower
		case "Uppercase":
			ri.CaseMode = cUpper
		}
	})

	// <code>; <lower> ; <title> ; <upper> ; (<condition_list> ;)?
	parse("SpecialCasing.txt", func(p *ucd.Parser) {
		// We drop all conditional special casing and deal with them manually in
		// the language-specific case mappers. Rune 0x03A3 is the only one with
		// a conditional formatting that is not language-specific. However,
		// dealing with this letter is tricky, especially in a streaming
		// context, so we deal with it in the Caser for Greek specifically.
		ri := get(p.Rune(0))
		if p.String(4) == "" {
			ri.HasSpecial = true
			ri.Special[cLower] = p.Runes(1)
			ri.Special[cTitle] = p.Runes(2)
			ri.Special[cUpper] = p.Runes(3)
		} else {
			ri.Conditional = true
		}
	})

	// TODO: Use text breaking according to UAX #29.
	// <code>; <word break type>
	parse("auxiliary/WordBreakProperty.txt", func(p *ucd.Parser) {
		ri := get(p.Rune(0))
		ri.BreakType = p.String(1)

		// We collapse the word breaking properties onto the categories we need.
		switch p.String(1) { // TODO: officially we need to canonicalize.
		case "Format", "MidLetter", "MidNumLet", "Single_Quote":
			ri.BreakCat = breakIgnored
		case "ALetter", "Hebrew_Letter", "Numeric", "Extend", "ExtendNumLet":
			ri.BreakCat = breakLetter
		}
	})

	// <code>; <type>; <mapping>
	parse("CaseFolding.txt", func(p *ucd.Parser) {
		ri := get(p.Rune(0))
		switch p.String(1) {
		case "C":
			ri.FoldSimple = p.Rune(2)
			ri.FoldFull = p.Runes(2)
		case "S":
			ri.FoldSimple = p.Rune(2)
		case "T":
			ri.FoldSpecial = p.Rune(2)
		case "F":
			ri.FoldFull = p.Runes(2)
		default:
			log.Fatalf("%U: unknown type: %s", p.Rune(0), p.String(1))
		}
	})

	return chars
}

func genTables() {
	chars := parseUCD()
	verifyProperties(chars)

	t := triegen.NewTrie("case")
	for i := range chars {
		c := &chars[i]
		makeEntry(c)
		t.Insert(rune(i), uint64(c.entry))
	}

	w := gen.NewCodeWriter()
	defer w.WriteGoFile("tables.go", "cases")

	gen.WriteUnicodeVersion(w)

	// TODO: write CLDR version after adding a mechanism to detect that the
	// tables on which the manually created locale-sensitive casing code is
	// based hasn't changed.

	w.WriteVar("xorData", string(xorData))
	w.WriteVar("exceptions", string(exceptionData))

	sz, err := t.Gen(w, triegen.Compact(&sparseCompacter{}))
	if err != nil {
		log.Fatal(err)
	}
	w.Size += sz
}

func makeEntry(ri *runeInfo) {
	if ri.CaseIgnorable {
		if ri.Cased {
			ri.entry = cIgnorableCased
		} else {
			ri.entry = cIgnorableUncased
		}
	} else {
		ri.entry = ri.CaseMode
	}

	// TODO: handle soft-dotted.

	ccc := cccOther
	switch ri.CCC {
	case 0: // Not_Reordered
		ccc = cccZero
	case above: // Above
		ccc = cccAbove
	}
	if ri.BreakCat == breakBreak {
		ccc = cccBreak
	}

	ri.entry |= ccc

	if ri.CaseMode == cUncased {
		return
	}

	// Need to do something special.
	if ri.CaseMode == cTitle || ri.HasSpecial || ri.mapping(cTitle) != ri.mapping(cUpper) {
		makeException(ri)
		return
	}
	if f := string(ri.FoldFull); len(f) > 0 && f != ri.mapping(cUpper) && f != ri.mapping(cLower) {
		makeException(ri)
		return
	}

	// Rune is either lowercase or uppercase.

	orig := string(ri.Rune)
	mapped := ""
	if ri.CaseMode == cUpper {
		mapped = ri.mapping(cLower)
	} else {
		mapped = ri.mapping(cUpper)
	}

	if len(orig) != len(mapped) {
		makeException(ri)
		return
	}

	if string(ri.FoldFull) == ri.mapping(cUpper) {
		ri.entry |= inverseFoldBit
	}

	n := len(orig)

	// Create per-byte XOR mask.
	var b []byte
	for i := 0; i < n; i++ {
		b = append(b, orig[i]^mapped[i])
	}

	// Remove leading 0 bytes, but keep at least one byte.
	for ; len(b) > 1 && b[0] == 0; b = b[1:] {
	}

	if len(b) == 1 && b[0]&0xc0 == 0 {
		ri.entry |= info(b[0]) << xorShift
		return
	}

	key := string(b)
	x, ok := xorCache[key]
	if !ok {
		xorData = append(xorData, 0) // for detecting start of sequence
		xorData = append(xorData, b...)

		x = len(xorData) - 1
		xorCache[key] = x
	}
	ri.entry |= info(x<<xorShift) | xorIndexBit
}

var xorCache = map[string]int{}

// xorData contains byte-wise XOR data for the least significant bytes of a
// UTF-8 encoded rune. An index points to the last byte. The sequence starts
// with a zero terminator.
var xorData = []byte{}

// See the comments in gen_trieval.go re "the exceptions slice".
var exceptionData = []byte{0}

// makeException encodes case mappings that cannot be expressed in a simple
// XOR diff.
func makeException(ri *runeInfo) {
	ccc := ri.entry & cccMask
	// Set exception bit and retain case type.
	ri.entry &= 0x0007
	ri.entry |= exceptionBit

	if len(exceptionData) >= 1<<numExceptionBits {
		log.Fatalf("%U:exceptionData too large %x > %d bits", ri.Rune, len(exceptionData), numExceptionBits)
	}

	// Set the offset in the exceptionData array.
	ri.entry |= info(len(exceptionData) << exceptionShift)

	orig := string(ri.Rune)
	tc := ri.mapping(cTitle)
	uc := ri.mapping(cUpper)
	lc := ri.mapping(cLower)
	ff := string(ri.FoldFull)

	// addString sets the length of a string and adds it to the expansions array.
	addString := func(s string, b *byte) {
		if len(s) == 0 {
			// Zero-length mappings exist, but only for conditional casing,
			// which we are representing outside of this table.
			log.Fatalf("%U: has zero-length mapping.", ri.Rune)
		}
		*b <<= 3
		if s != orig {
			n := len(s)
			if n > 7 {
				log.Fatalf("%U: mapping larger than 7 (%d)", ri.Rune, n)
			}
			*b |= byte(n)
			exceptionData = append(exceptionData, s...)
		}
	}

	// byte 0:
	exceptionData = append(exceptionData, byte(ccc)|byte(len(ff)))

	// byte 1:
	p := len(exceptionData)
	exceptionData = append(exceptionData, 0)

	if len(ff) > 7 { // May be zero-length.
		log.Fatalf("%U: fold string larger than 7 (%d)", ri.Rune, len(ff))
	}
	exceptionData = append(exceptionData, ff...)
	ct := ri.CaseMode
	if ct != cLower {
		addString(lc, &exceptionData[p])
	}
	if ct != cUpper {
		addString(uc, &exceptionData[p])
	}
	if ct != cTitle {
		// If title is the same as upper, we set it to the original string so
		// that it will be marked as not present. This implies title case is
		// the same as upper case.
		if tc == uc {
			tc = orig
		}
		addString(tc, &exceptionData[p])
	}
}

// sparseCompacter is a trie value block Compacter. There are many cases where
// successive runes alternate between lower- and upper-case. This Compacter
// exploits this by adding a special case type where the case value is obtained
// from or-ing it with the least-significant bit of the rune, creating large
// ranges of equal case values that compress well.
type sparseCompacter struct {
	sparseBlocks  [][]uint16
	sparseOffsets []uint16
	sparseCount   int
}

// makeSparse returns the number of elements that compact block would contain
// as well as the modified values.
func makeSparse(vals []uint64) ([]uint16, int) {
	// Copy the values.
	values := make([]uint16, len(vals))
	for i, v := range vals {
		values[i] = uint16(v)
	}

	alt := func(i int, v uint16) uint16 {
		if cm := info(v & fullCasedMask); cm == cUpper || cm == cLower {
			// Convert cLower or cUpper to cXORCase value, which has the form 11x.
			xor := v
			xor &^= 1
			xor |= uint16(i&1) ^ (v & 1)
			xor |= 0x4
			return xor
		}
		return v
	}

	var count int
	var previous uint16
	for i, v := range values {
		if v != 0 {
			// Try if the unmodified value is equal to the previous.
			if v == previous {
				continue
			}

			// Try if the xor-ed value is equal to the previous value.
			a := alt(i, v)
			if a == previous {
				values[i] = a
				continue
			}

			// This is a new value.
			count++

			// Use the xor-ed value if it will be identical to the next value.
			if p := i + 1; p < len(values) && alt(p, values[p]) == a {
				values[i] = a
				v = a
			}
		}
		previous = v
	}
	return values, count
}

func (s *sparseCompacter) Size(v []uint64) (int, bool) {
	_, n := makeSparse(v)

	// We limit using this method to having 16 entries.
	if n > 16 {
		return 0, false
	}

	return 2 + int(reflect.TypeOf(valueRange{}).Size())*n, true
}

func (s *sparseCompacter) Store(v []uint64) uint32 {
	h := uint32(len(s.sparseOffsets))
	values, sz := makeSparse(v)
	s.sparseBlocks = append(s.sparseBlocks, values)
	s.sparseOffsets = append(s.sparseOffsets, uint16(s.sparseCount))
	s.sparseCount += sz
	return h
}

func (s *sparseCompacter) Handler() string {
	// The sparse global variable and its lookup method is defined in gen_trieval.go.
	return "sparse.lookup"
}

func (s *sparseCompacter) Print(w io.Writer) (retErr error) {
	p := func(format string, args ...interface{}) {
		_, err := fmt.Fprintf(w, format, args...)
		if retErr == nil && err != nil {
			retErr = err
		}
	}

	ls := len(s.sparseBlocks)
	if ls == len(s.sparseOffsets) {
		s.sparseOffsets = append(s.sparseOffsets, uint16(s.sparseCount))
	}
	p("// sparseOffsets: %d entries, %d bytes\n", ls+1, (ls+1)*2)
	p("var sparseOffsets = %#v\n\n", s.sparseOffsets)

	ns := s.sparseCount
	p("// sparseValues: %d entries, %d bytes\n", ns, ns*4)
	p("var sparseValues = [%d]valueRange {", ns)
	for i, values := range s.sparseBlocks {
		p("\n// Block %#x, offset %#x", i, s.sparseOffsets[i])
		var v uint16
		for i, nv := range values {
			if nv != v {
				if v != 0 {
					p(",hi:%#02x},", 0x80+i-1)
				}
				if nv != 0 {
					p("\n{value:%#04x,lo:%#02x", nv, 0x80+i)
				}
			}
			v = nv
		}
		if v != 0 {
			p(",hi:%#02x},", 0x80+len(values)-1)
		}
	}
	p("\n}\n\n")
	return
}

// verifyProperties that properties of the runes that are relied upon in the
// implementation. Each property is marked with an identifier that is referred
// to in the places where it is used.
func verifyProperties(chars []runeInfo) {
	for i, c := range chars {
		r := rune(i)

		// Rune properties.

		// A.1: modifier never changes on lowercase. [ltLower]
		if c.CCC > 0 && unicode.ToLower(r) != r {
			log.Fatalf("%U: non-starter changes when lowercased", r)
		}

		// A.2: properties of decompositions starting with I or J. [ltLower]
		d := norm.NFD.PropertiesString(string(r)).Decomposition()
		if len(d) > 0 {
			if d[0] == 'I' || d[0] == 'J' {
				// A.2.1: we expect at least an ASCII character and a modifier.
				if len(d) < 3 {
					log.Fatalf("%U: length of decomposition was %d; want >= 3", r, len(d))
				}

				// All subsequent runes are modifiers and all have the same CCC.
				runes := []rune(string(d[1:]))
				ccc := chars[runes[0]].CCC

				for _, mr := range runes[1:] {
					mc := chars[mr]

					// A.2.2: all modifiers have a CCC of Above or less.
					if ccc == 0 || ccc > above {
						log.Fatalf("%U: CCC of successive rune (%U) was %d; want (0,230]", r, mr, ccc)
					}

					// A.2.3: a sequence of modifiers all have the same CCC.
					if mc.CCC != ccc {
						log.Fatalf("%U: CCC of follow-up modifier (%U) was %d; want %d", r, mr, mc.CCC, ccc)
					}

					// A.2.4: for each trailing r, r in [0x300, 0x311] <=> CCC == Above.
					if (ccc == above) != (0x300 <= mr && mr <= 0x311) {
						log.Fatalf("%U: modifier %U in [U+0300, U+0311] != ccc(%U) == 230", r, mr, mr)
					}

					if i += len(string(mr)); i >= len(d) {
						break
					}
				}
			}
		}

		// A.3: no U+0307 in decomposition of Soft-Dotted rune. [ltUpper]
		if unicode.Is(unicode.Soft_Dotted, r) && strings.Contains(string(d), "\u0307") {
			log.Fatalf("%U: decomposition of soft-dotted rune may not contain U+0307", r)
		}

		// A.4: only rune U+0345 may be of CCC Iota_Subscript. [elUpper]
		if c.CCC == iotaSubscript && r != 0x0345 {
			log.Fatalf("%U: only rune U+0345 may have CCC Iota_Subscript", r)
		}

		// A.5: soft-dotted runes do not have exceptions.
		if c.SoftDotted && c.entry&exceptionBit != 0 {
			log.Fatalf("%U: soft-dotted has exception", r)
		}

		// A.6: Greek decomposition. [elUpper]
		if unicode.Is(unicode.Greek, r) {
			if b := norm.NFD.PropertiesString(string(r)).Decomposition(); b != nil {
				runes := []rune(string(b))
				// A.6.1: If a Greek rune decomposes and the first rune of the
				// decomposition is greater than U+00FF, the rune is always
				// great and not a modifier.
				if f := runes[0]; unicode.IsMark(f) || f > 0xFF && !unicode.Is(unicode.Greek, f) {
					log.Fatalf("%U: expeced first rune of Greek decomposition to be letter, found %U", r, f)
				}
				// A.6.2: Any follow-up rune in a Greek decomposition is a
				// modifier of which the first should be gobbled in
				// decomposition.
				for _, m := range runes[1:] {
					switch m {
					case 0x0313, 0x0314, 0x0301, 0x0300, 0x0306, 0x0342, 0x0308, 0x0304, 0x345:
					default:
						log.Fatalf("%U: modifier %U is outside of expeced Greek modifier set", r, m)
					}
				}
			}
		}

		// Breaking properties.

		// B.1: all runes with CCC > 0 are of break type Extend.
		if c.CCC > 0 && c.BreakType != "Extend" {
			log.Fatalf("%U: CCC == %d, but got break type %s; want Extend", r, c.CCC, c.BreakType)
		}

		// B.2: all cased runes with c.CCC == 0 are of break type ALetter.
		if c.CCC == 0 && c.Cased && c.BreakType != "ALetter" {
			log.Fatalf("%U: cased, but got break type %s; want ALetter", r, c.BreakType)
		}

		// B.3: letter category.
		if c.CCC == 0 && c.BreakCat != breakBreak && !c.CaseIgnorable {
			if c.BreakCat != breakLetter {
				log.Fatalf("%U: check for letter break type gave %d; want %d", r, c.BreakCat, breakLetter)
			}
		}
	}
}

func genTablesTest() {
	w := &bytes.Buffer{}

	fmt.Fprintln(w, "var (")
	printProperties(w, "DerivedCoreProperties.txt", "Case_Ignorable", verifyIgnore)

	// We discard the output as we know we have perfect functions. We run them
	// just to verify the properties are correct.
	n := printProperties(ioutil.Discard, "DerivedCoreProperties.txt", "Cased", verifyCased)
	n += printProperties(ioutil.Discard, "DerivedCoreProperties.txt", "Lowercase", verifyLower)
	n += printProperties(ioutil.Discard, "DerivedCoreProperties.txt", "Uppercase", verifyUpper)
	if n > 0 {
		log.Fatalf("One of the discarded properties does not have a perfect filter.")
	}

	// <code>; <lower> ; <title> ; <upper> ; (<condition_list> ;)?
	fmt.Fprintln(w, "\tspecial = map[rune]struct{ toLower, toTitle, toUpper string }{")
	parse("SpecialCasing.txt", func(p *ucd.Parser) {
		// Skip conditional entries.
		if p.String(4) != "" {
			return
		}
		r := p.Rune(0)
		fmt.Fprintf(w, "\t\t0x%04x: {%q, %q, %q},\n",
			r, string(p.Runes(1)), string(p.Runes(2)), string(p.Runes(3)))
	})
	fmt.Fprint(w, "\t}\n\n")

	// <code>; <type>; <runes>
	table := map[rune]struct{ simple, full, special string }{}
	parse("CaseFolding.txt", func(p *ucd.Parser) {
		r := p.Rune(0)
		t := p.String(1)
		v := string(p.Runes(2))
		if t != "T" && v == string(unicode.ToLower(r)) {
			return
		}
		x := table[r]
		switch t {
		case "C":
			x.full = v
			x.simple = v
		case "S":
			x.simple = v
		case "F":
			x.full = v
		case "T":
			x.special = v
		}
		table[r] = x
	})
	fmt.Fprintln(w, "\tfoldMap = map[rune]struct{ simple, full, special string }{")
	for r := rune(0); r < 0x10FFFF; r++ {
		x, ok := table[r]
		if !ok {
			continue
		}
		fmt.Fprintf(w, "\t\t0x%04x: {%q, %q, %q},\n", r, x.simple, x.full, x.special)
	}
	fmt.Fprint(w, "\t}\n\n")

	// Break property
	notBreak := map[rune]bool{}
	parse("auxiliary/WordBreakProperty.txt", func(p *ucd.Parser) {
		switch p.String(1) {
		case "Extend", "Format", "MidLetter", "MidNumLet", "Single_Quote",
			"ALetter", "Hebrew_Letter", "Numeric", "ExtendNumLet":
			notBreak[p.Rune(0)] = true
		}
	})

	fmt.Fprintln(w, "\tbreakProp = []struct{ lo, hi rune }{")
	inBreak := false
	for r := rune(0); r <= lastRuneForTesting; r++ {
		if isBreak := !notBreak[r]; isBreak != inBreak {
			if isBreak {
				fmt.Fprintf(w, "\t\t{0x%x, ", r)
			} else {
				fmt.Fprintf(w, "0x%x},\n", r-1)
			}
			inBreak = isBreak
		}
	}
	if inBreak {
		fmt.Fprintf(w, "0x%x},\n", lastRuneForTesting)
	}
	fmt.Fprint(w, "\t}\n\n")

	// Word break test
	// Filter out all samples that do not contain cased characters.
	cased := map[rune]bool{}
	parse("DerivedCoreProperties.txt", func(p *ucd.Parser) {
		if p.String(1) == "Cased" {
			cased[p.Rune(0)] = true
		}
	})

	fmt.Fprintln(w, "\tbreakTest = []string{")
	parse("auxiliary/WordBreakTest.txt", func(p *ucd.Parser) {
		c := strings.Split(p.String(0), " ")

		const sep = '|'
		numCased := 0
		test := ""
		for ; len(c) >= 2; c = c[2:] {
			if c[0] == "÷" && test != "" {
				test += string(sep)
			}
			i, err := strconv.ParseUint(c[1], 16, 32)
			r := rune(i)
			if err != nil {
				log.Fatalf("Invalid rune %q.", c[1])
			}
			if r == sep {
				log.Fatalf("Separator %q not allowed in test data. Pick another one.", sep)
			}
			if cased[r] {
				numCased++
			}
			test += string(r)
		}
		if numCased > 1 {
			fmt.Fprintf(w, "\t\t%q,\n", test)
		}
	})
	fmt.Fprintln(w, "\t}")

	fmt.Fprintln(w, ")")

	gen.WriteGoFile("tables_test.go", "cases", w.Bytes())
}

// These functions are just used for verification that their definition have not
// changed in the Unicode Standard.

func verifyCased(r rune) bool {
	return verifyLower(r) || verifyUpper(r) || unicode.IsTitle(r)
}

func verifyLower(r rune) bool {
	return unicode.IsLower(r) || unicode.Is(unicode.Other_Lowercase, r)
}

func verifyUpper(r rune) bool {
	return unicode.IsUpper(r) || unicode.Is(unicode.Other_Uppercase, r)
}

// verifyIgnore is an approximation of the Case_Ignorable property using the
// core unicode package. It is used to reduce the size of the test data.
func verifyIgnore(r rune) bool {
	props := []*unicode.RangeTable{
		unicode.Mn,
		unicode.Me,
		unicode.Cf,
		unicode.Lm,
		unicode.Sk,
	}
	for _, p := range props {
		if unicode.Is(p, r) {
			return true
		}
	}
	return false
}

// printProperties prints tables of rune properties from the given UCD file.
// A filter func f can be given to exclude certain values. A rune r will have
// the indicated property if it is in the generated table or if f(r).
func printProperties(w io.Writer, file, property string, f func(r rune) bool) int {
	verify := map[rune]bool{}
	n := 0
	varNameParts := strings.Split(property, "_")
	varNameParts[0] = strings.ToLower(varNameParts[0])
	fmt.Fprintf(w, "\t%s = map[rune]bool{\n", strings.Join(varNameParts, ""))
	parse(file, func(p *ucd.Parser) {
		if p.String(1) == property {
			r := p.Rune(0)
			verify[r] = true
			if !f(r) {
				n++
				fmt.Fprintf(w, "\t\t0x%.4x: true,\n", r)
			}
		}
	})
	fmt.Fprint(w, "\t}\n\n")

	// Verify that f is correct, that is, it represents a subset of the property.
	for r := rune(0); r <= lastRuneForTesting; r++ {
		if !verify[r] && f(r) {
			log.Fatalf("Incorrect filter func for property %q.", property)
		}
	}
	return n
}

// The newCaseTrie, sparseValues and sparseOffsets definitions below are
// placeholders referred to by gen_trieval.go. The real definitions are
// generated by this program and written to tables.go.

func newCaseTrie(int) int { return 0 }

var (
	sparseValues  [0]valueRange
	sparseOffsets [0]uint16
)
