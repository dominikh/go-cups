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
		out, remainder := parseName(tt.in)
		if out != tt.out || remainder != tt.remainder {
			t.Errorf("parseName(%q) = %q, %q; want %q, %q", tt.in, out, remainder, tt.out, tt.remainder)
		}
	}
}

func TestSkipSpace(t *testing.T) {
	var tests = []struct {
		in  string
		out string
	}{
		{"", ""},
		{" ", ""},
		{"  ", ""},
		{"foo", "foo"},
		{" foo ", "foo "},
		{"\tfoo ", "foo "},
	}

	for _, tt := range tests {
		out := consumeSpace(tt.in)
		if out != tt.out {
			t.Errorf("consumeSpace(%q) = %q, want %q", tt.in, out, tt.out)
		}
	}
}

func TestStrings(t *testing.T) {
	var tests = []struct {
		in        string
		match     string
		remainder string
		quoted    bool
		ok        bool
	}{
		// Quoted
		{`"test"`, `test`, ``, true, true},
		{`'test'`, `test`, ``, true, true},
		{`"te\"st"`, `te"st`, ``, true, true},
		{`'te\'st'`, `te'st`, ``, true, true},
		{`"te'st"`, `te'st`, ``, true, true},
		{`"te\'st"`, `te'st`, ``, true, true},
		{`'te"st'`, `te"st`, ``, true, true},
		{`'te\"st'`, `te"st`, ``, true, true},
		{`"te\\st"`, `te\st`, ``, true, true},
		{`"\170"`, `x`, ``, true, true},
		{`"\1705"`, `x5`, ``, true, true},
		{`"!@#$%"`, `!@#$%`, ``, true, true},
		{`"test"moredata`, `test`, `moredata`, true, true},
		{`test`, ``, ``, true, false},
		{`"test`, ``, ``, true, false},
		{``, ``, ``, true, false},
		{`"\27"`, ``, ``, true, false},
		{"'\x00'", ``, ``, true, false},

		// Unquoted
		{`test`, `test`, ``, false, true},
		{`te\ st`, `te st`, ``, false, true},
		{`te st`, `te`, ` st`, false, true},
		{`te\"st`, `te"st`, ``, false, true},
		{`te\'st`, `te'st`, ``, false, true},
		{`te'st`, ``, ``, false, false},
		{`te"st`, ``, ``, false, false},
		{`te\\st`, `te\st`, ``, false, true},
		{`\170`, `x`, ``, false, true},
		{`\1705`, `x5`, ``, false, true},
		{`!@#$%`, `!@#$%`, ``, false, true},
		{`"test`, ``, ``, false, false},
	}

	for _, tt := range tests {
		match, remainder, ok := parseString(tt.in, tt.quoted)
		if match != tt.match ||
			remainder != tt.remainder ||
			ok != tt.ok {

			t.Errorf("parseQuotedValue(%q, %t) = %q, %q, %t; want %q, %q, %t",
				tt.in, tt.quoted,
				match, remainder, ok,
				tt.match, tt.remainder, tt.ok)
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
		ok  bool
	}{
		{
			"foo=false",
			[]Option{{"foo", []string{"false"}}},
			true,
		},
		{
			"foo=value1,value2",
			[]Option{{"foo", []string{"value1", "value2"}}},
			true,
		},
		{
			"foo=value1,value2 bar=value3",
			[]Option{
				{"foo", []string{"value1", "value2"}},
				{"bar", []string{"value3"}},
			},
			true,
		},
		{
			"foo=value1,value2 bar='value3,value4'",
			[]Option{{"foo", []string{"value1", "value2"}},
				{"bar", []string{"value3,value4"}}},
			true,
		},
		{
			"foo",
			[]Option{{"foo", nil}},
			true,
		},
		{
			"nofoo",
			[]Option{{"nofoo", nil}},
			true,
		},
		{
			"foo bar",
			[]Option{{"foo", nil}, {"bar", nil}},
			true,
		},
		{
			"foo=value bar",
			[]Option{{"foo", []string{"value"}}, {"bar", nil}},
			true,
		},
		{
			"foo bar=value",
			[]Option{{"foo", nil}, {"bar", []string{"value"}}},
			true,
		},
		{
			"media-col={media-size={x-dimension=123 y-dimension=456}}",
			[]Option{{"media-col", []string{"{media-size={x-dimension=123 y-dimension=456}}"}}},
			true,
		},
		{
			"{media-size={x-dimension=123 y-dimension=456}}",
			[]Option{{"media-size", []string{"{x-dimension=123 y-dimension=456}"}}},
			true,
		},
		{
			"{x-dimension=123 y-dimension=456}",
			[]Option{
				{"x-dimension", []string{"123"}},
				{"y-dimension", []string{"456"}},
			},
			true,
		},
		{
			"copies=123",
			[]Option{{"copies", []string{"123"}}},
			true,
		},
		{
			"hue=-123",
			[]Option{{"hue", []string{"-123"}}},
			true,
		},
		{
			"media=na-custom-foo.8000-10000",
			[]Option{{"media", []string{"na-custom-foo.8000-10000"}}},
			true,
		},
		{
			`job-name=John\'s\ Really\040Nice\ Document`,
			[]Option{{"job-name", []string{`John's Really Nice Document`}}},
			true,
		},
		{
			`job-name="John\'s Really Nice Document"`,
			[]Option{{"job-name", []string{`John's Really Nice Document`}}},
			true,
		},
		{
			`document-name='Another \"Word\042 document.doc'`,
			[]Option{{"document-name", []string{`Another "Word" document.doc`}}},
			true,
		},
		{
			"page-ranges=1-5",
			[]Option{{"page-ranges", []string{"1-5"}}},
			true,
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
			true,
		},
		{
			`{foo="bar}"}`,
			[]Option{{"foo", []string{"bar}"}}},
			true,
		},
		{
			`{foo="{bar}}"}`,
			[]Option{{"foo", []string{"{bar}}"}}},
			true,
		},
		{
			`{foo="b\"ar}"}`,
			[]Option{{"foo", []string{`b"ar}`}}},
			true,
		},

		// go-fuzz tests
		{"foo=value1,", nil, false},
		{"0=", nil, false},
	}

	for _, tt := range tests {
		got, ok := ParseOptions(tt.in)
		if !reflect.DeepEqual(got, tt.out) || ok != tt.ok {
			t.Errorf("ParseOptions(%q) = %#v, %t; want %#v, %t",
				tt.in, got, ok, tt.out, tt.ok)
		}
	}
}
