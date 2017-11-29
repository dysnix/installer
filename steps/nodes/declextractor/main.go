package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

func main() {
	flag.Parse()

	declRegexp := regexp.MustCompile(*Flags.DeclName)
	tmpl := template.Must(template.ParseFiles(*Flags.DstTemplate))

	dirs2look := make([]string, 0, 2)
	if *Flags.Vendor {
		dirName := getVendorDir()
		if dirName != "" {
			dirs2look = append(dirs2look, dirName)
		}
	}

	for _, dirName := range strings.Split(os.Getenv("GOPATH"), string(os.PathListSeparator)) {
		dirs2look = append(dirs2look, path.Join(dirName, "src"))
	}

	srcDir := findSrcDir(dirs2look, *Flags.SrcPackage)
	if srcDir == "" {
		panic(fmt.Errorf("No directory found for package %q", *Flags.SrcPackage))
	}

	_, err := fmt.Fprintf(os.Stderr, "Parsing directory %q\n", srcDir)
	panicOnErr(err)

	astNodes, err := parser.ParseDir(
		token.NewFileSet(),
		srcDir,
		nil,
		0,
	)
	panicOnErr(err)

	result := make([]string, 0, 100)

	for _, astNode := range astNodes {
		ast.Inspect(
			astNode,
			func(n ast.Node) bool {
				switch decl := n.(type) {
				case *ast.ValueSpec:
					if declRegexp.MatchString(decl.Names[0].Name) {
						result = append(result, decl.Names[0].Name)
						return false
					}
				}
				return true
			},
		)
	}

	var params = struct {
		SrcPackage string
		DstPackage string
		VarName    string
		VarType    string
		DeclName   string
		Values     []string
	}{
		*Flags.SrcPackage,
		*Flags.DstPackage,
		*Flags.VarName,
		*Flags.VarType,
		*Flags.DeclName,
		result,
	}

	file, err := os.Create(*Flags.DstFile)
	panicOnErr(err)
	defer func() {
		panicOnErr(file.Close())
	}()

	panicOnErr(tmpl.Execute(file, params))
}

func findSrcDir(dirs2look []string, pkgName string) string {
	pkgName = filepath.FromSlash(pkgName)

	for _, dirName := range dirs2look {
		srcDir := path.Join(dirName, pkgName)
		stat, err := os.Stat(srcDir)
		if stat != nil && stat.IsDir() {
			return srcDir
		}
		if os.IsNotExist(err) {
			continue
		}
		panicOnErr(err)
	}

	return ""
}

func getVendorDir() string {
	dir, err := filepath.Abs(".")
	panicOnErr(err)

	for {
		vendorDir := path.Join(dir, "vendor")

		dirStat, err := os.Stat(vendorDir)
		if err != nil && !os.IsNotExist(err) {
			panic(err)
		}

		if dirStat != nil && dirStat.IsDir() {
			return vendorDir
		}

		newDir, err := filepath.Abs(path.Join(dir, ".."))
		panicOnErr(err)
		if newDir == dir {
			return ""
		}
		dir = newDir
	}
}

func panicOnErr(err error) {
	if err != nil {
		panic(err)
	}
}

// Flags is a struct for command line flags ready to be utilized by flag.Parse()
var Flags = struct {
	Vendor      *bool
	SrcPackage  *string
	DstFile     *string
	DstTemplate *string
	DstPackage  *string
	DeclName    *string
	VarName     *string
	VarType     *string
}{
	flag.Bool("Vendor", true, "Use vendored packages"),
	flag.String("SrcPackage", "", "Source package name as in import ({{vendor}}|{{$GOPATH/src}}/{{SrcPackage}} directory will be parsed)"),
	flag.String("DstFile", "", "Destination file name"),
	flag.String("DstTemplate", "", "Destination template file name"),
	flag.String("DstPackage", "main", "Destination package name"),
	flag.String("DeclName", `.+`, "Declaration to extract name regexp"),
	flag.String("VarName", "dst", "Destination variable name"),
	flag.String("VarType", "[]interface{}", "Destination variable type"),
}
