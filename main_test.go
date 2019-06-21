package main

import (
	"reflect"
	"testing"
)

func TestNM(t *testing.T) {
	var tests = []struct {
		data      string
		splitWant []string
		key       string
	}{
		{"1001640         96 T _x_cgo_callers", []string{"1001640", "96", "T", "_x_cgo_callers"}, "_x_cgo_callers"},
		{"16b9450         16 R crypto.statictmp_0", []string{"16b9450", "16", "R", "crypto.statictmp_0"}, "crypto"},
		{"113b3e0        208 T crypto/cipher.xorBytes", []string{"113b3e0", "208", "T", "crypto/cipher.xorBytes"}, "crypto"},
		{"103cc90         32 T runtime.gcd", []string{"103cc90", "32", "T", "runtime.gcd"}, "runtime"},
		{"135ff60         64 T vendor/golang.org/x/net/ipv4.parseTTL", []string{"135ff60", "64", "T", "vendor/golang.org/x/net/ipv4.parseTTL"}, "vendor/golang.org/x/net"},
	}

	for _, test := range tests {
		ss := split(test.data)
		if !reflect.DeepEqual(ss, test.splitWant) {
			t.Errorf("want: %v, got: %v", test.splitWant, ss)
		}

		key := keygen(ss[3], 2)
		if key != test.key {
			t.Errorf("want: %v, got: %v", test.key, key)
		}
	}
}
