package hacks

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

// Parameter is a param value represented as a string.
type Parameter string

// AsString ...
func (p Parameter) AsString() (string, error) {
	return string(p), nil
}

// AsBool ...
func (p Parameter) AsBool() (bool, error) {
	return strconv.ParseBool(string(p))
}

// AsInt64 ...
func (p Parameter) AsInt64() (int64, error) {
	return strconv.ParseInt(string(p), 10, 32)
}

// IP ...
func (p Parameter) IP() (net.IP, error) {
	ip := net.ParseIP(string(p))
	if ip == nil {
		return nil, fmt.Errorf("invalid IP address")
	}

	return ip, nil
}

// Spec describes a parameterized hack to run.
type Spec struct {
	Hack       string
	Parameters map[string]Parameter
}

// ParseSpec parses a policy specification string. A policy
// specification string is of the form:
//
//      OPERATION[:PARAM=VALUE[,PARAM=VALUE]...]
func ParseSpec(specString string) (Spec, error) {
	if specString == "" {
		return Spec{}, fmt.Errorf("empty policy specification")
	}

	spec := Spec{
		Parameters: make(map[string]Parameter),
	}

	// Outer syntax is "$OP:$ARGS".
	parts := strings.SplitN(specString, ":", 2)
	spec.Hack = parts[0]

	switch len(parts) {
	case 1:
		// Just an operation, with no policy args.
		return spec, nil
	case 2:
		for _, arg := range strings.Split(parts[1], ",") {
			parts := strings.SplitN(arg, "=", 2)
			switch len(parts) {
			case 1:
				// If no parameter is specified, it is implicitly a boolean flag.
				spec.Parameters[parts[0]] = Parameter("true")
			case 2:
				spec.Parameters[parts[0]] = Parameter(parts[1])
			}
		}

		return spec, nil
	default:
		panic(fmt.Sprintf("invalid spec component %q", parts))
	}
}
