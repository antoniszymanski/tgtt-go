// SPDX-FileCopyrightText: 2025 Antoni Szymański
// SPDX-License-Identifier: MPL-2.0

package tgtt

import "unsafe"

func bytesToString(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}
