package deyaml

import (
	"fmt"
	"io"
	"reflect"
	"strconv"
	"text/tabwriter"

	"github.com/kr/text"
)

type formatter struct {
	v              reflect.Value
	force          bool
	quote          bool
	packageAliases map[string]string
}

func prettyFormatter(x interface{}, packageAliases map[string]string) (f fmt.Formatter) {
	return formatter{v: reflect.ValueOf(x), quote: true, packageAliases: packageAliases}
}

func (fo formatter) String() string {
	return fmt.Sprint(fo.v.Interface()) // unwrap it
}

func (fo formatter) passThrough(f fmt.State, c rune) {
	s := "%"
	for i := 0; i < 128; i++ {
		if f.Flag(i) {
			s += string(i)
		}
	}
	if w, ok := f.Width(); ok {
		s += fmt.Sprintf("%d", w)
	}
	if p, ok := f.Precision(); ok {
		s += fmt.Sprintf(".%d", p)
	}
	s += string(c)
	_, _ = fmt.Fprintf(f, s, fo.v.Interface())
}

func (fo formatter) Format(f fmt.State, c rune) {
	if fo.force || c == 'v' && f.Flag('#') && f.Flag(' ') {
		w := tabwriter.NewWriter(f, 4, 4, 1, ' ', 0)
		p := &printer{tw: w, Writer: w, visited: make(map[visit]int), packageAliases: fo.packageAliases}
		p.printValue(fo.v, true, fo.quote)
		_ = w.Flush()
		return
	}
	fo.passThrough(f, c)
}

type printer struct {
	io.Writer
	packageAliases map[string]string
	tw             *tabwriter.Writer
	visited        map[visit]int
	depth          int
}

func (p *printer) indent() *printer {
	q := *p
	q.tw = tabwriter.NewWriter(p.Writer, 4, 4, 1, ' ', 0)
	q.Writer = text.NewIndentWriter(q.tw, []byte{'\t'})
	return &q
}

func (p *printer) printInline(v reflect.Value, x interface{}, showType bool) {
	if showType {
		_, _ = io.WriteString(p, p.aliasedType(v.Type()))
		_, _ = fmt.Fprintf(p, "(%#v)", x)
	} else {
		_, _ = fmt.Fprintf(p, "%#v", x)
	}
}

// printValue must keep track of already-printed pointer values to avoid
// infinite recursion.
type visit struct {
	v   uintptr
	typ reflect.Type
}

func (p *printer) printValue(v reflect.Value, showType, quote bool) {
	if p.depth > 10 {
		_, _ = io.WriteString(p, "!%v(DEPTH EXCEEDED)")
		return
	}

	switch v.Kind() {
	case reflect.Bool:
		p.printInline(v, v.Bool(), showType)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		p.printInline(v, v.Int(), showType)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		p.printInline(v, v.Uint(), showType)
	case reflect.Float32, reflect.Float64:
		p.printInline(v, v.Float(), showType)
	case reflect.Complex64, reflect.Complex128:
		_, _ = fmt.Fprintf(p, "%#v", v.Complex())
	case reflect.String:
		p.fmtString(v.String(), quote)
	case reflect.Map:
		t := v.Type()
		_, _ = io.WriteString(p, p.aliasedType(t))
		writeByte(p, '{')
		if nonzero(v) {
			expand := !canInline(v.Type())
			pp := p
			if expand {
				writeByte(p, '\n')
				pp = p.indent()
			}
			keys := v.MapKeys()
			for i := 0; i < v.Len(); i++ {
				showTypeInStruct := true
				k := keys[i]
				mv := v.MapIndex(k)
				pp.printValue(k, true, true)
				writeByte(pp, ':')
				if expand {
					writeByte(pp, '\t')
				}
				showTypeInStruct = t.Elem().Kind() == reflect.Interface
				pp.printValue(mv, showTypeInStruct, true)
				if expand {
					_, _ = io.WriteString(pp, ",\n")
				} else if i < v.Len()-1 {
					_, _ = io.WriteString(pp, ", ")
				}
			}
			if expand {
				_ = pp.tw.Flush()
			}
		}
		writeByte(p, '}')
	case reflect.Struct:
		t := v.Type()
		if v.CanAddr() {
			addr := v.UnsafeAddr()
			vis := visit{addr, t}
			if vd, ok := p.visited[vis]; ok && vd < p.depth {
				p.fmtString(t.String()+"{(CYCLIC REFERENCE)}", false)
				break // don't print v again
			}
			p.visited[vis] = p.depth
		}

		_, _ = io.WriteString(p, p.aliasedType(t))
		writeByte(p, '{')
		if nonzero(v) {
			expand := !canInline(v.Type())
			pp := p
			if expand {
				writeByte(p, '\n')
				pp = p.indent()
			}
			for i := 0; i < v.NumField(); i++ {
				field := getField(v, i)
				if !nonzero(field) {
					continue
				}
				showTypeInStruct := true
				if f := t.Field(i); f.Name != "" {
					_, _ = io.WriteString(pp, f.Name)
					writeByte(pp, ':')
					if expand {
						writeByte(pp, '\t')
					}
					showTypeInStruct = labelType(f.Type)
				}
				pp.printValue(field, showTypeInStruct, true)
				if expand {
					_, _ = io.WriteString(pp, ",\n")
				} else if i < v.NumField()-1 {
					_, _ = io.WriteString(pp, ", ")
				}
			}
			if expand {
				_ = pp.tw.Flush()
			}
		}
		writeByte(p, '}')
	case reflect.Interface:
		switch e := v.Elem(); {
		case e.Kind() == reflect.Invalid:
			_, _ = io.WriteString(p, "nil")
		case e.IsValid():
			pp := *p
			pp.depth++
			pp.printValue(e, showType, true)
		default:
			_, _ = io.WriteString(p, p.aliasedType(v.Type()))
			_, _ = io.WriteString(p, "(nil)")
		}
	case reflect.Array, reflect.Slice:
		t := v.Type()
		if v.Kind() == reflect.Slice && v.IsNil() && showType {
			_, _ = io.WriteString(p, "(nil)")
			break
		}
		if v.Kind() == reflect.Slice && v.IsNil() {
			_, _ = io.WriteString(p, "nil")
			break
		}
		_, _ = io.WriteString(p, p.aliasedType(t))
		writeByte(p, '{')
		expand := !canInline(v.Type())
		pp := p
		if expand {
			writeByte(p, '\n')
			pp = p.indent()
		}
		for i := 0; i < v.Len(); i++ {
			showTypeInSlice := t.Elem().Kind() == reflect.Interface
			pp.printValue(v.Index(i), showTypeInSlice, true)
			if expand {
				_, _ = io.WriteString(pp, ",\n")
			} else if i < v.Len()-1 {
				_, _ = io.WriteString(pp, ", ")
			}
		}
		if expand {
			_ = pp.tw.Flush()
		}
		writeByte(p, '}')
	case reflect.Ptr:
		e := v.Elem()
		if !e.IsValid() {
			writeByte(p, '(')
			_, _ = io.WriteString(p, p.aliasedType(v.Type()))
			_, _ = io.WriteString(p, ")(nil)")
		} else {
			pp := *p
			pp.depth++
			writeByte(pp, '&')
			pp.printValue(e, true, true)
		}
	case reflect.Chan:
		x := v.Pointer()
		if showType {
			writeByte(p, '(')
			_, _ = io.WriteString(p, p.aliasedType(v.Type()))
			_, _ = fmt.Fprintf(p, ")(%#v)", x)
		} else {
			_, _ = fmt.Fprintf(p, "%#v", x)
		}
	case reflect.Func:
		_, _ = io.WriteString(p, p.aliasedType(v.Type()))
		_, _ = io.WriteString(p, " {...}")
	case reflect.UnsafePointer:
		p.printInline(v, v.Pointer(), showType)
	case reflect.Invalid:
		_, _ = io.WriteString(p, "nil")
	}
}

func (p *printer) aliasedType(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Ptr:
		return fmt.Sprintf("*%s", p.aliasedType(t.Elem()))
	case reflect.Slice:
		return fmt.Sprintf("[]%s", p.aliasedType(t.Elem()))
	case reflect.Map:
		return fmt.Sprintf("map[%s]%s", p.aliasedType(t.Key()), p.aliasedType(t.Elem()))
	}

	if alias := p.packageAliases[t.PkgPath()]; alias != "" {
		return fmt.Sprintf("%s.%s", alias, t.Name())
	}
	return t.String()
}

func canInline(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Map:
		return !canExpand(t.Elem())
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			if canExpand(t.Field(i).Type) {
				return false
			}
		}
		return true
	case reflect.Interface:
		return false
	case reflect.Array, reflect.Slice:
		return !canExpand(t.Elem())
	case reflect.Ptr:
		return false
	case reflect.Chan, reflect.Func, reflect.UnsafePointer:
		return false
	}
	return true
}

func canExpand(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Map, reflect.Struct,
		reflect.Interface, reflect.Array, reflect.Slice,
		reflect.Ptr:
		return true
	}
	return false
}

func labelType(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Interface, reflect.Struct:
		return true
	}
	return false
}

func (p *printer) fmtString(s string, quote bool) {
	if quote {
		s = strconv.Quote(s)
	}
	_, _ = io.WriteString(p, s)
}

func writeByte(w io.Writer, b byte) {
	_, _ = w.Write([]byte{b})
}

func getField(v reflect.Value, i int) reflect.Value {
	val := v.Field(i)
	if val.Kind() == reflect.Interface && !val.IsNil() {
		val = val.Elem()
	}
	return val
}

func nonzero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Bool:
		return v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() != 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() != 0
	case reflect.Float32, reflect.Float64:
		return v.Float() != 0
	case reflect.Complex64, reflect.Complex128:
		return v.Complex() != complex(0, 0)
	case reflect.String:
		return v.String() != ""
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if nonzero(getField(v, i)) {
				return true
			}
		}
		return false
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if nonzero(v.Index(i)) {
				return true
			}
		}
		return false
	case reflect.Map, reflect.Interface, reflect.Slice, reflect.Ptr, reflect.Chan, reflect.Func:
		return !v.IsNil()
	case reflect.UnsafePointer:
		return v.Pointer() != 0
	}
	return true
}
