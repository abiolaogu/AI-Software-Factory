package spec

import (
	"github.com/getkin/kin-openapi/openapi3"
	embedded "github.com/stratogroup/neuroforge-sdx/api/openapi"
)

func YAML() ([]byte, error) {
	return embedded.YAML()
}

func JSON() ([]byte, error) {
	return embedded.JSON()
}

func Load() (*openapi3.T, error) {
	return embedded.Load()
}
