package inject_test

import (
	"testing"

	"github.com/black-desk/deepin-network-proxy-manager/internal/inject"
)

func TestInjectInt(t *testing.T) {
	container := inject.New()

	if err := container.Register(1); err != nil {
		t.Fatalf("%v", err)
	}

	if err := container.Register(2); err == nil {
		t.Fatalf("%v", err)
	}

	x := 3

	if err := container.Fill(&x); err != nil {
		t.Fatalf("%v", err)
	}

	if x != 1 {
		t.Fatalf("should get 1, but %v", x)
	}

	s := struct {
		A int `inject:"true"`
	}{}

	if err := container.Fill(&s); err != nil {
		t.Fatalf("%v", err)
	}

	if s.A != 1 {
		t.Fatalf("should get 1, but %v", s.A)
	}

}
