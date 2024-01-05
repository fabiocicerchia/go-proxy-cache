package jwt

import (
	"testing"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/utils/slice"
)

func TestContains(t *testing.T) {

	v := []string{"a", "b"}
	res := slice.ContainsString(v, "a")
	if !res {
		t.Error("Expected true but got", res)
	}

	res = slice.ContainsString(v, "c")
	if res {
		t.Error("Expected false but got", res)
	}
	res = slice.ContainsString(v, "")
	if res {
		t.Error("Expected false but got", res)
	}

	v = []string{}
	res = slice.ContainsString(v, "a")
	if res {
		t.Error("Expected false but got", res)
	}

	res = slice.ContainsString(v, "")
	if res {
		t.Error("Expected false but got", res)
	}

}

func TestIsIncluded(t *testing.T) {
	co = &config.Jwt{IncludedPaths: []string{"/a"}}
	res := IsIncluded(co.IncludedPaths, "/a")
	if !res {
		t.Error("Expected true but got", res)
	}

	res = IsIncluded(co.IncludedPaths, "/b")
	if res {
		t.Error("Expected false but got", res)
	}
	co = &config.Jwt{IncludedPaths: []string{}}
	res = IsIncluded(co.IncludedPaths, "/b")
	if res {
		t.Error("Expected false but got", res)
	}

}
