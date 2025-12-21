package vcs

import (
	"os"
	"testing"
)

// TestGit does a basic test on this git library. I don't want to do some docker madness
// or anything, so this test is designed only to be run on my laptop.
func TestGit(t *testing.T) {
	h, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	if h != "ElephantInTheRoom.local" {
		t.Skipf("not on jdoak's machine, so not testing... yeah its a lame test....")
	}

	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	g, err := NewGit(wd)
	if err != nil {
		panic(err)
	}

	if !g.Using() {
		t.Fatalf("TestGit: says we are not using git in our git directory")
	}

	rootWant := "/Users/jdoak/trees/claw/"
	if g.Root() != rootWant {
		t.Fatalf("TestGit: Root(): got %s, want %s", g.Root(), rootWant)
	}
}
