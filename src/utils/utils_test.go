package utils

import "testing"

func TestGetNodeAndEndpoint(t *testing.T) {
	node,endpoint := GetNodeAndEndpoint("abcd")
	if node != "abcd" || endpoint != "" {
		t.Error("Wrong node name")
	}
	node,endpoint = GetNodeAndEndpoint("99_4")
	if node != "99" || endpoint != "4" {
		t.Error("Wrong node or endpoint")
	}


}
