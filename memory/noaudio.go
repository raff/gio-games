//go:build ios || android || js

package main

const (
	AudioFlip  = 0
	AudioReset = 1
)

func audioInit() {}

func audioPlay(aid int) {}
