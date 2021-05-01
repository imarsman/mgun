package common

import "testing"

func TestPaths(t *testing.T) {
	t.Log("Root dir", RootDir())
	t.Log("Project dir", ProjectDir())
}
