package scope

import (
	_ "embed"
	"encoding/json"
	"time"
)

//go:embed data/prograscope.json
var prograscopeData []byte

//go:embed data/alcoscope.json
var alcoscopeData []byte

var (
	prograscopeHoroscopes []string
	alcoscopeHoroscopes   []string
)

func init() {
	_ = json.Unmarshal(prograscopeData, &prograscopeHoroscopes)
	_ = json.Unmarshal(alcoscopeData, &alcoscopeHoroscopes)
}

func Prograscope(senderID int64) string {
	userDigit := lastDigit(senderID)
	now := time.Now()
	index := wrapIndex(userDigit*now.Day()*int(now.Month()), len(prograscopeHoroscopes))
	return "Прогноз для программиста #" + itoa(userDigit) + ": " + prograscopeHoroscopes[index]
}

func Alcoscope(senderID int64) string {
	userDigit := lastDigit(senderID)
	now := time.Now()
	index := wrapIndex(userDigit*now.Day(), len(alcoscopeHoroscopes))
	return "Прогноз #" + itoa(userDigit) + ": " + alcoscopeHoroscopes[index]
}

func wrapIndex(num, length int) int {
	if num < 0 {
		return 0
	}
	if num < length {
		return num
	}
	return wrapIndex(num-length, length)
}

func lastDigit(num int64) int {
	if num < 0 {
		num = -num
	}
	return int(num % 10)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
