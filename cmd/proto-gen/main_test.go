package main

import "testing"

func TestCheckOutFlags(t *testing.T) {
	tests := []struct {
		name    string
		flags   outFlags
		wantErr bool
	}{
		{"scan only", outFlags{}, false},
		{"generate proto", outFlags{
			goPackage: "g", out: "f.proto", protoPackage: "p", service: "S",
		}, false},
		{"generate converters", outFlags{
			convOut: "f.go", convPackage: "c", goPackage: "g",
			goTypesPackage: "t",
		}, false},
		{"generate both", outFlags{
			convOut: "f.go", convPackage: "c", goPackage: "g",
			goTypesPackage: "t", out: "f.proto", protoPackage: "p",
			service: "S",
		}, false},
		{"Go package alone", outFlags{goPackage: "g"}, true},
		{"package without out", outFlags{protoPackage: "p"}, true},
		{"service without out", outFlags{service: "S"}, true},
		{"conv package without convOut", outFlags{convPackage: "c"}, true},
		{"types package without convOut", outFlags{goTypesPackage: "t"}, true},
		{"out without anything", outFlags{out: "f.proto"}, true},
		{"out without Go package", outFlags{
			out: "f.proto", protoPackage: "p", service: "S",
		}, true},
		{"out without package", outFlags{
			goPackage: "g", out: "f.proto", service: "S",
		}, true},
		{"out without service", outFlags{
			goPackage: "g", out: "f.proto", protoPackage: "p",
		}, true},
		{"convOut without anything", outFlags{convOut: "f.go"}, true},
		{"convOut without Go package", outFlags{
			convOut: "f.go", convPackage: "c", goTypesPackage: "t",
		}, true},
		{"convOut without package", outFlags{
			convOut: "f.go", goPackage: "g", goTypesPackage: "t",
		}, true},
		{"convOut without types package", outFlags{
			convOut: "f.go", convPackage: "c", goPackage: "g",
		}, true},
	}
	for _, test := range tests {
		err := checkOutFlags(test.flags)
		if (err != nil) != test.wantErr {
			t.Errorf("%s: got error %v, want error %v", test.name, err,
				test.wantErr)
		}
	}
}
