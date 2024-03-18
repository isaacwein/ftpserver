package tools

import "unicode"

type printableType interface {
	~string | ~[]rune | ~[]byte
}

func IsPrintable[T printableType](v T) string {
	var result []rune

	switch v := any(v).(type) {
	case string:
		for _, r := range v {
			if unicode.IsPrint(r) {
				result = append(result, r)
			}
		}
	case []rune:
		for _, r := range v {
			if unicode.IsPrint(r) {
				result = append(result, r)
			}
		}
	case []byte:
		for _, r := range v {
			if unicode.IsPrint(rune(r)) {
				result = append(result, rune(r))
			}
		}
	}
	return string(result)
}
