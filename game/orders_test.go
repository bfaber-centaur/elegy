package game

import (
	"bytes"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/bfaber-centaur/elegy/engine"
)

// TestOrderKindsCoverEngine reads the engine's source for every type
// with an apply method (the engine.Order kinds) and checks that the order
// file format names each one, so a new order kind cannot be left out.
func TestOrderKindsCoverEngine(t *testing.T) {
	files, err := filepath.Glob("../engine/*.go")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{}
	fset := token.NewFileSet()
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		af, err := parser.ParseFile(fset, f, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range af.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Recv == nil || fd.Name.Name != "apply" {
				continue
			}
			if id, ok := fd.Recv.List[0].Type.(*ast.Ident); ok && ast.IsExported(id.Name) {
				want[id.Name] = true
			}
		}
	}
	have := map[string]bool{}
	for _, o := range orderKinds {
		have[reflect.TypeOf(o).Name()] = true
	}
	var missing, extra []string
	for n := range want {
		if !have[n] {
			missing = append(missing, n)
		}
	}
	for n := range have {
		if !want[n] {
			extra = append(extra, n)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	if len(want) == 0 || len(missing) > 0 || len(extra) > 0 {
		t.Fatalf("engine order kinds %d; missing from the file format %v; not engine orders %v", len(want), missing, extra)
	}
}

// fill sets every field of v to a non-zero value derived from seed.
func fill(v reflect.Value, seed *int) {
	*seed++
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v.SetInt(int64(*seed))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v.SetUint(uint64(*seed))
	case reflect.Bool:
		v.SetBool(true)
	case reflect.String:
		v.SetString("s" + strings.Repeat("x", *seed%5))
	case reflect.Struct:
		for i := range v.NumField() {
			fill(v.Field(i), seed)
		}
	case reflect.Array:
		for i := range v.Len() {
			fill(v.Index(i), seed)
		}
	case reflect.Slice:
		s := reflect.MakeSlice(v.Type(), 2, 2)
		for i := range 2 {
			fill(s.Index(i), seed)
		}
		v.Set(s)
	case reflect.Pointer:
		p := reflect.New(v.Type().Elem())
		fill(p.Elem(), seed)
		v.Set(p)
	}
}

func TestOrderFileRoundTrip(t *testing.T) {
	kinds := make([]string, 0, len(orderKinds))
	for k := range orderKinds {
		kinds = append(kinds, k)
	}
	sort.Strings(kinds)
	in := OrderFile{GameID: 7, Year: 2412, Player: 3}
	seed := 0
	for _, k := range kinds {
		if bad := opaqueFields(reflect.TypeOf(orderKinds[k]), map[reflect.Type]bool{}); len(bad) > 0 {
			t.Fatalf("%s: %v cannot be written to an order file", k, bad)
		}
		p := reflect.New(reflect.TypeOf(orderKinds[k]))
		fill(p.Elem(), &seed)
		in.Orders = append(in.Orders, p.Elem().Interface().(engine.Order))
	}
	var buf bytes.Buffer
	if err := EncodeOrders(&buf, in); err != nil {
		t.Fatal(err)
	}
	out, err := DecodeOrders(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("round trip changed the file:\nin  %+v\nout %+v", in, out)
	}
}

func TestDecodeOrdersRejects(t *testing.T) {
	for name, doc := range map[string]string{
		"not json":      `{`,
		"format":        `{"format": "x", "version": 1, "orders": []}`,
		"version":       `{"format": "elegy-orders", "version": 2, "orders": []}`,
		"unknown kind":  `{"format": "elegy-orders", "version": 1, "orders": [{"kind": "Teleport", "order": {}}]}`,
		"unknown field": `{"format": "elegy-orders", "version": 1, "orders": [{"kind": "Research", "order": {"Budgett": 1}}]}`,
	} {
		if _, err := DecodeOrders(strings.NewReader(doc)); !errors.Is(err, ErrOrderFile) {
			t.Errorf("%s: err = %v, want ErrOrderFile", name, err)
		}
	}
}
