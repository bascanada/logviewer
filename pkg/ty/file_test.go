package ty

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadFileText(t *testing.T) {
	var payload = "test123: value\n" +
		"rare=value\n" +
		"rare2= value\n" +
		"Origin: https://splunk.com"
	path := CreateTestFile(t.Name(), payload)
	defer DeleteTestFile(path)

	ms := MS{}

	_ = (&ms).LoadMS(path)

	assert.Equal(t, "value", ms["test123"], "value not equal")
	assert.Equal(t, "value", ms["rare"], "value not equal")
	assert.Equal(t, "value", ms["rare2"], "value not equal")
	assert.Equal(t, "https://splunk.com", ms["Origin"], "value not equal")

}

func TestLoadFileJson(t *testing.T) {
	var payload = `{"test123": "value", "rare": "value", "rare2": "value"}`
	path := CreateTestFile(t.Name(), payload)
	defer DeleteTestFile(path)

	ms := MS{}

	_ = (&ms).LoadMS(path)

	assert.Equal(t, "value", ms["test123"], "value not equal")
	assert.Equal(t, "value", ms["rare"], "value not equal")
	assert.Equal(t, "value", ms["rare2"], "value not equal")
}

func CreateTestFile(name string, content string) string {
	f, err := os.Create(fmt.Sprintf("/tmp/%s", name))
	if err != nil {
		panic(f)
	}

	defer func() { _ = f.Close() }()

	if _, err := f.Write([]byte(content)); err != nil {
		panic(err)
	}

	return f.Name()

}

func DeleteTestFile(path string) {
	if err := os.Remove(path); err != nil {
		panic(err)
	}
}
