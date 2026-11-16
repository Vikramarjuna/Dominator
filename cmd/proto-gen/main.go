package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/Cloud-Foundations/Dominator/lib/flags/loadflags"
	"github.com/Cloud-Foundations/Dominator/lib/fsutil"
)

var (
	convOutFile = flag.String("convOut", "",
		"File to write the generated converters to")
	convPackage = flag.String("convPackage", "",
		"Name of the Go package of the converters (with -convOut)")
	goPackage = flag.String("goPackage", "",
		"Import path of the generated proto Go code (with -out or -convOut)")
	goTypesPackage = flag.String("goTypesPackage", "",
		"Import path of the package containing the request and reply types "+
			"(with -convOut)")
	outFile = flag.String("out", "",
		"File to write the generated .proto to. If neither -out nor "+
			"-convOut is given, print the scan")
	protoDir = flag.String("proto", "",
		"Directory of the Go package containing the request and reply types")
	protoPackage = flag.String("package", "",
		"Name of the proto package (with -out)")
	rpcdDir = flag.String("rpcd", "",
		"Directory of the Go package containing the annotated SRPC methods")
	serviceName = flag.String("service", "",
		"Name of the gRPC service (with -out)")
)

// outFlags holds the flags which control what is written.
type outFlags struct {
	convOut        string
	convPackage    string
	goPackage      string
	goTypesPackage string
	out            string
	protoPackage   string
	service        string
}

// printScan writes a description of the methods and the types they use.
func printScan(methods []methodInfo, infos []typeInfo) {
	for _, method := range methods {
		fmt.Printf("method %s: %s -> %s", method.Name, method.RequestType,
			method.ResponseType)
		if method.NoRest {
			fmt.Print(" (no-rest)")
		}
		fmt.Println()
	}
	for _, info := range infos {
		fmt.Printf("message %s\n", info.Name)
		for _, field := range info.Fields {
			fmt.Printf("  %d %s %s\n", field.Number, field.Name,
				field.GoType)
		}
	}
}

// checkOutFlags returns an error if the generation flags are inconsistent.
// -goPackage, -package and -service are required with -out.
// -goPackage, -convPackage and -goTypesPackage are required with -convOut.
// Each is refused without the output that it belongs to.
func checkOutFlags(flags outFlags) error {
	if flags.out == "" && (flags.protoPackage != "" || flags.service != "") {
		return errors.New("-package and -service need -out")
	}
	if flags.convOut == "" &&
		(flags.convPackage != "" || flags.goTypesPackage != "") {
		return errors.New("-convPackage and -goTypesPackage need -convOut")
	}
	if flags.out == "" && flags.convOut == "" && flags.goPackage != "" {
		return errors.New("-goPackage needs -out or -convOut")
	}
	if flags.out != "" && (flags.goPackage == "" || flags.protoPackage == "" ||
		flags.service == "") {
		return errors.New(
			"-goPackage, -package and -service are required with -out")
	}
	if flags.convOut != "" && (flags.goPackage == "" ||
		flags.convPackage == "" || flags.goTypesPackage == "") {
		return errors.New("-goPackage, -convPackage and -goTypesPackage " +
			"are required with -convOut")
	}
	return nil
}

// doMain scans the annotated methods and their types. It writes the .proto
// file if -out is given and the converters if -convOut is given, and prints
// the scan if neither is. It returns the process exit status.
func doMain() int {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: proto-gen -rpcd dir -proto dir")
		fmt.Fprintln(os.Stderr,
			"       [-out file -package name -goPackage path -service name]")
		flag.PrintDefaults()
	}
	err := loadflags.LoadForCli("proto-gen")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	flag.Parse()
	if *protoDir == "" || *rpcdDir == "" {
		flag.Usage()
		return 2
	}
	err = checkOutFlags(outFlags{
		convOut:        *convOutFile,
		convPackage:    *convPackage,
		goPackage:      *goPackage,
		goTypesPackage: *goTypesPackage,
		out:            *outFile,
		protoPackage:   *protoPackage,
		service:        *serviceName,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		flag.Usage()
		return 2
	}
	methods, err := findMethods(*rpcdDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	infos, err := scanTypes(*protoDir, rootTypeNames(methods))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if *outFile == "" && *convOutFile == "" {
		printScan(methods, infos)
		return 0
	}
	named, err := scanNamedTypes(*protoDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	var protoText, convText string
	if *outFile != "" {
		config := protoConfig{
			GoPackage: *goPackage,
			Package:   *protoPackage,
			Service:   *serviceName,
		}
		protoText, err = generateProto(config, methods, infos, named)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}
	if *convOutFile != "" {
		config := convConfig{
			GoPackage:    *goPackage,
			Package:      *convPackage,
			TypesPackage: *goTypesPackage,
		}
		convText, err = generateConverters(config, infos, named)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}
	if *outFile != "" {
		err = os.WriteFile(*outFile, []byte(protoText), fsutil.PublicFilePerms)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}
	if *convOutFile != "" {
		err = os.WriteFile(*convOutFile, []byte(convText),
			fsutil.PublicFilePerms)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}
	return 0
}

// main runs doMain and exits with its status.
func main() {
	os.Exit(doMain())
}
