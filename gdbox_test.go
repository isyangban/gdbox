package main

import (
	"testing"
)

func Test_bfsFolders(t *testing.T) {
	//want := []string{"contacts.txt", "gdbox.go", "json.go", "maps.go", "max.go", "numbers.go", "slice.go", "strings.go"}
	result := bfsFolders("/data/service/share/", 1000)
	t.Log("Result of traversing ../:", result)
}

func TestDownload(t *testing.T) {
	files := []string{"contacts.txt"}
	folders := []string{"temp", "영재원"}

}
