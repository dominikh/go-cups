// Package options implements parsing of CUPS's text options, also
// known as PAPI text attributes.
package options

// TODO(dh): don't return a boolean error, instead describe where in
// the input the error occured

import (
	"math"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// The parser implemented in this file parses PAPI attributes/text
// options. Its specification can be found in
// http://www.opensource.apple.com/source/cups/cups-136.9/cups/standards/papi-1.0.pdf
// on page 120.

type Range struct {
	Start int
	End   int
}

type Resolution struct {
	X int
	Y int
}

type Option struct {
	Name   string
	Values []string
}

type SyntaxError struct {
	Offset int
	msg    string
}

func (err *SyntaxError) Error() string {
	return err.msg
}

type decoder struct {
	input  string
	offset int
}

func (d *decoder) eof() bool {
	return d.offset >= len(d.input)
}

func (d *decoder) string() string {
	if d.eof() {
		return ""
	}
	return d.input[d.offset:]
}

func (d *decoder) byte() byte {
	if d.eof() {
		return 0
	}
	return d.input[d.offset]
}

func ParseOptions(s string) (v []Option, err error) {
	if len(s) == 0 {
		return nil, nil
	}
	if len(s) >= 2 && s[0] == '{' && s[len(s)-1] == '}' {
		s = s[1 : len(s)-1]
	}
	d := &decoder{input: s}
	var option Option
	for !d.eof() {
		if option.Name != "" {
			v = append(v, option)
			option = Option{}
		}
		name := d.parseName()
		if name == "" {
			break
		}
		option.Name = name
		d.consumeSpace()
		if d.eof() {
			break
		}
		if d.byte() == '=' {
			// this is a value option
			d.offset++
			var value string
		valueLoop:
			for !d.eof() {
				value, err = d.parseValue()
				if err != nil {
					return nil, err
				}
				option.Values = append(option.Values, value)
				if !d.eof() {
					if d.byte() == ',' {
						if len(d.string()) == 1 {
							return nil, &SyntaxError{d.offset, "unexpected end of input"}
						}
						d.offset++
					} else if d.byte() == ' ' {
						break valueLoop
					}
				}
			}
			if len(option.Values) == 0 {
				// saw an equal sign but no value -> invalid
				return nil, &SyntaxError{d.offset, "unexpected end of input"}
			}
		} else {
			if option.Name != "" {
				v = append(v, option)
				option = Option{}
			}
		}
	}
	if option.Name != "" {
		v = append(v, option)
		option = Option{}
	}
	return v, nil
}

func (d *decoder) parseValue() (value string, err error) {
	d.consumeSpace()
	if d.eof() {
		return "", &SyntaxError{d.offset, "unexpected end of input"}
	}
	switch d.byte() {
	case '{':
		return d.extractCollection()
	case '\'', '"':
		v, err := d.parseString(true)
		if err != nil {
			return "", err
		}
		return v, nil
	default:
		v, err := d.parseString(false)
		if err != nil {
			return "", err
		}
		return v, nil
	}
}

func (d *decoder) extractCollection() (string, error) {
	depth := 0
	escape := false
	var quote byte
	start := d.offset
loop:
	for ; !d.eof(); d.offset++ {
		c := d.byte()
		switch c {
		case '{':
			if !escape && quote == 0 {
				depth++
			}
		case '}':
			if !escape && quote == 0 {
				depth--
			}
			if depth == 0 {
				d.offset++
				return d.input[start:d.offset], nil
			}
		case '\\':
			if !escape {
				escape = true
				continue loop
			}
		case '\'', '"':
			if escape {
				break
			}
			if c == quote {
				quote = 0
			} else if quote == 0 {
				quote = c
			}

		}
		escape = false
	}
	return "", &SyntaxError{d.offset, "unexpected end of input"}
}

func (d *decoder) parseOctal(s string) (string, error) {
	if len(s) != 3 {
		return "", &SyntaxError{d.offset, "invalid octal number"}
	}
	n, err := strconv.ParseInt(s, 8, 32)
	if err != nil {
		return "", &SyntaxError{d.offset, err.Error()}
	}
	return string(n), nil
}

// ParseBool interprets s as a boolean value. "yes" and "true"
// evaluate to true, while "no" and "false" evaluate to false. Other
// values are not permitted.
func ParseBool(s string) (v bool, ok bool) {
	if s == "yes" || s == "no" || s == "true" || s == "false" {
		return s == "yes" || s == "true", true
	}
	return false, false
}

// ParseNumber interprets s as a whole number, optionally with a sign.
func ParseNumber(s string) (v int, ok bool) {
	if !isNumber(s) {
		return 0, false
	}
	n, err := strconv.ParseInt(s, 10, 32)
	return int(n), err == nil
}

// ParseRange interprets s as a range consisting of two whole,
// positive numbers without signs.
func ParseRange(s string) (v Range, ok bool) {
	parts := strings.SplitN(s, "-", 2)
	if len(parts) != 2 || !isDigits(parts[0]) || !isDigits(parts[1]) {
		return Range{}, false
	}
	n1, _ := strconv.ParseInt(parts[0], 10, 32)
	n2, _ := strconv.ParseInt(parts[1], 10, 32)
	return Range{int(n1), int(n2)}, true
}

// ParseResolution interprets s as a resolution. Valid inputs look
// like "600dpi", "600x300dpi", "600dpc" or "600x300dpc". Resolutions
// in dots per centimeter will be converted to dots per inch.
func ParseResolution(s string) (v Resolution, ok bool) {
	if len(s) < 4 {
		return Resolution{}, false
	}
	suffix := s[len(s)-3:]
	prefix := s[:len(s)-3]
	if suffix != "dpi" && suffix != "dpc" {
		return Resolution{}, false
	}
	parts := strings.SplitN(prefix, "x", 2)
	s1 := parts[0]
	s2 := s1
	if len(parts) == 2 {
		s2 = parts[1]
	}
	if !isDigits(s1) || !isDigits(s2) {
		return Resolution{}, false
	}
	n1, _ := strconv.ParseInt(s1, 10, 32)
	n2, _ := strconv.ParseInt(s2, 10, 32)

	if suffix == "dpi" {
		return Resolution{int(n1), int(n2)}, true
	}
	return Resolution{
		int(math.Floor(float64(n1)*2.54 + 0.5)),
		int(math.Floor(float64(n2)*2.54 + 0.5)),
	}, true
}

// ParseDate interprets s as a date/time. Valid formats are:
// 	- HHmm
// 	- HHmmss
// 	- yyyyMMdd
// 	- yyyyMMddHHmm
// 	- yyyyMMddHHmmss
func ParseDate(s string) (v time.Time, ok bool) {
	var t time.Time
	var err error
	switch len(s) {
	case 4:
		t, err = time.Parse("1504", s)
	case 6:
		t, err = time.Parse("150405", s)
	case 8:
		t, err = time.Parse("20060102", s)
	case 12:
		t, err = time.Parse("200601021504", s)
	case 14:
		t, err = time.Parse("20060102150405", s)
	default:
		return time.Time{}, false
	}
	return t, err == nil
}

func isNumber(s string) bool {
	if len(s) == 0 {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] == '-' || s[i] == '+' {
			if i != 0 {
				return false
			}
			continue
		}
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func isDigits(s string) bool {
	if len(s) == 0 {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func (d *decoder) parseString(quoted bool) (string, error) {
	if d.eof() {
		return "", &SyntaxError{d.offset, "unexpected end of input"}
	}
	if quoted && len(d.string()) < 2 {
		return "", &SyntaxError{d.offset, "unexpected end of input"}
	}
	var i int
	var v string
	var escape bool
	var octal string
	var open byte
	if quoted {
		open = d.byte()
		d.offset++
		if open != '"' && open != '\'' {
			return "", &SyntaxError{d.offset, "improperly quoted string"}
		}
	}
loop:
	for ; !d.eof(); d.offset++ {
		c := d.byte()
		if octal != "" && (c < '0' || c > '7' || len(octal) == 3) {
			escape = false
			n, err := d.parseOctal(octal)
			if err != nil {
				return "", err
			}
			v += n
			octal = ""
		}
		switch c {
		case '\\':
			if escape {
				v += string(c)
			} else {
				escape = true
				continue loop
			}
		case '"', '\'':
			if !escape {
				if quoted && c == open {
					break loop
				}
				if !quoted {
					// unquoted string, unescaped quote -> invalid
					return "", &SyntaxError{d.offset, "unescaped quote in unquoted string"}
				}
			}
			v += string(c)
		case ' ':
			if quoted || escape {
				v += string(c)
			} else {
				i--
				break loop
			}
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			if c < '8' && (octal != "" || escape) {
				octal += string(c)
			} else {
				v += string(c)
			}
		case ',':
			if quoted {
				v += string(c)
			} else {
				// commas separate multiple values; even if the spec
				// permits commas in unquoted strings.
				i--
				break loop
			}
		default:
			if c == 0x21 ||
				(c >= 0x23 && c <= 0x26) ||
				(c >= 0x28 && c <= 0x5b) ||
				(c >= 0x5d && c <= 0x7e) ||
				(c >= 0xa0 && c <= 0xff) {

				v += string(c)
			} else {
				return "", &SyntaxError{d.offset, "invalid byte in string"}
			}
		}
		escape = false
	}
	if quoted && d.eof() {
		// didn't see a closing quote
		return "", &SyntaxError{d.offset, "unexpected end of input"}
	}
	if quoted {
		d.offset++
	}
	if octal != "" {
		n, err := d.parseOctal(octal)
		if err != nil {
			return "", err
		}
		v += n
	}
	return v, nil
}

func (d *decoder) parseName() string {
	d.consumeSpace()
	start := d.offset
	for ; !d.eof(); d.offset++ {
		c := d.byte()
		if unicode.IsSpace(rune(c)) || c == '=' {
			break
		}
	}
	return d.input[start:d.offset]
}

func (d *decoder) consumeSpace() {
	if d.eof() {
		return
	}
	idx := strings.IndexFunc(d.string(), func(r rune) bool { return !unicode.IsSpace(r) })
	if idx == -1 {
		d.offset = len(d.input)
	} else {
		d.offset += idx
	}
}
