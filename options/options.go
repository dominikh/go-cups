// Package options implements parsing of CUPS's text options, also
// known as PAPI text attributes.
package options

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

func ParseOptions(s string) (v []Option, ok bool) {
	if len(s) == 0 {
		return nil, true
	}
	if len(s) >= 2 && s[0] == '{' && s[len(s)-1] == '}' {
		s = s[1 : len(s)-1]
	}
	var option Option
	for len(s) > 0 {
		if option.Name != "" {
			v = append(v, option)
			option = Option{}
		}
		var name string
		name, s = parseName(s)
		if name == "" {
			break
		}
		option.Name = name
		s = consumeSpace(s)
		if len(s) == 0 {
			break
		}
		if s[0] == '=' {
			// this is a value option
			var value string
			s = s[1:]
		valueLoop:
			for len(s) > 0 {
				value, s = parseValue(s)
				option.Values = append(option.Values, value)
				if len(s) > 0 {
					if s[0] == ',' {
						if len(s) == 1 {
							return nil, false
						}
						s = s[1:]
					} else if s[0] == ' ' {
						break valueLoop
					}
				}
			}
			if len(option.Values) == 0 {
				// saw an equal sign but no value -> invalid
				return nil, false
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
	return v, true
}

func parseValue(s string) (value string, remainder string) {
	s = consumeSpace(s)
	if len(s) == 0 {
		// TODO invalid option string, error
	}
	switch s[0] {
	case '{':
		// TODO find closing }, return string
		return extractCollection(s)
	case '\'', '"':
		v, remainder, ok := parseString(s, true)
		if !ok {
			// TODO bubble up failure
		}
		return v, remainder
	default:
		v, remainder, ok := parseString(s, false)
		if !ok {
			// TODO bubble up failure
		}
		return v, remainder
	}
}

func extractCollection(s string) (value string, remainder string) {
	depth := 0
	escape := false
	var quote byte
loop:
	for i := 0; i < len(s); i++ {
		c := s[i]
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
				return s[:i+1], s[i+1:]
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
	return s, ""
}

func parseOctal(s string) (string, bool) {
	if len(s) != 3 {
		return "", false
	}
	n, err := strconv.ParseInt(s, 8, 32)
	if err != nil {
		return "", false
	}
	return string(n), true
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
//   - HHmm
//   - HHmmss
//   - yyyyMMdd
//   - yyyyMMddHHmm
//   - yyyyMMddHHmmss
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

func parseString(s string, quoted bool) (match string, remainder string, ok bool) {
	if len(s) == 0 {
		return "", "", false
	}
	if quoted && len(s) < 2 {
		return "", "", false
	}
	var i int
	var v string
	var escape bool
	var octal string
	var open byte
	if quoted {
		open = s[0]
		s = s[1:]
		if open != '"' && open != '\'' {
			return "", "", false
		}
	}
loop:
	for i = 0; i < len(s); i++ {
		c := s[i]
		if octal != "" && (c < '0' || c > '7' || len(octal) == 3) {
			escape = false
			n, ok := parseOctal(octal)
			if !ok {
				return "", "", false
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
					return "", "", false
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
				return "", "", false
			}
		}
		escape = false
	}
	if quoted && i >= len(s) {
		// didn't see a closing quote
		return "", "", false
	}
	if octal != "" {
		n, ok := parseOctal(octal)
		if !ok {
			return "", "", false
		}
		v += n
	}
	if i == len(s) {
		return v, "", true
	}
	return v, s[i+1:], true
}

func parseName(s string) (match string, remainder string) {
	s = consumeSpace(s)
	var i int
	for i = 0; i < len(s); i++ {
		if unicode.IsSpace(rune(s[i])) || s[i] == '=' {
			break
		}
	}
	return s[:i], s[i:]
}

func consumeSpace(s string) (remainder string) {
	idx := strings.IndexFunc(s, func(r rune) bool { return !unicode.IsSpace(r) })
	if idx == -1 {
		idx = len(s)
	}
	return s[idx:]
}
