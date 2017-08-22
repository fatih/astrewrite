package astrewrite

import (
	"fmt"
	"go/ast"
)

// WalkFunc describes a function to be called for each node during a Walk. The
// returned node can be used to rewrite the AST. Returning nil will remove the node.
// Walking stops if the returned bool is false.
type WalkFunc func(ast.Node) (ast.Node, bool)

// Walk traverses an AST in depth-first order: It starts by calling
// fn(node); if node is nil, the node will be removed. It returns the rewritten node. If fn returns
// true, Walk invokes fn recursively for each of the non-nil children of node,
// followed by a call of fn(nil). The returned node of fn can be used to
// rewrite the passed node to fn. Panics if the returned type is not the same
// type as the original one.
func Walk(node ast.Node, fn WalkFunc) ast.Node {
	rewritten, ok := fn(node)
	if !ok {
		return rewritten
	}

	// walk children
	// (the order of the cases matches the order
	// of the corresponding node types in ast.go)
	switch n := node.(type) {
	// Comments and fields
	case *ast.Comment:
		// nothing to do

	case *ast.CommentGroup:
		out := n.List[:0]
		for _, c := range n.List {
			if c, _ = Walk(c, fn).(*ast.Comment); c != nil {
				out = append(out, c)
			}
		}
		n.List = out

	case *ast.Field:
		if n.Doc != nil {
			n.Doc = Walk(n.Doc, fn).(*ast.CommentGroup)
		}
		n.Names = walkIdentList(n.Names, fn)
		if n.Type, _ = Walk(n.Type, fn).(ast.Expr); n.Type == nil {
			return nil
		}
		if n.Tag != nil {
			n.Tag = Walk(n.Tag, fn).(*ast.BasicLit)
		}
		if n.Comment != nil {
			n.Comment = Walk(n.Comment, fn).(*ast.CommentGroup)
		}

	case *ast.FieldList:
		out := n.List[:0]
		for _, f := range n.List {
			if f, _ = Walk(f, fn).(*ast.Field); f != nil {
				out = append(out, f)
			}
		}
		n.List = out

	// Expressions
	case *ast.BadExpr, *ast.Ident, *ast.BasicLit:
		// nothing to do

	case *ast.Ellipsis:
		if n.Elt != nil {
			if n.Elt, _ = Walk(n.Elt, fn).(ast.Expr); n.Elt == nil {
				return nil
			}
		}

	case *ast.FuncLit:
		if n.Type, _ = Walk(n.Type, fn).(*ast.FuncType); n.Type == nil {
			return nil
		}
		n.Body = Walk(n.Body, fn).(*ast.BlockStmt)

	case *ast.CompositeLit:
		if n.Type != nil {
			n.Type, _ = Walk(n.Type, fn).(ast.Expr)
		}
		n.Elts = walkExprList(n.Elts, fn)

	case *ast.ParenExpr:
		n.X = Walk(n.X, fn).(ast.Expr)

	case *ast.SelectorExpr:
		n.X = Walk(n.X, fn).(ast.Expr)
		n.Sel = Walk(n.Sel, fn).(*ast.Ident)

	case *ast.IndexExpr:
		n.X = Walk(n.X, fn).(ast.Expr)
		n.Index = Walk(n.Index, fn).(ast.Expr)

	case *ast.SliceExpr:
		n.X = Walk(n.X, fn).(ast.Expr)
		if n.Low != nil {
			n.Low = Walk(n.Low, fn).(ast.Expr)
		}
		if n.High != nil {
			n.High = Walk(n.High, fn).(ast.Expr)
		}
		if n.Max != nil {
			n.Max = Walk(n.Max, fn).(ast.Expr)
		}

	case *ast.TypeAssertExpr:
		n.X = Walk(n.X, fn).(ast.Expr)
		if n.Type != nil {
			n.Type = Walk(n.Type, fn).(ast.Expr)
		}

	case *ast.CallExpr:
		if n.Fun, _ = Walk(n.Fun, fn).(ast.Expr); n.Fun == nil {
			return nil
		}
		n.Args = walkExprList(n.Args, fn)

	case *ast.StarExpr:
		n.X = Walk(n.X, fn).(ast.Expr)

	case *ast.UnaryExpr:
		n.X = Walk(n.X, fn).(ast.Expr)

	case *ast.BinaryExpr:
		n.X = Walk(n.X, fn).(ast.Expr)
		n.Y = Walk(n.Y, fn).(ast.Expr)

	case *ast.KeyValueExpr:
		n.Key = Walk(n.Key, fn).(ast.Expr)
		n.Value = Walk(n.Value, fn).(ast.Expr)

	// Types
	case *ast.ArrayType:
		if n.Len != nil {
			if n.Len, _ = Walk(n.Len, fn).(ast.Expr); n.Len == nil {
				return nil
			}
		}
		if n.Elt, _ = Walk(n.Elt, fn).(ast.Expr); n.Elt == nil {
			return nil
		}

	case *ast.StructType:
		if n.Fields, _ = Walk(n.Fields, fn).(*ast.FieldList); n.Fields == nil {
			return nil
		}

	case *ast.FuncType:
		// allow changing the params and/or results or completely removing them
		if n.Params != nil {
			n.Params, _ = Walk(n.Params, fn).(*ast.FieldList)
		}
		if n.Results != nil {
			n.Results, _ = Walk(n.Results, fn).(*ast.FieldList)
		}

	case *ast.InterfaceType:
		if n.Methods, _ = Walk(n.Methods, fn).(*ast.FieldList); n.Methods == nil {
			return nil
		}

	case *ast.MapType:
		if n.Key, _ = Walk(n.Key, fn).(ast.Expr); n.Key == nil {
			return nil
		}
		if n.Value, _ = Walk(n.Value, fn).(ast.Expr); n.Value == nil {
			return nil
		}

	case *ast.ChanType:
		if n.Value, _ = Walk(n.Value, fn).(ast.Expr); n.Value == nil {
			return nil
		}

	// Statements
	case *ast.BadStmt:
		// nothing to do

	case *ast.DeclStmt:
		if n.Decl, _ = Walk(n.Decl, fn).(ast.Decl); n.Decl == nil {
			return nil
		}

	case *ast.EmptyStmt:
		// nothing to do

	case *ast.LabeledStmt:
		n.Label = Walk(n.Label, fn).(*ast.Ident)
		n.Stmt = Walk(n.Stmt, fn).(ast.Stmt)

	case *ast.ExprStmt:
		if n.X, _ = Walk(n.X, fn).(ast.Expr); n.X == nil {
			return nil
		}

	case *ast.SendStmt:
		n.Chan = Walk(n.Chan, fn).(ast.Expr)
		n.Value = Walk(n.Value, fn).(ast.Expr)

	case *ast.IncDecStmt:
		n.X = Walk(n.X, fn).(ast.Expr)

	case *ast.AssignStmt:
		n.Lhs = walkExprList(n.Lhs, fn)
		n.Rhs = walkExprList(n.Rhs, fn)

	case *ast.GoStmt:
		n.Call = Walk(n.Call, fn).(*ast.CallExpr)

	case *ast.DeferStmt:
		n.Call = Walk(n.Call, fn).(*ast.CallExpr)

	case *ast.ReturnStmt:
		n.Results = walkExprList(n.Results, fn)

	case *ast.BranchStmt:
		if n.Label != nil {
			n.Label = Walk(n.Label, fn).(*ast.Ident)
		}

	case *ast.BlockStmt:
		n.List = walkStmtList(n.List, fn)

	case *ast.IfStmt:
		if n.Init != nil {
			n.Init = Walk(n.Init, fn).(ast.Stmt)
		}
		n.Cond = Walk(n.Cond, fn).(ast.Expr)
		n.Body = Walk(n.Body, fn).(*ast.BlockStmt)
		if n.Else != nil {
			n.Else = Walk(n.Else, fn).(ast.Stmt)
		}

	case *ast.CaseClause:
		n.List = walkExprList(n.List, fn)
		n.Body = walkStmtList(n.Body, fn)

	case *ast.SwitchStmt:
		if n.Init != nil {
			n.Init = Walk(n.Init, fn).(ast.Stmt)
		}
		if n.Tag != nil {
			n.Tag = Walk(n.Tag, fn).(ast.Expr)
		}
		n.Body = Walk(n.Body, fn).(*ast.BlockStmt)

	case *ast.TypeSwitchStmt:
		if n.Init != nil {
			n.Init = Walk(n.Init, fn).(ast.Stmt)
		}
		n.Assign = Walk(n.Assign, fn).(ast.Stmt)
		n.Body = Walk(n.Body, fn).(*ast.BlockStmt)

	case *ast.CommClause:
		if n.Comm != nil {
			n.Comm, _ = Walk(n.Comm, fn).(ast.Stmt)
		}
		n.Body = walkStmtList(n.Body, fn)

	case *ast.SelectStmt:
		n.Body = Walk(n.Body, fn).(*ast.BlockStmt)

	case *ast.ForStmt:
		if n.Init != nil {
			n.Init = Walk(n.Init, fn).(ast.Stmt)
		}
		if n.Cond != nil {
			n.Cond = Walk(n.Cond, fn).(ast.Expr)
		}
		if n.Post != nil {
			n.Post = Walk(n.Post, fn).(ast.Stmt)
		}
		n.Body = Walk(n.Body, fn).(*ast.BlockStmt)

	case *ast.RangeStmt:
		if n.Key != nil {
			n.Key = Walk(n.Key, fn).(ast.Expr)
		}
		if n.Value != nil {
			n.Value = Walk(n.Value, fn).(ast.Expr)
		}
		n.X = Walk(n.X, fn).(ast.Expr)
		n.Body = Walk(n.Body, fn).(*ast.BlockStmt)

	// Declarations
	case *ast.ImportSpec:
		if n.Doc != nil {
			n.Doc = Walk(n.Doc, fn).(*ast.CommentGroup)
		}
		if n.Name != nil {
			n.Name = Walk(n.Name, fn).(*ast.Ident)
		}
		n.Path = Walk(n.Path, fn).(*ast.BasicLit)
		if n.Comment != nil {
			n.Comment = Walk(n.Comment, fn).(*ast.CommentGroup)
		}

	case *ast.ValueSpec:
		if n.Doc != nil {
			n.Doc = Walk(n.Doc, fn).(*ast.CommentGroup)
		}
		n.Names = walkIdentList(n.Names, fn)
		if n.Type != nil {
			n.Type = Walk(n.Type, fn).(ast.Expr)
		}
		n.Values = walkExprList(n.Values, fn)
		if n.Comment != nil {
			n.Comment = Walk(n.Comment, fn).(*ast.CommentGroup)
		}

	case *ast.TypeSpec:
		if n.Doc != nil {
			n.Doc = Walk(n.Doc, fn).(*ast.CommentGroup)
		}
		Walk(n.Name, fn)
		Walk(n.Type, fn)
		if n.Comment != nil {
			n.Comment = Walk(n.Comment, fn).(*ast.CommentGroup)
		}

	case *ast.BadDecl:
		// nothing to do

	case *ast.GenDecl:
		if n.Doc != nil {
			n.Doc = Walk(n.Doc, fn).(*ast.CommentGroup)
		}
		for i, s := range n.Specs {
			s, _ = Walk(s, fn).(ast.Spec)
			if s != nil {
				n.Specs[i] = s
				continue
			}
			n.Specs = append(n.Specs[:i], n.Specs[i+1:]...)
		}
		if len(n.Specs) == 0 {
			return nil
		}
	case *ast.FuncDecl:
		if n.Doc != nil {
			n.Doc = Walk(n.Doc, fn).(*ast.CommentGroup)
		}
		if n.Recv != nil {
			n.Recv = Walk(n.Recv, fn).(*ast.FieldList)
		}
		n.Name = Walk(n.Name, fn).(*ast.Ident)
		n.Type = Walk(n.Type, fn).(*ast.FuncType)
		if n.Body != nil {
			n.Body = Walk(n.Body, fn).(*ast.BlockStmt)
		}

	// Files and packages
	case *ast.File:
		if n.Doc != nil {
			n.Doc = Walk(n.Doc, fn).(*ast.CommentGroup)
		}
		n.Name = Walk(n.Name, fn).(*ast.Ident)
		n.Decls = walkDeclList(n.Decls, fn)
		// don't walk n.Comments - they have been
		// visited already through the individual
		// nodes

	case *ast.Package:
		for i, f := range n.Files {
			n.Files[i] = Walk(f, fn).(*ast.File)
		}

	default:
		panic(fmt.Sprintf("ast.Walk: unexpected node type %T", n))
	}

	fn(nil)
	return rewritten
}

func walkIdentList(list []*ast.Ident, fn WalkFunc) []*ast.Ident {
	out := list[:0]
	for _, x := range list {
		if x = Walk(x, fn).(*ast.Ident); x != nil {
			out = append(out, x)
		}
	}
	return out
}

func walkExprList(list []ast.Expr, fn WalkFunc) []ast.Expr {
	out := list[:0]
	for _, x := range list {
		if x, _ = Walk(x, fn).(ast.Expr); x != nil {
			out = append(out, x)
		}
	}
	return out
}

func walkStmtList(list []ast.Stmt, fn WalkFunc) []ast.Stmt {
	out := list[:0]
	for _, x := range list {
		if x, _ = Walk(x, fn).(ast.Stmt); x != nil {
			out = append(out, x)
		}
	}
	return out
}

func walkDeclList(list []ast.Decl, fn WalkFunc) []ast.Decl {
	out := list[:0]
	for _, x := range list {
		if x, _ = Walk(x, fn).(ast.Decl); x != nil {
			out = append(out, x)
		}
	}
	return out
}
