package yaml

import (
	"encoding"
	"encoding/base64"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"time"
)

const (
	DocumentNode = 1 << iota
	MappingNode
	SequenceNode
	ScalarNode
	AliasNode
)

type Node struct {
	Kind         int
	Line, Column int
	Tag          string
	Value        string
	Implicit     bool
	Children     []*Node
	Anchors      map[string]*Node
}

// ----------------------------------------------------------------------------
// Parser, produces a Node tree out of a libyaml event stream.

type parser struct {
	parser yaml_parser_t
	event  yaml_event_t
	doc    *Node
}

func newParser(b []byte) *parser {
	p := parser{}
	if !yaml_parser_initialize(&p.parser) {
		panic("failed to initialize YAML emitter")
	}

	if len(b) == 0 {
		b = []byte{'\n'}
	}

	yaml_parser_set_input_string(&p.parser, b)

	p.skip()
	if p.event.typ != yaml_STREAM_START_EVENT {
		panic("expected stream start event, got " + strconv.Itoa(int(p.event.typ)))
	}
	p.skip()
	return &p
}

func (p *parser) destroy() {
	if p.event.typ != yaml_NO_EVENT {
		yaml_event_delete(&p.event)
	}
	yaml_parser_delete(&p.parser)
}

func (p *parser) skip() {
	if p.event.typ != yaml_NO_EVENT {
		if p.event.typ == yaml_STREAM_END_EVENT {
			failf("attempted to go past the end of stream; corrupted value?")
		}
		yaml_event_delete(&p.event)
	}
	if !yaml_parser_parse(&p.parser, &p.event) {
		p.fail()
	}
}

func (p *parser) fail() {
	var where string
	var line int
	if p.parser.problem_mark.line != 0 {
		line = p.parser.problem_mark.line
	} else if p.parser.context_mark.line != 0 {
		line = p.parser.context_mark.line
	}
	if line != 0 {
		where = "line " + strconv.Itoa(line) + ": "
	}
	var msg string
	if len(p.parser.problem) > 0 {
		msg = p.parser.problem
	} else {
		msg = "unknown problem parsing YAML content"
	}
	failf("%s%s", where, msg)
}

func (p *parser) anchor(n *Node, anchor []byte) {
	if anchor != nil {
		p.doc.Anchors[string(anchor)] = n
	}
}

func (p *parser) parse() *Node {
	switch p.event.typ {
	case yaml_SCALAR_EVENT:
		return p.scalar()
	case yaml_ALIAS_EVENT:
		return p.alias()
	case yaml_MAPPING_START_EVENT:
		return p.mapping()
	case yaml_SEQUENCE_START_EVENT:
		return p.sequence()
	case yaml_DOCUMENT_START_EVENT:
		return p.document()
	case yaml_STREAM_END_EVENT:
		// Happens when attempting to decode an empty buffer.
		return nil
	default:
		panic("attempted to parse unknown event: " + strconv.Itoa(int(p.event.typ)))
	}
}

func (p *parser) Node(kind int) *Node {
	return &Node{
		Kind:   kind,
		Line:   p.event.start_mark.line,
		Column: p.event.start_mark.column,
	}
}

func (p *parser) document() *Node {
	n := p.Node(DocumentNode)
	n.Anchors = make(map[string]*Node)
	p.doc = n
	p.skip()
	n.Children = append(n.Children, p.parse())
	if p.event.typ != yaml_DOCUMENT_END_EVENT {
		panic("expected end of document event but got " + strconv.Itoa(int(p.event.typ)))
	}
	p.skip()
	return n
}

func (p *parser) alias() *Node {
	n := p.Node(AliasNode)
	n.Value = string(p.event.anchor)
	p.skip()
	return n
}

func (p *parser) scalar() *Node {
	n := p.Node(ScalarNode)
	n.Value = string(p.event.value)
	n.Tag = string(p.event.tag)
	n.Implicit = p.event.implicit
	p.anchor(n, p.event.anchor)
	p.skip()
	return n
}

func (p *parser) sequence() *Node {
	n := p.Node(SequenceNode)
	p.anchor(n, p.event.anchor)
	p.skip()
	for p.event.typ != yaml_SEQUENCE_END_EVENT {
		n.Children = append(n.Children, p.parse())
	}
	p.skip()
	return n
}

func (p *parser) mapping() *Node {
	n := p.Node(MappingNode)
	p.anchor(n, p.event.anchor)
	p.skip()
	for p.event.typ != yaml_MAPPING_END_EVENT {
		n.Children = append(n.Children, p.parse(), p.parse())
	}
	p.skip()
	return n
}

// ----------------------------------------------------------------------------
// Decoder, unmarshals a Node into a provided value.

type decoder struct {
	doc     *Node
	aliases map[string]bool
	mapType reflect.Type
	terrors []string
	strict  bool
}

var (
	mapItemType    = reflect.TypeOf(MapItem{})
	durationType   = reflect.TypeOf(time.Duration(0))
	defaultMapType = reflect.TypeOf(map[interface{}]interface{}{})
	ifaceType      = defaultMapType.Elem()
)

func newDecoder(strict bool) *decoder {
	d := &decoder{mapType: defaultMapType, strict: strict}
	d.aliases = make(map[string]bool)
	return d
}

func (d *decoder) terror(n *Node, tag string, out reflect.Value) {
	if n.Tag != "" {
		tag = n.Tag
	}
	value := n.Value
	if tag != yaml_SEQ_TAG && tag != yaml_MAP_TAG {
		if len(value) > 10 {
			value = " `" + value[:7] + "...`"
		} else {
			value = " `" + value + "`"
		}
	}
	d.terrors = append(d.terrors, fmt.Sprintf("line %d: cannot unmarshal %s%s into %s", n.Line+1, shortTag(tag), value, out.Type()))
}

func (d *decoder) callUnmarshaler(n *Node, u Unmarshaler) (good bool) {
	terrlen := len(d.terrors)
	err := u.UnmarshalYAML(func(v interface{}) (err error) {
		defer handleErr(&err)
		d.unmarshal(n, reflect.ValueOf(v))
		if len(d.terrors) > terrlen {
			issues := d.terrors[terrlen:]
			d.terrors = d.terrors[:terrlen]
			return &TypeError{issues}
		}
		return nil
	})
	if e, ok := err.(*TypeError); ok {
		d.terrors = append(d.terrors, e.Errors...)
		return false
	}
	if err != nil {
		fail(err)
	}
	return true
}

// d.prepare initializes and dereferences pointers and calls UnmarshalYAML
// if a value is found to implement it.
// It returns the initialized and dereferenced out value, whether
// unmarshalling was already done by UnmarshalYAML, and if so whether
// its types unmarshalled appropriately.
//
// If n holds a null value, prepare returns before doing anything.
func (d *decoder) prepare(n *Node, out reflect.Value) (newout reflect.Value, unmarshaled, good bool) {
	if n.Tag == yaml_NULL_TAG || n.Kind == ScalarNode && n.Tag == "" && (n.Value == "null" || n.Value == "" && n.Implicit) {
		return out, false, false
	}
	again := true
	for again {
		again = false
		if out.Kind() == reflect.Ptr {
			if out.IsNil() {
				out.Set(reflect.New(out.Type().Elem()))
			}
			out = out.Elem()
			again = true
		}
		if out.CanAddr() {
			if u, ok := out.Addr().Interface().(Unmarshaler); ok {
				good = d.callUnmarshaler(n, u)
				return out, true, good
			}
		}
	}
	return out, false, false
}

func (d *decoder) unmarshal(n *Node, out reflect.Value) (good bool) {
	switch n.Kind {
	case DocumentNode:
		return d.document(n, out)
	case AliasNode:
		return d.alias(n, out)
	}
	out, unmarshaled, good := d.prepare(n, out)
	if unmarshaled {
		return good
	}
	switch n.Kind {
	case ScalarNode:
		good = d.scalar(n, out)
	case MappingNode:
		good = d.mapping(n, out)
	case SequenceNode:
		good = d.sequence(n, out)
	default:
		panic("internal error: unknown Node kind: " + strconv.Itoa(n.Kind))
	}
	return good
}

func (d *decoder) document(n *Node, out reflect.Value) (good bool) {
	if len(n.Children) == 1 {
		d.doc = n
		d.unmarshal(n.Children[0], out)
		return true
	}
	return false
}

func (d *decoder) alias(n *Node, out reflect.Value) (good bool) {
	an, ok := d.doc.Anchors[n.Value]
	if !ok {
		failf("unknown anchor '%s' referenced", n.Value)
	}
	if d.aliases[n.Value] {
		failf("anchor '%s' value contains itself", n.Value)
	}
	d.aliases[n.Value] = true
	good = d.unmarshal(an, out)
	delete(d.aliases, n.Value)
	return good
}

var zeroValue reflect.Value

func resetMap(out reflect.Value) {
	for _, k := range out.MapKeys() {
		out.SetMapIndex(k, zeroValue)
	}
}

func (d *decoder) scalar(n *Node, out reflect.Value) (good bool) {
	var tag string
	var resolved interface{}
	if n.Tag == "" && !n.Implicit {
		tag = yaml_STR_TAG
		resolved = n.Value
	} else {
		tag, resolved = resolve(n.Tag, n.Value)
		if tag == yaml_BINARY_TAG {
			data, err := base64.StdEncoding.DecodeString(resolved.(string))
			if err != nil {
				failf("!!binary value contains invalid base64 data")
			}
			resolved = string(data)
		}
	}
	if resolved == nil {
		if out.Kind() == reflect.Map && !out.CanAddr() {
			resetMap(out)
		} else {
			out.Set(reflect.Zero(out.Type()))
		}
		return true
	}
	if s, ok := resolved.(string); ok && out.CanAddr() {
		if u, ok := out.Addr().Interface().(encoding.TextUnmarshaler); ok {
			err := u.UnmarshalText([]byte(s))
			if err != nil {
				fail(err)
			}
			return true
		}
	}
	switch out.Kind() {
	case reflect.String:
		if tag == yaml_BINARY_TAG {
			out.SetString(resolved.(string))
			good = true
		} else if resolved != nil {
			out.SetString(n.Value)
			good = true
		}
	case reflect.Interface:
		if resolved == nil {
			out.Set(reflect.Zero(out.Type()))
		} else {
			out.Set(reflect.ValueOf(resolved))
		}
		good = true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch resolved := resolved.(type) {
		case int:
			if !out.OverflowInt(int64(resolved)) {
				out.SetInt(int64(resolved))
				good = true
			}
		case int64:
			if !out.OverflowInt(resolved) {
				out.SetInt(resolved)
				good = true
			}
		case uint64:
			if resolved <= math.MaxInt64 && !out.OverflowInt(int64(resolved)) {
				out.SetInt(int64(resolved))
				good = true
			}
		case float64:
			if resolved <= math.MaxInt64 && !out.OverflowInt(int64(resolved)) {
				out.SetInt(int64(resolved))
				good = true
			}
		case string:
			if out.Type() == durationType {
				d, err := time.ParseDuration(resolved)
				if err == nil {
					out.SetInt(int64(d))
					good = true
				}
			}
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		switch resolved := resolved.(type) {
		case int:
			if resolved >= 0 && !out.OverflowUint(uint64(resolved)) {
				out.SetUint(uint64(resolved))
				good = true
			}
		case int64:
			if resolved >= 0 && !out.OverflowUint(uint64(resolved)) {
				out.SetUint(uint64(resolved))
				good = true
			}
		case uint64:
			if !out.OverflowUint(uint64(resolved)) {
				out.SetUint(uint64(resolved))
				good = true
			}
		case float64:
			if resolved <= math.MaxUint64 && !out.OverflowUint(uint64(resolved)) {
				out.SetUint(uint64(resolved))
				good = true
			}
		}
	case reflect.Bool:
		switch resolved := resolved.(type) {
		case bool:
			out.SetBool(resolved)
			good = true
		}
	case reflect.Float32, reflect.Float64:
		switch resolved := resolved.(type) {
		case int:
			out.SetFloat(float64(resolved))
			good = true
		case int64:
			out.SetFloat(float64(resolved))
			good = true
		case uint64:
			out.SetFloat(float64(resolved))
			good = true
		case float64:
			out.SetFloat(resolved)
			good = true
		}
	case reflect.Ptr:
		if out.Type().Elem() == reflect.TypeOf(resolved) {
			// TODO DOes this make sense? When is out a Ptr except when decoding a nil value?
			elem := reflect.New(out.Type().Elem())
			elem.Elem().Set(reflect.ValueOf(resolved))
			out.Set(elem)
			good = true
		}
	}
	if !good {
		d.terror(n, tag, out)
	}
	return good
}

func settableValueOf(i interface{}) reflect.Value {
	v := reflect.ValueOf(i)
	sv := reflect.New(v.Type()).Elem()
	sv.Set(v)
	return sv
}

func (d *decoder) sequence(n *Node, out reflect.Value) (good bool) {
	l := len(n.Children)

	var iface reflect.Value
	switch out.Kind() {
	case reflect.Slice:
		out.Set(reflect.MakeSlice(out.Type(), l, l))
	case reflect.Interface:
		// No type hints. Will have to use a generic sequence.
		iface = out
		out = settableValueOf(make([]interface{}, l))
	default:
		d.terror(n, yaml_SEQ_TAG, out)
		return false
	}
	et := out.Type().Elem()

	j := 0
	for i := 0; i < l; i++ {
		e := reflect.New(et).Elem()
		if ok := d.unmarshal(n.Children[i], e); ok {
			out.Index(j).Set(e)
			j++
		}
	}
	out.Set(out.Slice(0, j))
	if iface.IsValid() {
		iface.Set(out)
	}
	return true
}

func (d *decoder) mapping(n *Node, out reflect.Value) (good bool) {
	switch out.Kind() {
	case reflect.Struct:
		return d.mappingStruct(n, out)
	case reflect.Slice:
		return d.mappingSlice(n, out)
	case reflect.Map:
		// okay
	case reflect.Interface:
		if d.mapType.Kind() == reflect.Map {
			iface := out
			out = reflect.MakeMap(d.mapType)
			iface.Set(out)
		} else {
			slicev := reflect.New(d.mapType).Elem()
			if !d.mappingSlice(n, slicev) {
				return false
			}
			out.Set(slicev)
			return true
		}
	default:
		d.terror(n, yaml_MAP_TAG, out)
		return false
	}
	outt := out.Type()
	kt := outt.Key()
	et := outt.Elem()

	mapType := d.mapType
	if outt.Key() == ifaceType && outt.Elem() == ifaceType {
		d.mapType = outt
	}

	if out.IsNil() {
		out.Set(reflect.MakeMap(outt))
	}
	l := len(n.Children)
	for i := 0; i < l; i += 2 {
		if isMerge(n.Children[i]) {
			d.merge(n.Children[i+1], out)
			continue
		}
		k := reflect.New(kt).Elem()
		if d.unmarshal(n.Children[i], k) {
			kkind := k.Kind()
			if kkind == reflect.Interface {
				kkind = k.Elem().Kind()
			}
			if kkind == reflect.Map || kkind == reflect.Slice {
				failf("invalid map key: %#v", k.Interface())
			}
			e := reflect.New(et).Elem()
			if d.unmarshal(n.Children[i+1], e) {
				out.SetMapIndex(k, e)
			}
		}
	}
	d.mapType = mapType
	return true
}

func (d *decoder) mappingSlice(n *Node, out reflect.Value) (good bool) {
	outt := out.Type()
	if outt.Elem() != mapItemType {
		d.terror(n, yaml_MAP_TAG, out)
		return false
	}

	mapType := d.mapType
	d.mapType = outt

	var slice []MapItem
	var l = len(n.Children)
	for i := 0; i < l; i += 2 {
		if isMerge(n.Children[i]) {
			d.merge(n.Children[i+1], out)
			continue
		}
		item := MapItem{}
		k := reflect.ValueOf(&item.Key).Elem()
		if d.unmarshal(n.Children[i], k) {
			v := reflect.ValueOf(&item.Value).Elem()
			if d.unmarshal(n.Children[i+1], v) {
				slice = append(slice, item)
			}
		}
	}
	out.Set(reflect.ValueOf(slice))
	d.mapType = mapType
	return true
}

func (d *decoder) mappingStruct(n *Node, out reflect.Value) (good bool) {
	sinfo, err := getStructInfo(out.Type())
	if err != nil {
		panic(err)
	}
	name := settableValueOf("")
	l := len(n.Children)

	var inlineMap reflect.Value
	var elemType reflect.Type
	if sinfo.InlineMap != -1 {
		inlineMap = out.Field(sinfo.InlineMap)
		inlineMap.Set(reflect.New(inlineMap.Type()).Elem())
		elemType = inlineMap.Type().Elem()
	}

	for i := 0; i < l; i += 2 {
		ni := n.Children[i]
		if isMerge(ni) {
			d.merge(n.Children[i+1], out)
			continue
		}
		if !d.unmarshal(ni, name) {
			continue
		}
		if info, ok := sinfo.FieldsMap[name.String()]; ok {
			var field reflect.Value
			if info.Inline == nil {
				field = out.Field(info.Num)
			} else {
				field = out.FieldByIndex(info.Inline)
			}
			d.unmarshal(n.Children[i+1], field)
		} else if sinfo.InlineMap != -1 {
			if inlineMap.IsNil() {
				inlineMap.Set(reflect.MakeMap(inlineMap.Type()))
			}
			value := reflect.New(elemType).Elem()
			d.unmarshal(n.Children[i+1], value)
			inlineMap.SetMapIndex(name, value)
		} else if d.strict {
			d.terrors = append(d.terrors, fmt.Sprintf("line %d: field %s not found in struct %s", n.Line+1, name.String(), out.Type()))
		}
	}
	return true
}

func failWantMap() {
	failf("map merge requires map or sequence of maps as the value")
}

func (d *decoder) merge(n *Node, out reflect.Value) {
	switch n.Kind {
	case MappingNode:
		d.unmarshal(n, out)
	case AliasNode:
		an, ok := d.doc.Anchors[n.Value]
		if ok && an.Kind != MappingNode {
			failWantMap()
		}
		d.unmarshal(n, out)
	case SequenceNode:
		// Step backwards as earlier Nodes take precedence.
		for i := len(n.Children) - 1; i >= 0; i-- {
			ni := n.Children[i]
			if ni.Kind == AliasNode {
				an, ok := d.doc.Anchors[ni.Value]
				if ok && an.Kind != MappingNode {
					failWantMap()
				}
			} else if ni.Kind != MappingNode {
				failWantMap()
			}
			d.unmarshal(ni, out)
		}
	default:
		failWantMap()
	}
}

func isMerge(n *Node) bool {
	return n.Kind == ScalarNode && n.Value == "<<" && (n.Implicit == true || n.Tag == yaml_MERGE_TAG)
}
