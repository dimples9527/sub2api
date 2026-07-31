package service

import "testing"

func TestNormalizeSupplierNewAPIKeyStatus(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want string
	}{
		{name: "int active", in: 1, want: "active"},
		{name: "int disabled", in: 2, want: "disabled"},
		{name: "int expired", in: 3, want: "expired"},
		{name: "int quota", in: 4, want: "quota_exhausted"},
		{name: "float active", in: float64(1), want: "active"},
		{name: "float disabled", in: float64(2), want: "disabled"},
		{name: "string 1", in: "1", want: "active"},
		{name: "string 2", in: "2", want: "disabled"},
		{name: "string active", in: "active", want: "active"},
		{name: "string disabled", in: "disabled", want: "disabled"},
		{name: "unknown int", in: 9, want: "unknown"},
		{name: "nil", in: nil, want: "unknown"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeSupplierNewAPIKeyStatus(tc.in)
			if got != tc.want {
				t.Fatalf("normalizeSupplierNewAPIKeyStatus(%v) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestNormalizeSupplierSub2APIKeyStatus(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{in: "active", want: "active"},
		{in: "inactive", want: "disabled"},
		{in: "disabled", want: "disabled"},
		{in: "expired", want: "expired"},
		{in: "quota_exhausted", want: "quota_exhausted"},
		{in: " INACTIVE ", want: "disabled"},
		{in: "weird", want: "unknown"},
		{in: "", want: "unknown"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got := normalizeSupplierSub2APIKeyStatus(tc.in)
			if got != tc.want {
				t.Fatalf("normalizeSupplierSub2APIKeyStatus(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}