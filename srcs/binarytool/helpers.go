// Copyright 2019 The UNICORE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file
//
// Author: Gaulthier Gain <gaulthier.gain@uliege.be>

package binarytool

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"
)

func ReadJsonFile(path string) (*Unikernels, error) {
	jsonFile, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer jsonFile.Close()
	byteValue, _ := ioutil.ReadAll(jsonFile)
	unikernels := new(Unikernels)
	if err := json.Unmarshal(byteValue, unikernels); err != nil {
		return nil, err
	}

	return unikernels, nil
}

func stringInSlice(name string, plats []string) bool {
	for _, plat := range plats {
		if strings.Contains(name, plat) {
			return true
		}
	}
	return false
}
