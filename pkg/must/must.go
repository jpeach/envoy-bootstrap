package must

import "net"

// StringSlice ...
func StringSlice(s []string, err error) []string {
	if err != nil {
		panic(err.Error())
	}

	return s
}

// String ...
func String(s string, err error) string {
	if err != nil {
		panic(err.Error())
	}

	return s
}

// Int64 ...
func Int64(i int64, err error) int64 {
	if err != nil {
		panic(err.Error())
	}

	return i
}

// Bytes ...
func Bytes(b []byte, err error) []byte {
	if err != nil {
		panic(err.Error())
	}

	return b
}

// IP ...
func IP(ip net.IP, err error) net.IP {
	if err != nil {
		panic(err.Error())
	}

	return ip
}
