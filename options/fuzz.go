// +build gofuzz

package options

func FuzzQuotedValue(data []byte) int {
	_, _, ok := parseString(string(data), true)
	if ok {
		return 1
	}
	return 0
}

func FuzzStringValue(data []byte) int {
	_, _, ok := parseString(string(data), false)
	if ok {
		return 1
	}
	return 0
}

func FuzzConsumeSpace(data []byte) int {
	ret := consumeSpace(string(data))
	if ret != string(data) {
		return 1
	}
	return 0
}

func FuzzParseResolution(data []byte) int {
	_, ok := ParseResolution(string(data))
	if ok {
		return 1
	}
	return 0
}

func FuzzParseOptions(data []byte) int {
	_, ok := ParseOptions(string(data))
	if ok {
		return 1
	}
	return 0
}
