package argmapper

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"
)

func init() {
	hclog.L().SetLevel(hclog.Trace)
}

func TestFuncRedefine(t *testing.T) {
	cases := []struct {
		Name       string
		Func       interface{}
		Args       []Arg
		Err        string
		CallArgs   []Arg
		CallResult []interface{}
	}{
		{
			"all arguments satisfied",
			func(in struct {
				Struct

				A, B int
			}) int {
				return in.A + in.B
			},
			[]Arg{
				Named("a", 12),
				Named("b", 24),
			},
			"",
			[]Arg{},
			[]interface{}{36},
		},

		{
			"missing a named argument",
			func(in struct {
				Struct

				A, B int
			}) int {
				return in.A + in.B
			},
			[]Arg{
				Named("a", 12),
			},
			"",
			[]Arg{
				Named("b", 24),
			},
			[]interface{}{36},
		},

		//-----------------------------------------------------------
		// FILTER INPUT

		{
			"only through strings direct",
			func(in struct {
				Struct

				A, B int
			}) int {
				return in.A + in.B
			},
			[]Arg{
				Named("a", 12),
				Converter(func(v string) (int, error) { return strconv.Atoi(v) }),
				FilterInput(func(v Value) bool { return v.Type.Kind() == reflect.String }),
			},
			"",
			[]Arg{
				Typed("24"),
			},
			[]interface{}{36},
		},

		{
			"only through strings with bidirectional converter",
			func(in struct {
				Struct

				A, B int
			}) int {
				return in.A + in.B
			},
			[]Arg{
				Named("a", 12),
				Converter(func(v string) (int, error) { return strconv.Atoi(v) }),
				Converter(func(v int) string { return strconv.Itoa(v) }),
				FilterInput(func(v Value) bool { return v.Type.Kind() == reflect.String }),
			},
			"",
			[]Arg{
				Typed("24"),
			},
			[]interface{}{36},
		},

		{
			"input with two possible converters",
			func(in struct {
				Struct

				A, B int
			}) int {
				return in.A + in.B
			},
			[]Arg{
				Named("a", 12),
				Converter(func(v string) (int, error) { return strconv.Atoi(v) }),
				Converter(func(v []byte) string { return string(v) }),
				Converter(func(v string) []byte { return []byte(v) }),
				FilterInput(FilterOr(
					FilterType(reflect.TypeOf("")),
				)),
			},
			"",
			[]Arg{
				Typed("24"),
			},
			[]interface{}{36},
		},

		{
			"unsatisfiable",
			func(in struct {
				Struct

				A, B int
			}) int {
				return in.A + in.B
			},
			[]Arg{
				Named("a", 12),
				FilterInput(func(v Value) bool { return v.Type.Kind() == reflect.String }),
			},
			`"b" (type: int)`,
			[]Arg{},
			nil,
		},

		//-----------------------------------------------------------
		// FILTER OUTPUT

		{
			"satisfy output type",
			func(in struct {
				Struct

				A, B int
			}) int {
				return in.A + in.B
			},
			[]Arg{
				Named("a", 12),
				Named("b", 24),
				FilterOutput(FilterType(reflect.TypeOf(int(0)))),
			},
			"",
			[]Arg{},
			[]interface{}{36},
		},

		{
			"satisfy output type with error",
			func(in struct {
				Struct

				A, B int
			}) (int, error) {
				return in.A + in.B, nil
			},
			[]Arg{
				Named("a", 12),
				Named("b", 24),
				FilterOutput(FilterType(reflect.TypeOf(int(0)))),
			},
			"",
			[]Arg{},
			[]interface{}{36},
		},

		{
			"satisfy output type with only error result",
			func(in struct {
				Struct

				A, B int
			}) error {
				return nil
			},
			[]Arg{
				Named("a", 12),
				Named("b", 24),
				FilterOutput(FilterType(reflect.TypeOf(int(0)))),
			},
			"",
			[]Arg{},
			[]interface{}{nil},
		},

		{
			"fail to satisfy output type",
			func(in struct {
				Struct

				A, B int
			}) int {
				return in.A + in.B
			},
			[]Arg{
				Named("a", 12),
				Named("b", 24),
				FilterOutput(FilterType(reflect.TypeOf(string("")))),
			},
			"output type: int",
			nil,
			nil,
		},
	}

	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			require := require.New(t)

			f, err := NewFunc(tt.Func)
			require.NoError(err)

			redefined, err := f.Redefine(tt.Args...)
			if tt.Err == "" {
				require.NoError(err)
			} else {
				require.Error(err)
				require.Contains(err.Error(), tt.Err)
				return
			}

			result := redefined.Call(tt.CallArgs...)
			require.NoError(result.Err())
			for i, out := range tt.CallResult {
				require.Equal(out, result.Out(i))
			}
		})
	}
}
