// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package misc

import (
	"strings"
	"testing"

	"golang.org/x/tools/gopls/internal/lsp/protocol"
	. "golang.org/x/tools/gopls/internal/lsp/regtest"
	"golang.org/x/tools/internal/testenv"
)

func TestPrepareRenameMainPackage(t *testing.T) {
	const files = `
-- go.mod --
module mod.com

go 1.18
-- main.go --
package main

import (
	"fmt"
)

func main() {
	fmt.Println(1)
}
`
	const wantErr = "can't rename package \"main\""
	Run(t, files, func(t *testing.T, env *Env) {
		env.OpenFile("main.go")
		pos := env.RegexpSearch("main.go", `main`)
		tdpp := protocol.TextDocumentPositionParams{
			TextDocument: env.Editor.TextDocumentIdentifier("main.go"),
			Position:     pos.ToProtocolPosition(),
		}
		params := &protocol.PrepareRenameParams{
			TextDocumentPositionParams: tdpp,
		}
		_, err := env.Editor.Server.PrepareRename(env.Ctx, params)
		if err == nil {
			t.Errorf("missing can't rename package main error from PrepareRename")
		}

		if err.Error() != wantErr {
			t.Errorf("got %v, want %v", err.Error(), wantErr)
		}
	})
}

func TestPrepareRenameWithNoPackageDeclaration(t *testing.T) {
	testenv.NeedsGo1Point(t, 15)
	const files = `
go 1.14
-- lib/a.go --
import "fmt"

const A = 1

func bar() {
	fmt.Println("Bar")
}

-- main.go --
package main

import "fmt"

func main() {
	fmt.Println("Hello")
}
`
	const wantErr = "no object found"
	Run(t, files, func(t *testing.T, env *Env) {
		env.OpenFile("lib/a.go")
		pos := env.RegexpSearch("lib/a.go", "fmt")

		err := env.Editor.Rename(env.Ctx, "lib/a.go", pos, "fmt1")
		if err == nil {
			t.Errorf("missing no object found from Rename")
		}

		if err.Error() != wantErr {
			t.Errorf("got %v, want %v", err.Error(), wantErr)
		}
	})
}

func TestPrepareRenameFailWithUnknownModule(t *testing.T) {
	testenv.NeedsGo1Point(t, 17)
	const files = `
go 1.14
-- lib/a.go --
package lib

const A = 1

-- main.go --
package main

import (
	"mod.com/lib"
)

func main() {
	println("Hello")
}
`
	const wantErr = "can't rename package: missing module information for package"
	Run(t, files, func(t *testing.T, env *Env) {
		pos := env.RegexpSearch("lib/a.go", "lib")
		tdpp := protocol.TextDocumentPositionParams{
			TextDocument: env.Editor.TextDocumentIdentifier("lib/a.go"),
			Position:     pos.ToProtocolPosition(),
		}
		params := &protocol.PrepareRenameParams{
			TextDocumentPositionParams: tdpp,
		}
		_, err := env.Editor.Server.PrepareRename(env.Ctx, params)
		if err == nil || !strings.Contains(err.Error(), wantErr) {
			t.Errorf("missing cannot rename packages with unknown module from PrepareRename")
		}
	})
}

func TestRenamePackageWithConflicts(t *testing.T) {
	testenv.NeedsGo1Point(t, 17)
	const files = `
-- go.mod --
module mod.com

go 1.18
-- lib/a.go --
package lib

const A = 1

-- lib/nested/a.go --
package nested

const B = 1

-- lib/x/a.go --
package nested1

const C = 1

-- main.go --
package main

import (
	"mod.com/lib"
	"mod.com/lib/nested"
	nested1 "mod.com/lib/x"
)

func main() {
	println("Hello")
}
`
	Run(t, files, func(t *testing.T, env *Env) {
		env.OpenFile("lib/a.go")
		pos := env.RegexpSearch("lib/a.go", "lib")
		env.Rename("lib/a.go", pos, "nested")

		// Check if the new package name exists.
		env.RegexpSearch("nested/a.go", "package nested")
		env.RegexpSearch("main.go", `nested2 "mod.com/nested"`)
		env.RegexpSearch("main.go", "mod.com/nested/nested")
		env.RegexpSearch("main.go", `nested1 "mod.com/nested/x"`)
	})
}

func TestRenamePackageWithAlias(t *testing.T) {
	testenv.NeedsGo1Point(t, 17)
	const files = `
-- go.mod --
module mod.com

go 1.18
-- lib/a.go --
package lib

const A = 1

-- lib/nested/a.go --
package nested

const B = 1

-- main.go --
package main

import (
	"mod.com/lib"
	lib1 "mod.com/lib/nested"
)

func main() {
	println("Hello")
}
`
	Run(t, files, func(t *testing.T, env *Env) {
		env.OpenFile("lib/a.go")
		pos := env.RegexpSearch("lib/a.go", "lib")
		env.Rename("lib/a.go", pos, "nested")

		// Check if the new package name exists.
		env.RegexpSearch("nested/a.go", "package nested")
		env.RegexpSearch("main.go", "mod.com/nested")
		env.RegexpSearch("main.go", `lib1 "mod.com/nested/nested"`)
	})
}

func TestRenamePackageWithDifferentDirectoryPath(t *testing.T) {
	testenv.NeedsGo1Point(t, 17)
	const files = `
-- go.mod --
module mod.com

go 1.18
-- lib/a.go --
package lib

const A = 1

-- lib/nested/a.go --
package foo

const B = 1

-- main.go --
package main

import (
	"mod.com/lib"
	foo "mod.com/lib/nested"
)

func main() {
	println("Hello")
}
`
	Run(t, files, func(t *testing.T, env *Env) {
		env.OpenFile("lib/a.go")
		pos := env.RegexpSearch("lib/a.go", "lib")
		env.Rename("lib/a.go", pos, "nested")

		// Check if the new package name exists.
		env.RegexpSearch("nested/a.go", "package nested")
		env.RegexpSearch("main.go", "mod.com/nested")
		env.RegexpSearch("main.go", `foo "mod.com/nested/nested"`)
	})
}

func TestRenamePackage(t *testing.T) {
	testenv.NeedsGo1Point(t, 17)
	const files = `
-- go.mod --
module mod.com

go 1.18
-- lib/a.go --
package lib

const A = 1

-- lib/b.go --
package lib

const B = 1

-- lib/nested/a.go --
package nested

const C = 1

-- main.go --
package main

import (
	"mod.com/lib"
	"mod.com/lib/nested"
)

func main() {
	println("Hello")
}
`
	Run(t, files, func(t *testing.T, env *Env) {
		env.OpenFile("lib/a.go")
		pos := env.RegexpSearch("lib/a.go", "lib")
		env.Rename("lib/a.go", pos, "lib1")

		// Check if the new package name exists.
		env.RegexpSearch("lib1/a.go", "package lib1")
		env.RegexpSearch("lib1/b.go", "package lib1")
		env.RegexpSearch("main.go", "mod.com/lib1")
		env.RegexpSearch("main.go", "mod.com/lib1/nested")
	})
}

// Test for golang/go#47564.
func TestRenameInTestVariant(t *testing.T) {
	const files = `
-- go.mod --
module mod.com

go 1.12
-- stringutil/stringutil.go --
package stringutil

func Identity(s string) string {
	return s
}
-- stringutil/stringutil_test.go --
package stringutil

func TestIdentity(t *testing.T) {
	if got := Identity("foo"); got != "foo" {
		t.Errorf("bad")
	}
}
-- main.go --
package main

import (
	"fmt"

	"mod.com/stringutil"
)

func main() {
	fmt.Println(stringutil.Identity("hello world"))
}
`

	Run(t, files, func(t *testing.T, env *Env) {
		env.OpenFile("main.go")
		pos := env.RegexpSearch("main.go", `stringutil\.(Identity)`)
		env.Rename("main.go", pos, "Identityx")
		text := env.Editor.BufferText("stringutil/stringutil_test.go")
		if !strings.Contains(text, "Identityx") {
			t.Errorf("stringutil/stringutil_test.go: missing expected token `Identityx` after rename:\n%s", text)
		}
	})
}

// This is a test that rename operation initiated by the editor function as expected.
func TestRenameFileFromEditor(t *testing.T) {
	const files = `
-- go.mod --
module mod.com

go 1.16
-- a/a.go --
package a

const X = 1
-- a/x.go --
package a

const X = 2
-- b/b.go --
package b
`

	Run(t, files, func(t *testing.T, env *Env) {
		// Rename files and verify that diagnostics are affected accordingly.

		// Initially, we should have diagnostics on both X's, for their duplicate declaration.
		env.Await(
			OnceMet(
				InitialWorkspaceLoad,
				env.DiagnosticAtRegexp("a/a.go", "X"),
				env.DiagnosticAtRegexp("a/x.go", "X"),
			),
		)

		// Moving x.go should make the diagnostic go away.
		env.RenameFile("a/x.go", "b/x.go")
		env.Await(
			OnceMet(
				env.DoneWithChangeWatchedFiles(),
				EmptyDiagnostics("a/a.go"),                  // no more duplicate declarations
				env.DiagnosticAtRegexp("b/b.go", "package"), // as package names mismatch
			),
		)

		// Renaming should also work on open buffers.
		env.OpenFile("b/x.go")

		// Moving x.go back to a/ should cause the diagnostics to reappear.
		env.RenameFile("b/x.go", "a/x.go")
		// TODO(rfindley): enable using a OnceMet precondition here. We can't
		// currently do this because DidClose, DidOpen and DidChangeWatchedFiles
		// are sent, and it is not easy to use all as a precondition.
		env.Await(
			env.DiagnosticAtRegexp("a/a.go", "X"),
			env.DiagnosticAtRegexp("a/x.go", "X"),
		)

		// Renaming the entire directory should move both the open and closed file.
		env.RenameFile("a", "x")
		env.Await(
			env.DiagnosticAtRegexp("x/a.go", "X"),
			env.DiagnosticAtRegexp("x/x.go", "X"),
		)

		// As a sanity check, verify that x/x.go is open.
		if text := env.Editor.BufferText("x/x.go"); text == "" {
			t.Fatal("got empty buffer for x/x.go")
		}
	})
}

func TestRenameWithTestPackage(t *testing.T) {
	testenv.NeedsGo1Point(t, 17)
	const files = `
-- go.mod --
module mod.com

go 1.18
-- lib/a.go --
package lib

const A = 1

-- lib/b.go --
package lib

const B = 1

-- lib/a_test.go --
package lib_test

import (
	"mod.com/lib"
	"fmt
)

const C = 1

-- lib/b_test.go --
package lib

import (
	"fmt
)

const D = 1

-- lib/nested/a.go --
package nested

const D = 1

-- main.go --
package main

import (
	"mod.com/lib"
	"mod.com/lib/nested"
)

func main() {
	println("Hello")
}
`
	Run(t, files, func(t *testing.T, env *Env) {
		env.OpenFile("lib/a.go")
		pos := env.RegexpSearch("lib/a.go", "lib")
		env.Rename("lib/a.go", pos, "lib1")

		// Check if the new package name exists.
		env.RegexpSearch("lib1/a.go", "package lib1")
		env.RegexpSearch("lib1/b.go", "package lib1")
		env.RegexpSearch("main.go", "mod.com/lib1")
		env.RegexpSearch("main.go", "mod.com/lib1/nested")

		// Check if the test package is renamed
		env.RegexpSearch("lib1/a_test.go", "package lib1_test")
		env.RegexpSearch("lib1/b_test.go", "package lib1")
	})
}

func TestRenameWithNestedModule(t *testing.T) {
	testenv.NeedsGo1Point(t, 17)
	const files = `
-- go.mod --
module mod.com

go 1.18

require (
    mod.com/foo/bar v0.0.0
)

replace mod.com/foo/bar => ./foo/bar
-- foo/foo.go --
package foo

import "fmt"

func Bar() {
	fmt.Println("In foo before renamed to foox.")
}

-- foo/bar/go.mod --
module mod.com/foo/bar

-- foo/bar/bar.go --
package bar

const Msg = "Hi"

-- main.go --
package main

import (
	"fmt"
	"mod.com/foo/bar"
	"mod.com/foo"
)

func main() {
	foo.Bar()
	fmt.Println(bar.Msg)
}
`
	Run(t, files, func(t *testing.T, env *Env) {
		env.OpenFile("foo/foo.go")
		pos := env.RegexpSearch("foo/foo.go", "foo")
		env.Rename("foo/foo.go", pos, "foox")

		env.RegexpSearch("foox/foo.go", "package foox")
		env.OpenFile("foox/bar/bar.go")
		env.OpenFile("foox/bar/go.mod")

		env.RegexpSearch("main.go", "mod.com/foo/bar")
		env.RegexpSearch("main.go", "mod.com/foox")
		env.RegexpSearch("main.go", "foox.Bar()")
	})
}

func TestRenamePackageWithNonBlankSameImportPaths(t *testing.T) {
	testenv.NeedsGo1Point(t, 17)
	const files = `
-- go.mod --
module mod.com

go 1.18
-- lib/a.go --
package lib

const A = 1

-- lib/nested/a.go --
package nested

const B = 1

-- main.go --
package main

import (
	"mod.com/lib"
	lib1 "mod.com/lib"
	lib2 "mod.com/lib/nested"
)

func main() {
	println("Hello")
}
`
	Run(t, files, func(t *testing.T, env *Env) {
		env.OpenFile("lib/a.go")
		pos := env.RegexpSearch("lib/a.go", "lib")
		env.Rename("lib/a.go", pos, "nested")

		// Check if the new package name exists.
		env.RegexpSearch("nested/a.go", "package nested")
		env.RegexpSearch("main.go", "mod.com/nested")
		env.RegexpSearch("main.go", `lib1 "mod.com/nested"`)
		env.RegexpSearch("main.go", `lib2 "mod.com/nested/nested"`)
	})
}

func TestRenamePackageWithBlankSameImportPaths(t *testing.T) {
	testenv.NeedsGo1Point(t, 17)
	const files = `
-- go.mod --
module mod.com

go 1.18
-- lib/a.go --
package lib

const A = 1

-- lib/nested/a.go --
package nested

const B = 1

-- main.go --
package main

import (
	"mod.com/lib"
	_ "mod.com/lib"
	lib1 "mod.com/lib/nested"
)

func main() {
	println("Hello")
}
`
	Run(t, files, func(t *testing.T, env *Env) {
		env.OpenFile("lib/a.go")
		pos := env.RegexpSearch("lib/a.go", "lib")
		env.Rename("lib/a.go", pos, "nested")

		// Check if the new package name exists.
		env.RegexpSearch("nested/a.go", "package nested")
		env.RegexpSearch("main.go", "mod.com/nested")
		env.RegexpSearch("main.go", `_ "mod.com/nested"`)
		env.RegexpSearch("main.go", `lib1 "mod.com/nested/nested"`)
	})
}
