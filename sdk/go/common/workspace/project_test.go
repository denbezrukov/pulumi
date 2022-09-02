package workspace

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

func TestProjectRuntimeInfoRoundtripYAML(t *testing.T) {
	t.Parallel()

	doTest := func(marshal func(interface{}) ([]byte, error), unmarshal func([]byte, interface{}) error) {
		ri := NewProjectRuntimeInfo("nodejs", nil)
		byts, err := marshal(ri)
		assert.NoError(t, err)

		var riRountrip ProjectRuntimeInfo
		err = unmarshal(byts, &riRountrip)
		assert.NoError(t, err)
		assert.Equal(t, "nodejs", riRountrip.Name())
		assert.Nil(t, riRountrip.Options())

		ri = NewProjectRuntimeInfo("nodejs", map[string]interface{}{
			"typescript":   true,
			"stringOption": "hello",
		})
		byts, err = marshal(ri)
		assert.NoError(t, err)
		err = unmarshal(byts, &riRountrip)
		assert.NoError(t, err)
		assert.Equal(t, "nodejs", riRountrip.Name())
		assert.Equal(t, true, riRountrip.Options()["typescript"])
		assert.Equal(t, "hello", riRountrip.Options()["stringOption"])
	}

	doTest(yaml.Marshal, yaml.Unmarshal)
	doTest(json.Marshal, json.Unmarshal)
}

func TestProjectValidation(t *testing.T) {
	t.Parallel()
	var err error

	// Test lack of name
	proj := Project{}
	err = proj.Validate()
	assert.Error(t, err)
	assert.Equal(t, "project is missing a 'name' attribute", err.Error())
	// Test lack of runtime
	proj.Name = "a project"
	err = proj.Validate()
	assert.Error(t, err)
	assert.Equal(t, "project is missing a 'runtime' attribute", err.Error())

	// Test success
	proj.Runtime = NewProjectRuntimeInfo("test", nil)
	err = proj.Validate()
	assert.NoError(t, err)
}

func TestProjectUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var proj Project

	// Test wrong type
	data := "\"hello\""
	err := json.Unmarshal([]byte(data), &proj)
	assert.Equal(t, "expected a JSON object", err.Error())

	// Test lack of name
	data = "{}"
	err = json.Unmarshal([]byte(data), &proj)
	assert.Equal(t, "project is missing a 'name' attribute", err.Error())

	// Test bad name
	data = "{\"name\": \"\"}"
	err = json.Unmarshal([]byte(data), &proj)
	assert.Equal(t, "project is missing a non-empty string 'name' attribute", err.Error())

	// Test missing runtime
	data = "{\"name\": \"project\"}"
	err = json.Unmarshal([]byte(data), &proj)
	assert.Equal(t, "project is missing a 'runtime' attribute", err.Error())

	// Test other schema errors
	data = "{\"name\": \"project\", \"runtime\": 4}"
	err = json.Unmarshal([]byte(data), &proj)
	expected := `3 errors occurred:
	* #/runtime: oneOf failed
	* #/runtime: expected string, but got number
	* #/runtime: expected object, but got number

`
	assert.Equal(t, expected, err.Error())

	data = "{\"name\": \"project\", \"runtime\": \"test\", \"backend\": 4, \"main\": {}}"
	err = json.Unmarshal([]byte(data), &proj)
	expected = `2 errors occurred:
	* #/main: expected string, but got object
	* #/backend: expected string, but got number

`
	assert.Equal(t, expected, err.Error())

	// Test success
	data = "{\"name\": \"project\", \"runtime\": \"test\"}"
	err = json.Unmarshal([]byte(data), &proj)
	assert.NoError(t, err)
	assert.Equal(t, "project", proj.Name)
	assert.Equal(t, "test", proj.Runtime.Name())
}

func TestProjectUnmarshalYAML(t *testing.T) {
	t.Parallel()
	var proj Project

	// Test wrong type
	data := "\"hello\""
	err := yaml.Unmarshal([]byte(data), &proj)
	assert.Equal(t, "expected a YAML object", err.Error())

	// Test bad key
	data = "4: hello"
	err = yaml.Unmarshal([]byte(data), &proj)
	assert.Equal(t, "expected only string keys, got '%!s(int=4)'", err.Error())

	// Test nested bad key
	data = "hello:\n    6: bad"
	err = yaml.Unmarshal([]byte(data), &proj)
	assert.Equal(t, "expected only string keys, got '%!s(int=6)'", err.Error())

	// Test lack of name
	data = "{}"
	err = yaml.Unmarshal([]byte(data), &proj)
	assert.Equal(t, "project is missing a 'name' attribute", err.Error())

	// Test bad name
	data = "name:"
	err = yaml.Unmarshal([]byte(data), &proj)
	assert.Equal(t, "project is missing a non-empty string 'name' attribute", err.Error())

	// Test missing runtime
	data = "name: project"
	err = yaml.Unmarshal([]byte(data), &proj)
	assert.Equal(t, "project is missing a 'runtime' attribute", err.Error())

	// Test other schema errors
	data = "name: project\nruntime: 4"
	err = yaml.Unmarshal([]byte(data), &proj)
	expected := `3 errors occurred:
	* #/runtime: oneOf failed
	* #/runtime: expected string, but got number
	* #/runtime: expected object, but got number

`
	assert.Equal(t, expected, err.Error())

	data = "name: project\nruntime: test\nbackend: 4\nmain: {}"
	err = yaml.Unmarshal([]byte(data), &proj)
	expected = `2 errors occurred:
	* #/main: expected string, but got object
	* #/backend: expected string, but got number

`
	assert.Equal(t, expected, err.Error())

	// Test success
	data = "name: project\nruntime: test"
	err = json.Unmarshal([]byte(data), &proj)
	assert.NoError(t, err)
	assert.Equal(t, "project", proj.Name)
	assert.Equal(t, "test", proj.Runtime.Name())
}
