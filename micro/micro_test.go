package micro

import (
	"fmt"
	"os"
	"testing"
)

func TestConfig(t *testing.T) {
	str := `
		aaa:${env}
		bbb:${ demo }
		ccc:${string}
`
	os.Setenv("env", "env-test")
	os.Setenv("demo", "demo-test")
	os.Setenv("string", "string-test")

	aaa := envYamlReplace(str)
	fmt.Println("|", aaa, "|")
}
