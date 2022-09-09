// Copyright 2019 The UNICORE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file
//
// Author: Gaulthier Gain <gaulthier.gain@uliege.be>

package common

const branch = "RELEASE-0.7.0"

// GitCloneRepository clones a git repository at the the given url.
//
// It returns two pointers of string which are respectively stdout and stderr
// and an error if any, otherwise it returns nil.
func GitCloneRepository(url, dir string, v bool) (*string, *string, error) {
	return ExecuteRunCmd("git", dir, v, "clone", url)
}

// GitBranchStaging updates the current branch of a git repository to the
// 'staging' branch.
//
// It returns two pointers of string which are respectively stdout and stderr
// and an error if any, otherwise it returns nil.
func GitBranchStaging(dir string, v bool) (*string, *string, error) {
	strOut, strErr, err := ExecuteRunCmd("git", dir, v, "branch", "-r")
	if err != nil {
		return strOut, strErr, err
	}

	//todo review
	//if strings.Contains(*strOut, branch) || strings.Contains(*strErr, branch) {
	return ExecuteRunCmd("git", dir, v, "checkout", branch)
	//}

	//return strOut, strErr, err
}

// GitPull pulls the current git repository.
//
// It returns two pointers of string which are respectively stdout and stderr
// and an error if any, otherwise it returns nil.
func GitPull(dir string, v bool) (*string, *string, error) {
	return ExecuteRunCmd("git", dir, v, "pull")
}
