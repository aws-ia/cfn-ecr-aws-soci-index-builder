// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package fs

import "testing"

func TestGetFreeSpace(t *testing.T) {
	if CalculateFreeSpace("/tmp") <= 0 {
		t.Fatalf("Expected free space of /tmp to be greater than 0")
	}
}
