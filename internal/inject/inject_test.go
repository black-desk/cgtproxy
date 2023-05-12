package inject_test

import (
	"testing"

	"github.com/black-desk/deepin-network-proxy-manager/internal/inject"
)

func TestInjectInt(t *testing.T) {
	container := inject.New()

	var err error

	err = container.Register(1)
	if err != nil {
		t.Fatalf("%v", err)
	}

	if err := container.Register(2); err == nil {
		t.Fatalf("%v", err)
	}

	x := 3

	err = container.Fill(&x)
	if err != nil {
		t.Fatalf("%v", err)
	}

	if x != 1 {
		t.Fatalf("should get 1, but %v", x)
	}

	s := struct {
		A int `inject:"true"`
	}{}

	err = container.Fill(&s)
	if err != nil {
		t.Fatalf("%v", err)
	}

	if s.A != 1 {
		t.Fatalf("should get 1, but %v", s.A)
	}

}
