package main

import (
	"os"
	"strings"
	"testing"
)

const (
	testTypesPath = "github.com/Cloud-Foundations/Dominator/proto/hypervisor"
	testGoPackage = testTypesPath + "/grpc"
)

func TestGenerateConvertersGolden(t *testing.T) {
	_, infos, named := scanTestdata(t)
	convInfos, err := scanTypes("testdata/convtypes", nil)
	if err != nil {
		t.Fatal(err)
	}
	convNamed, err := scanNamedTypes("testdata/convtypes")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		goldenFile string
		config     convConfig
		infos      []typeInfo
		named      map[string]string
	}{
		{"testdata/expected_converters.go.txt", convConfig{
			GoPackage:    testGoPackage,
			Package:      "grpcd",
			TypesPackage: testTypesPath,
		}, infos, named},
		{"testdata/expected_convtypes.go.txt", convConfig{
			GoPackage:    testGoPackage,
			Package:      "grpcd",
			TypesPackage: testTypesPath + "/convtypes",
		}, convInfos, convNamed},
	}
	for _, test := range tests {
		got, err := generateConverters(test.config, test.infos, test.named)
		if err != nil {
			t.Fatal(err)
		}
		if *update {
			err := os.WriteFile(test.goldenFile, []byte(got), 0o644)
			if err != nil {
				t.Fatal(err)
			}
		}
		want, err := os.ReadFile(test.goldenFile)
		if err != nil {
			t.Fatal(err)
		}
		if got != string(want) {
			t.Errorf("generated converters differ from %s (use -update):\n%s",
				test.goldenFile, got)
		}
	}
}

func TestGenerateConvertersConfig(t *testing.T) {
	_, infos, named := scanTestdata(t)
	valid := convConfig{
		GoPackage:    testGoPackage,
		Package:      "grpcd",
		TypesPackage: testTypesPath,
	}
	tests := []struct {
		name    string
		change  func(*convConfig)
		infos   []typeInfo
		wantErr string // Part of the error message, empty for no error.
	}{
		{"valid", func(*convConfig) {}, infos, ""},
		{"bad package", func(c *convConfig) { c.Package = "a-b" }, infos,
			"invalid Go package name"},
		{"empty proto package", func(c *convConfig) { c.GoPackage = "" },
			infos, "empty Go package path"},
		{"empty types package", func(c *convConfig) { c.TypesPackage = "" },
			infos, "empty Go package path"},
		{"types package named pb",
			func(c *convConfig) { c.TypesPackage = "x/pb" }, infos,
			`cannot use package name "pb"`},
		{"types package named net",
			func(c *convConfig) { c.TypesPackage = "x/net" }, infos,
			`cannot use package name "net"`},
		{"types package named in",
			func(c *convConfig) { c.TypesPackage = "x/in" }, infos,
			`cannot use package name "in"`},
		{"types package with dash",
			func(c *convConfig) { c.TypesPackage = "x/my-types" }, infos,
			`cannot use package name "my-types"`},
		{"no types", func(*convConfig) {}, nil, "no @grpc types"},
	}
	for _, test := range tests {
		config := valid
		test.change(&config)
		_, err := generateConverters(config, test.infos, named)
		if test.wantErr == "" && err != nil {
			t.Errorf("%s: unexpected error %v", test.name, err)
		}
		if test.wantErr != "" &&
			(err == nil || !strings.Contains(err.Error(), test.wantErr)) {
			t.Errorf("%s: got error %v, want one containing %q", test.name,
				err, test.wantErr)
		}
	}
}

func TestGenerateConvertersTypeErrors(t *testing.T) {
	named := map[string]string{
		"Alias": "Inner",
		"Loop":  "Loop",
		"Pair":  "[2]byte",
	}
	tests := []struct {
		goType  string
		wantErr string
	}{
		{"*string", "cannot convert type *string"},
		{"map[string]*string", "cannot convert type *string"},
		{"Alias", "cannot convert type Alias of a message"},
		{"[4]byte", "cannot map array type [4]byte"},
		{"Pair", "cannot map array type [2]byte"},
		{"url.URL", "cannot map type url.URL"},
		{"[]url.URL", "cannot map type url.URL"},
		{"complex64", "cannot map type complex64"},
		{"Unknown", "cannot map type Unknown"},
		{"Loop", "cannot map recursive type Loop"},
		{"[][]string", "cannot map type [][]string"},
		{"map[float64]string", "cannot map type map[float64]string"},
		{"func()", "cannot map type func()"},
	}
	config := convConfig{
		GoPackage:    testGoPackage,
		Package:      "grpcd",
		TypesPackage: testTypesPath,
	}
	for _, test := range tests {
		infos := []typeInfo{
			{Name: "Inner", Fields: []fieldInfo{{"N", "string", 1}}},
			{Name: "M", Fields: []fieldInfo{{"F", test.goType, 1}}},
		}
		_, err := generateConverters(config, infos, named)
		if err == nil || !strings.Contains(err.Error(), test.wantErr) {
			t.Errorf("%s: got error %v, want one containing %q", test.goType,
				err, test.wantErr)
		}
	}
}

func TestGenerateConvertersFieldNames(t *testing.T) {
	config := convConfig{
		GoPackage:    testGoPackage,
		Package:      "grpcd",
		TypesPackage: testTypesPath,
	}
	tests := []struct {
		name    string
		wantErr bool
	}{
		{"Sha256", false},
		{"Sha256Sum", false},
		{"IPv4", false},
		{"Sha256sum", true},
		{"Foo_bar", true},
		{"String", true},
		{"Reset", true},
		{"Stringer", false},
		{"Marshal", true},
		{"hidden", true},
	}
	for _, test := range tests {
		infos := []typeInfo{
			{Name: "M", Fields: []fieldInfo{{test.name, "string", 1}}},
		}
		_, err := generateConverters(config, infos, nil)
		if (err != nil) != test.wantErr {
			t.Errorf("%s: got error %v, want error %v", test.name, err,
				test.wantErr)
		}
	}
}

func TestGenerateConvertersNames(t *testing.T) {
	config := convConfig{
		GoPackage:    testGoPackage,
		Package:      "grpcd",
		TypesPackage: testTypesPath,
	}
	tests := []struct {
		name    string
		infos   []typeInfo
		wantErr string
	}{
		{"unexported message", []typeInfo{{Name: "inner"}}, "message inner"},
		{"renamed message", []typeInfo{{Name: "IPv4config"}},
			"message IPv4config"},
		{"getter clash", []typeInfo{{Name: "M", Fields: []fieldInfo{
			{"Name", "string", 1}, {"GetName", "string", 2}}}},
			"field GetName"},
		{"getter without field", []typeInfo{{Name: "M", Fields: []fieldInfo{
			{"GetName", "string", 1}}}}, ""},
	}
	for _, test := range tests {
		_, err := generateConverters(config, test.infos, nil)
		if test.wantErr == "" && err != nil {
			t.Errorf("%s: unexpected error %v", test.name, err)
		}
		if test.wantErr != "" &&
			(err == nil || !strings.Contains(err.Error(), test.wantErr)) {
			t.Errorf("%s: got error %v, want one containing %q", test.name,
				err, test.wantErr)
		}
	}
}

func TestWellKnownTablesAgree(t *testing.T) {
	for name := range wellKnownTypes {
		if _, ok := wellKnownConvs[name]; !ok {
			t.Errorf("%s has a proto type but no converter", name)
		}
	}
	for name := range wellKnownConvs {
		if _, ok := wellKnownTypes[name]; !ok {
			t.Errorf("%s has a converter but no proto type", name)
		}
	}
}
