//go:build linux
// +build linux

package main

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2023 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	_ "embed"

	CLONE "github.com/essentialkaos/rbinstall/clone"
)

// ////////////////////////////////////////////////////////////////////////////////// //

//go:embed go.mod
var gomod []byte

// gitrev is short hash of the latest git commit
var gitrev string

// ////////////////////////////////////////////////////////////////////////////////// //

func main() {
	CLONE.Init(gitrev, gomod)
}
