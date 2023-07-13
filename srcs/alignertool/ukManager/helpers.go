// Copyright 2019 The UNICORE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file
//
// Author: Gaulthier Gain <gaulthier.gain@uliege.be>

package ukManager

import (
	"strings"
)

func stringInSlice(name string, plats []string) bool {
	for _, plat := range plats {
		if strings.Contains(name, plat) {
			return true
		}
	}
	return false
}
