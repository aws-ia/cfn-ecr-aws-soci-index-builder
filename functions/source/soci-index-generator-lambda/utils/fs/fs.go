// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package fs contains utilities for checking free space in a directory
package fs

import "golang.org/x/sys/unix"

// Calculate free splace in bytes of a directory
func CalculateFreeSpace(path string) uint64 {
	var stat unix.Statfs_t
	err := unix.Statfs(path, &stat)
	if err != nil {
		panic(err)
	}
	// Available blocks * size per block = available space in bytes
	return stat.Bavail * uint64(stat.Bsize)
}
