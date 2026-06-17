package systemdsyntax

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

func SplitItems(value string) ([]string, error) {
	var items []string
	for i := 0; ; {
		for i < len(value) && unicode.IsSpace(rune(value[i])) {
			i++
		}
		if i >= len(value) {
			return items, nil
		}
		item, next, err := readItem(value, i)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
		i = next
	}
}

func readItem(value string, start int) (string, int, error) {
	var b strings.Builder
	quoted := false
	quote := byte(0)
	for i := start; i < len(value); i++ {
		c := value[i]
		if quoted {
			if c == quote {
				quoted = false
				if i+1 < len(value) && !unicode.IsSpace(rune(value[i+1])) {
					return "", 0, fmt.Errorf("quoted item must be followed by whitespace or end of line")
				}
				continue
			}
			if c == '\\' {
				r, next, err := readEscape(value, i)
				if err != nil {
					return "", 0, err
				}
				b.WriteRune(r)
				i = next - 1
				continue
			}
			b.WriteByte(c)
			continue
		}

		if unicode.IsSpace(rune(c)) {
			return b.String(), i, nil
		}
		if (c == '\'' || c == '"') && i == start {
			quoted = true
			quote = c
			continue
		}
		if c == '\\' {
			r, next, err := readEscape(value, i)
			if err != nil {
				return "", 0, err
			}
			b.WriteRune(r)
			i = next - 1
			continue
		}
		b.WriteByte(c)
	}
	if quoted {
		return "", 0, fmt.Errorf("unterminated quoted item")
	}
	return b.String(), len(value), nil
}

func readEscape(value string, start int) (rune, int, error) {
	if start+1 >= len(value) {
		return 0, 0, fmt.Errorf("trailing backslash")
	}
	switch c := value[start+1]; c {
	case 'a':
		return '\a', start + 2, nil
	case 'b':
		return '\b', start + 2, nil
	case 'f':
		return '\f', start + 2, nil
	case 'n':
		return '\n', start + 2, nil
	case 'r':
		return '\r', start + 2, nil
	case 't':
		return '\t', start + 2, nil
	case 'v':
		return '\v', start + 2, nil
	case '\\', '"', '\'':
		return rune(c), start + 2, nil
	case 's':
		return ' ', start + 2, nil
	case 'x':
		return readHexEscape(value, start, 2)
	case 'u':
		return readHexEscape(value, start, 4)
	case 'U':
		return readHexEscape(value, start, 8)
	default:
		if c >= '0' && c <= '7' {
			return readOctalEscape(value, start)
		}
		return 0, 0, fmt.Errorf("unsupported escape \\%c", c)
	}
}

func readHexEscape(value string, start, digits int) (rune, int, error) {
	end := start + 2 + digits
	if end > len(value) {
		return 0, 0, fmt.Errorf("short hex escape")
	}
	n, err := strconv.ParseInt(value[start+2:end], 16, 32)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid hex escape: %w", err)
	}
	return rune(n), end, nil
}

func readOctalEscape(value string, start int) (rune, int, error) {
	end := start + 1
	for end < len(value) && end < start+4 && value[end] >= '0' && value[end] <= '7' {
		end++
	}
	n, err := strconv.ParseInt(value[start+1:end], 8, 32)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid octal escape: %w", err)
	}
	return rune(n), end, nil
}
