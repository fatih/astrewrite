package astrewrite

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
)

func ExampleWalk() {
	src := `package main

type Foo struct{}`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "foo.go", src, parser.ParseComments)
	if err != nil {
		panic(err)
	}

	rewriteFunc := func(n ast.Node) (ast.Node, bool) {
		x, ok := n.(*ast.TypeSpec)
		if !ok {
			return n, true
		}

		// change struct type name to "Bar"
		x.Name.Name = "Bar"
		return x, true
	}

	rewritten := Walk(file, rewriteFunc)

	var buf bytes.Buffer
	printer.Fprint(&buf, fset, rewritten)
	fmt.Println(buf.String())
	// Output:
	// package main
	//
	// type Bar struct{}
}
