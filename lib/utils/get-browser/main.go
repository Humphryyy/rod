// Package main ...
package main

import (
	"fmt"

	"github.com/Humphryyy/rod/lib/launcher"
	"github.com/Humphryyy/rod/lib/utils"
)

func main() {
	p, err := launcher.NewBrowser().Get()
	utils.E(err)

	fmt.Println(p)
}
