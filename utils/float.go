package utils

import "strconv"

func FloatFromBytes(b []byte) float64 {
	f, _ := strconv.ParseFloat(string(b), 64)
	return f
}
func FLoatToBytes(f float64) []byte {
	return []byte(strconv.FormatFloat(f, 'f', -1, 64))
}
