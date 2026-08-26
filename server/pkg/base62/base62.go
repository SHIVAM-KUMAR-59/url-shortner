package base62

import "fmt"

func Encode(id uint64) string {
	if id == 0 {
		return "0"
	}

	const characterSet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

	// Max capacity needed for uint64 in base62 is 11 bytes
	var buf [11]byte
	idx := len(buf)

	quotient := id
	for quotient > 0 {
		idx--
		buf[idx] = characterSet[quotient%62]
		quotient /= 62
	}

	// Slice the buffer from where data actually starts to the end
	return string(buf[idx:])
}

func Decode(url string) (uint64, error) {
	var id uint64

	for i := 0; i < len(url); i++ {
		char := url[i]
		var value uint64

		switch {
		case char >= '0' && char <= '9':
			value = uint64(char - '0') // '0'-'9' maps to 0-9
		case char >= 'a' && char <= 'z':
			value = uint64(char - 'a' + 10) // 'a'-'z' maps to 10-35
		case char >= 'A' && char <= 'Z':
			value = uint64(char - 'A' + 36) // 'A'-'Z' maps to 36-61
		default:
			return 0, fmt.Errorf("ErrInvalidCharacter: found character '%c' at index %d", char, i)
		}

		// Shift the base and add the character's value
		id = id*62 + value
	}

	return id, nil
}
