package options

import (
	"reflect"
	"testing"
	"time"
)

func TestOptionNameParsing(t *testing.T) {
	var tests = []struct {
		in        string
		out       string
		remainder string
	}{
		{"foo", "foo", ""},
		{"  foo", "foo", ""},
		{"  foo ", "foo", " "},
		{"foo=yes", "foo", "=yes"},
		{"foo = yes", "foo", " = yes"},
		{" foo = yes", "foo", " = yes"},
		{"", "", ""},
		{"   ", "", ""},
		{"=", "", "="},
	}

	for _, tt := range tests {
		d := &decoder{input: tt.in}
		out := d.parseName()
		if out != tt.out || d.input[d.offset:] != tt.remainder {
			t.Errorf("parseName(%q) = %q with remainder %q; want %q with remainder %q",
				tt.in, out, d.input[d.offset:], tt.out, tt.remainder)
		}
	}
}

func TestSkipSpace(t *testing.T) {
	var tests = []struct {
		in        string
		remainder string
	}{
		{"", ""},
		{" ", ""},
		{"  ", ""},
		{"foo", "foo"},
		{" foo ", "foo "},
		{"\tfoo ", "foo "},
	}

	for _, tt := range tests {
		d := &decoder{input: tt.in}
		d.consumeSpace()
		if d.input[d.offset:] != tt.remainder {
			t.Errorf("consumeSpace(%q) = %q, want %q", tt.in, d.input[d.offset:], tt.remainder)
		}
	}
}

func TestParseString(t *testing.T) {
	var tests = []struct {
		in        string
		match     string
		remainder string
		quoted    bool
		err       error
	}{
		// Quoted
		{`"test"`, `test`, ``, true, nil},
		{`'test'`, `test`, ``, true, nil},
		{`"te\"st"`, `te"st`, ``, true, nil},
		{`'te\'st'`, `te'st`, ``, true, nil},
		{`"te'st"`, `te'st`, ``, true, nil},
		{`"te\'st"`, `te'st`, ``, true, nil},
		{`'te"st'`, `te"st`, ``, true, nil},
		{`'te\"st'`, `te"st`, ``, true, nil},
		{`"te\\st"`, `te\st`, ``, true, nil},
		{`"\170"`, `x`, ``, true, nil},
		{`"\1705"`, `x5`, ``, true, nil},
		{`"!@#$%"`, `!@#$%`, ``, true, nil},
		{`"test"moredata`, `test`, `moredata`, true, nil},
		{`"\999"`, `999`, ``, true, nil},

		{`test`, ``, ``, true, &SyntaxError{Offset: 1, msg: "improperly quoted string"}},
		{`"test`, ``, ``, true, &SyntaxError{Offset: 5, msg: "unexpected end of input"}},
		{``, ``, ``, true, &SyntaxError{Offset: 0, msg: "unexpected end of input"}},
		{`"\27"`, ``, ``, true, &SyntaxError{Offset: 4, msg: "invalid octal number"}},
		{`\27`, ``, ``, false, &SyntaxError{Offset: 3, msg: "invalid octal number"}},
		{"'\x00'", ``, ``, true, &SyntaxError{Offset: 1, msg: "invalid byte in string"}},
		{`"`, ``, ``, true, &SyntaxError{Offset: 0, msg: "unexpected end of input"}},

		// Unquoted
		{`test`, `test`, ``, false, nil},
		{`te\ st`, `te st`, ``, false, nil},
		{`te st`, `te`, ` st`, false, nil},
		{`te\"st`, `te"st`, ``, false, nil},
		{`te\'st`, `te'st`, ``, false, nil},

		{`te'st`, ``, ``, false, &SyntaxError{Offset: 2, msg: "unescaped quote in unquoted string"}},
		{`te"st`, ``, ``, false, &SyntaxError{Offset: 2, msg: "unescaped quote in unquoted string"}},

		{`te\\st`, `te\st`, ``, false, nil},
		{`\170`, `x`, ``, false, nil},
		{`\1705`, `x5`, ``, false, nil},
		{`!@#$%`, `!@#$%`, ``, false, nil},

		{`"test`, ``, ``, false, &SyntaxError{Offset: 0, msg: "unescaped quote in unquoted string"}},
	}

	for _, tt := range tests {
		d := &decoder{input: tt.in}
		match, err := d.parseString(tt.quoted)
		remainder := d.input[d.offset:]
		if tt.err != nil {
			if match != tt.match || !reflect.DeepEqual(err, tt.err) {
				t.Errorf("parseQuotedValue(%q, %t) = %q, %#v; want %q, %#v",
					tt.in, tt.quoted,
					match, err,
					tt.match, tt.err)
			}
		} else {
			if match != tt.match ||
				(tt.err == nil && remainder != tt.remainder) ||
				err != tt.err {

				t.Errorf("parseQuotedValue(%q, %t) = %q, %q, %#v; want %q, %q, %#v",
					tt.in, tt.quoted,
					match, remainder, err,
					tt.match, tt.remainder, tt.err)
			}
		}
	}
}

func TestParseBool(t *testing.T) {
	var tests = []struct {
		in  string
		out bool
		ok  bool
	}{
		// boolvalue
		{"yes", true, true},
		{"true", true, true},
		{"no", false, true},
		{"false", false, true},
		{"foo", false, false},
		{"true_", false, false},
	}

	for _, tt := range tests {
		ret, ok := ParseBool(tt.in)
		if ret != tt.out {
			t.Errorf("ParseBool(%q) = %t, %t; want %t, %t", tt.in, ret, ok, tt.out, tt.ok)
		}
	}
}

func TestParseNumber(t *testing.T) {
	var tests = []struct {
		in  string
		out int
		ok  bool
	}{
		{"123", 123, true},
		{"-123", -123, true},
		{"+123", 123, true},
		{"123_", 0, false},
		{"foo", 0, false},
		{"", 0, false},
		{"12-3", 0, false},
	}

	for _, tt := range tests {
		ret, ok := ParseNumber(tt.in)
		if ret != tt.out {
			t.Errorf("ParseNumber(%q) = %d, %t; want %d, %t", tt.in, ret, ok, tt.out, tt.ok)
		}
	}
}

func TestParseRange(t *testing.T) {
	var tests = []struct {
		in  string
		out Range
		ok  bool
	}{
		// rangevalue
		{"1-2", Range{1, 2}, true},
		{"123-234", Range{123, 234}, true},
		{"1-2_", Range{}, false},
		{"foo", Range{}, false},
		{"123-", Range{}, false},
		{"123--123", Range{}, false},
		{"123-+123", Range{}, false},
		{"-123-123", Range{}, false},
	}

	for _, tt := range tests {
		ret, ok := ParseRange(tt.in)
		if ret != tt.out {
			t.Errorf("ParseRange(%q) = %v, %t; want %v, %t", tt.in, ret, ok, tt.out, tt.ok)
		}
	}
}

func TestParseResolution(t *testing.T) {
	var tests = []struct {
		in  string
		out Resolution
		ok  bool
	}{
		// resvalue
		{"300dpi", Resolution{300, 300}, true},
		{"300x100dpi", Resolution{300, 100}, true},
		{"118dpc", Resolution{300, 300}, true},
		{"300dpx", Resolution{}, false},
		{"300x300x300dpi", Resolution{}, false},
		{"-300dpi", Resolution{}, false},
		{"dpi", Resolution{}, false},
	}

	for _, tt := range tests {
		ret, ok := ParseResolution(tt.in)
		if ret != tt.out {
			t.Errorf("ParseResolution(%q) = %v, %t; want %v, %t", tt.in, ret, ok, tt.out, tt.ok)
		}
	}
}

func TestParseDate(t *testing.T) {
	var tests = []struct {
		in  string
		out time.Time
		ok  bool
	}{
		{"1234", time.Date(0, 1, 1, 12, 34, 0, 0, time.UTC), true},
		{"123456", time.Date(0, 1, 1, 12, 34, 56, 0, time.UTC), true},
		{"20020904", time.Date(2002, 9, 4, 0, 0, 0, 0, time.UTC), true},
		{"200209041234", time.Date(2002, 9, 4, 12, 34, 0, 0, time.UTC), true},
		{"20020904123456", time.Date(2002, 9, 4, 12, 34, 56, 0, time.UTC), true},

		{"9999", time.Time{}, false},
		{"999", time.Time{}, false},
	}

	for _, tt := range tests {
		ret, ok := ParseDate(tt.in)
		if !ret.Equal(tt.out) || ok != tt.ok {
			t.Errorf("ParseDate(%q) = %s, %t; want %s, %t",
				tt.in, ret, ok, tt.out, tt.ok)
		}
	}
}

func TestParseOptions(t *testing.T) {
	var tests = []struct {
		in  string
		out []Option
		err error
	}{
		{"", nil, nil},
		{"  ", nil, nil},
		{
			"foo=false",
			[]Option{{"foo", []string{"false"}}},
			nil,
		},
		{
			"foo=value1,value2",
			[]Option{{"foo", []string{"value1", "value2"}}},
			nil,
		},
		{
			"foo=value1,value2 bar=value3",
			[]Option{
				{"foo", []string{"value1", "value2"}},
				{"bar", []string{"value3"}},
			},
			nil,
		},
		{
			"foo=value1,value2 bar='value3,value4'",
			[]Option{{"foo", []string{"value1", "value2"}},
				{"bar", []string{"value3,value4"}}},
			nil,
		},
		{
			"foo",
			[]Option{{"foo", nil}},
			nil,
		},
		{
			"nofoo",
			[]Option{{"nofoo", nil}},
			nil,
		},
		{
			"foo bar",
			[]Option{{"foo", nil}, {"bar", nil}},
			nil,
		},
		{
			"foo=value bar",
			[]Option{{"foo", []string{"value"}}, {"bar", nil}},
			nil,
		},
		{
			"foo bar=value",
			[]Option{{"foo", nil}, {"bar", []string{"value"}}},
			nil,
		},
		{
			"media-col={media-size={x-dimension=123 y-dimension=456}}",
			[]Option{{"media-col", []string{"{media-size={x-dimension=123 y-dimension=456}}"}}},
			nil,
		},
		{
			"{media-size={x-dimension=123 y-dimension=456}}",
			[]Option{{"media-size", []string{"{x-dimension=123 y-dimension=456}"}}},
			nil,
		},
		{
			"{x-dimension=123 y-dimension=456}",
			[]Option{
				{"x-dimension", []string{"123"}},
				{"y-dimension", []string{"456"}},
			},
			nil,
		},
		{
			"copies=123",
			[]Option{{"copies", []string{"123"}}},
			nil,
		},
		{
			"hue=-123",
			[]Option{{"hue", []string{"-123"}}},
			nil,
		},
		{
			"media=na-custom-foo.8000-10000",
			[]Option{{"media", []string{"na-custom-foo.8000-10000"}}},
			nil,
		},
		{
			`job-name=John\'s\ Really\040Nice\ Document`,
			[]Option{{"job-name", []string{`John's Really Nice Document`}}},
			nil,
		},
		{
			`job-name="John\'s Really Nice Document"`,
			[]Option{{"job-name", []string{`John's Really Nice Document`}}},
			nil,
		},
		{
			`document-name='Another \"Word\042 document.doc'`,
			[]Option{{"document-name", []string{`Another "Word" document.doc`}}},
			nil,
		},
		{
			"page-ranges=1-5",
			[]Option{{"page-ranges", []string{"1-5"}}},
			nil,
		},
		{
			"job-sheets=standard page-ranges=1-2,5-6,101-120 resolution=360dpi",
			[]Option{
				{
					"job-sheets",
					[]string{"standard"},
				},
				{
					"page-ranges",
					[]string{"1-2", "5-6", "101-120"},
				},
				{
					"resolution",
					[]string{"360dpi"},
				},
			},
			nil,
		},
		{
			`{foo="bar}"}`,
			[]Option{{"foo", []string{"bar}"}}},
			nil,
		},
		{
			`{foo="{bar}}"}`,
			[]Option{{"foo", []string{"{bar}}"}}},
			nil,
		},
		{
			`{foo="b\"ar}"}`,
			[]Option{{"foo", []string{`b"ar}`}}},
			nil,
		},
		{
			`field={foo="bar}"}`,
			[]Option{{"field", []string{`{foo="bar}"}`}}},
			nil,
		},
		{
			`field={foo="{bar}}"}`,
			[]Option{{"field", []string{`{foo="{bar}}"}`}}},
			nil,
		},
		{
			`field={foo="b\"ar}"}`,
			[]Option{{"field", []string{`{foo="b\"ar}"}`}}},
			nil,
		},

		{
			`field=  `,
			nil,
			&SyntaxError{Offset: 8, msg: "unexpected end of input"},
		},
		{
			`field={`,
			nil,
			&SyntaxError{Offset: 7, msg: "unexpected end of input"},
		},
		{
			`field="`,
			nil,
			&SyntaxError{Offset: 6, msg: "unexpected end of input"},
		},
		{
			`field=\23`,
			nil,
			&SyntaxError{Offset: 9, msg: "invalid octal number"},
		},
		{
			`field=,`,
			nil,
			&SyntaxError{Offset: 6, msg: "unexpected end of input"},
		},

		// go-fuzz tests
		{"foo=value1,", nil, &SyntaxError{Offset: 10, msg: "unexpected end of input"}},
		{"0=", nil, &SyntaxError{Offset: 2, msg: "unexpected end of input"}},
	}

	for _, tt := range tests {
		got, err := ParseOptions(tt.in)
		if !reflect.DeepEqual(got, tt.out) || !reflect.DeepEqual(err, tt.err) {
			t.Errorf("ParseOptions(%q) = %#v, %#v; want %#v, %#v",
				tt.in, got, err, tt.out, tt.err)
		}
	}
}

func TestOptionRealName(t *testing.T) {
	var tests = []struct {
		in  Option
		out string
	}{
		{Option{"foo", nil}, "foo"},
		{Option{"nofoo", nil}, "foo"},
		{Option{"nofoo", []string{"value"}}, "nofoo"},
		{Option{"foo", []string{"value"}}, "foo"},
	}

	for i, tt := range tests {
		name := tt.in.RealName()
		if name != tt.out {
			t.Errorf("%d: RealName() = %q, want %q", i, name, tt.out)
		}
	}
}

func TestOptionBool(t *testing.T) {
	var tests = []struct {
		in  Option
		out bool
	}{
		{Option{"foo", nil}, true},
		{Option{"nofoo", nil}, false},
		{Option{"foo", []string{"false"}}, false},
		{Option{"foo", []string{"true"}}, true},
		{Option{"foo", []string{"true", "true"}}, false},
		{Option{"foo", []string{"value"}}, false},
		{Option{"nofoo", []string{"false"}}, false},
		{Option{"nofoo", []string{"true"}}, true},
	}

	for i, tt := range tests {
		v := tt.in.Bool()
		if v != tt.out {
			t.Errorf("%d: Bool() = %t, want %t", i, v, tt.out)
		}
	}
}
