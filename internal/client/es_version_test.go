package client

import (
	"testing"
)

func Test_EsVersion_IsAtLeastVersion(t *testing.T) {
	tests := []struct {
		v1   EventStoreVersion
		v2   string
		want bool
	}{
		{v1: "20.6.0.0", v2: "20.6.0.0", want: true},
		{v1: "20.6.0.0", v2: "20.6.1.0", want: false},
		{v1: "20.6.0.0", v2: "21.6.0.0", want: false},
		{v1: "20.6.0.0", v2: "20.5.0.0", want: true},
		{v1: "20.6.0.0", v2: "19.6.0.0", want: true},
	}

	for _, test := range tests {
		if got := test.v1.IsAtLeastVersion(test.v2); got != test.want {
			t.Errorf("Expected %s to be at least %s: %v", test.v1, test.v2, test.want)
		}
	}
}

func Test_EsVersion_IsVersionLowerThan(t *testing.T) {
	tests := []struct {
		v1   EventStoreVersion
		v2   string
		want bool
	}{
		{v1: "20.6.0.0", v2: "20.6.0.0", want: false},
		{v1: "20.6.0.0", v2: "20.6.1.0", want: true},
		{v1: "20.6.0.0", v2: "21.6.0.0", want: true},
		{v1: "20.6.0.0", v2: "20.5.0.0", want: false},
		{v1: "20.6.0.0", v2: "19.6.0.0", want: false},
	}

	for _, test := range tests {
		if got := test.v1.IsVersionLowerThan(test.v2); got != test.want {
			t.Errorf("Expected %s to be lower than %s: %v", test.v1, test.v2, test.want)
		}
	}
}
