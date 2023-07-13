// Copyright 2019 The UNICORE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file
//
// Author: Gaulthier Gain <gaulthier.gain@uliege.be>

package common

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

func StringInSlice(name string, plats []string) bool {
	for _, plat := range plats {
		if strings.Contains(name, plat) {
			return true
		}
	}
	return false
}

// Contains checks if a given slice contains a particular string.
//
// It returns true if the given contains the searched string.
func Contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// DownloadFile downloads a file from an URL and reads its content.
//
// It returns a pointer to a string that represents the file content and an
// error if any, otherwise it returns nil.
func DownloadFile(url string) (*string, error) {

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var bodyString string
	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		bodyString = string(bodyBytes)
	}
	return &bodyString, err
}

// GetProgramPath returns the absolute path of a given program.
//
// It returns a string that represents the absolute path of a program and an
// error if any, otherwise it returns nil.
func GetProgramPath(programName *string) (string, error) {
	var programPath string
	if ok, err := Exists(*programName); err != nil {
		return programPath, err
	} else if ok {
		// Program (binary) is installed
		if filepath.IsAbs(*programName) {
			programPath = *programName
			*programName = filepath.Base(programPath)
		} else if programPath, err = filepath.Abs(*programName); err != nil {
			return programPath, err
		}
	} else {
		// Run 'which' command to determine if program has a symbolic name
		out, err := ExecuteCommand("which", []string{*programName})
		if err != nil {
			return programPath, err
		} else {
			// Check if out is a valid path
			if _, err := os.Stat(out); err == nil {
				PrintWarning("Unknown Program -> will skip gathering" +
					" symbols, system calls and shared libs process")
			} else {
				programPath = strings.TrimSpace(out)
			}
		}
	}
	return programPath, nil
}

// ---------------------------------Record Data---------------------------------

// RecordDataTxt saves data into a text file named by filename.
//
// It returns an error if any, otherwise it returns nil.
func RecordDataTxt(filename string, headers []string, data interface{}) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	v := reflect.ValueOf(data)
	values := make([]interface{}, v.NumField())

	for i := 0; i < v.NumField(); i++ {
		values[i] = v.Field(i).Interface()
		if err := WriteMapToFile(file, headers[i], values[i]); err != nil {
			return err
		}
	}

	return nil
}

// WriteMapToFile saves map into a text file named by filename.
//
// It returns an error if any, otherwise it returns nil.
func WriteMapToFile(file *os.File, headerName string, in interface{}) error {
	header := "----------------------------------------------\n" +
		headerName + "\n----------------------------------------------\n"

	if _, err := file.WriteString(header); err != nil {
		return err
	}

	switch v := in.(type) {
	case map[string]string:
		for key, value := range v {

			var str string
			if len(value) > 0 {
				str = key + "@" + value
			} else {
				str = key
			}

			if _, err := file.WriteString(str + "\n"); err != nil {
				return err
			}
		}
	case map[string][]string:
		for key, values := range v {

			var str string
			if len(values) > 0 {
				str = key + "->" + strings.Join(values, ",")
			} else {
				str = key
			}

			if _, err := file.WriteString(str + "\n"); err != nil {
				return err
			}
		}
	}

	return nil
}

// -------------------------------------JSON------------------------------------

// RecordDataJson saves json into a json file named by filename.
//
// It returns an error if any, otherwise it returns nil.
func RecordDataJson(filename string, data interface{}) error {

	b, err := json.Marshal(data)
	if err != nil {
		return err
	}

	var prettyJSON bytes.Buffer
	if err = json.Indent(&prettyJSON, b, "", "\t"); err != nil {
		return err
	}
	if err = ioutil.WriteFile(filename+".json", prettyJSON.Bytes(), os.ModePerm); err != nil {
		return err
	}

	return nil
}

// ReadDataJson load json from a json file named by filename.
//
// It returns a Data structure initialized and an error if any, otherwise it
// returns nil.
func ReadDataJson(filename string, data *Data) (*Data, error) {

	jsonFile, err := os.Open(filename + ".json")
	if err != nil {
		return nil, err
	}
	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(byteValue, &data); err != nil {
		return nil, err
	}

	return data, nil
}
