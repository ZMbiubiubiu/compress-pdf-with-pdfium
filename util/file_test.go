package util

import (
	"bytes"
	"image"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetermineImgType(t *testing.T) {
	var filePath = map[string]string{
		"./1.jpeg": "jpeg",
		"./1.png":  "png",
		"./1.jpg":  "jpeg",
	}

	for k, v := range filePath {
		data, err := os.ReadFile(k)
		if err != nil {
			t.Errorf("error: %v", err)
		}
		_, format, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			t.Errorf("error: %v", err)
		}
		assert.Equal(t, v, format)
	}
}
