package micro

import (
	"fmt"
	"os"
	"testing"
)

func TestConfig(t *testing.T) {
	str := `
		aaa:${env}
		bbb:${ de_mo }
		ccc:${string}
`
	os.Setenv("env", "env-test")
	os.Setenv("de_mo", "demo-test")
	os.Setenv("string", "string-test")

	aaa := envYamlReplace(str)
	fmt.Println("|", aaa, "|")
}
