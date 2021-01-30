// Copyright 2019 The UNICORE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file
//
// Author: Gaulthier Gain <gaulthier.gain@uliege.be>

package elf64core

import (
	"os"
	"strings"
)

var mapList = map[string][]string{"libkvmplat.o": {"haltme"}}

func isInSlice(libName, symbol string) bool {

	if strings.Contains(libName, string(os.PathSeparator)) {

		libNameSplit := strings.Split(libName, string(os.PathSeparator))

		if list, ok := mapList[libNameSplit[len(libNameSplit)-1]]; ok {
			for _, b := range list {
				if b == symbol {
					return true
				}
			}
		}
	}

	return false
}
