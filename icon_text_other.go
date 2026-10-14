//go:build !windows

package main

import (
	"image"
	"image/color"
)

func drawNativeIconNumber(*image.NRGBA, string, color.NRGBA, int) bool { return false }
