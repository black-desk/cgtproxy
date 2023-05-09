package inject

import (
	"fmt"
	"reflect"
	"sync"
	"unsafe"

	"github.com/black-desk/deepin-network-proxy-manager/internal/log"
)

type Container struct {
	store sync.Map
}

var defaultContainer = &Container{}

func Default() *Container {
	return defaultContainer
}

func New() *Container {
	return &Container{}
}

func (c *Container) Register(v any) (err error) {
	rtype := reflect.TypeOf(v)
	if _, loaded := c.store.LoadOrStore(rtype, reflect.ValueOf(v)); loaded {
		err = fmt.Errorf(`type "%s" had been registered.`, rtype.String())
		return
	} else {
		log.Debug().Printf(`register type "%s"`, rtype.String())
	}

	return
}

func (c *Container) RegisterI(ptrToI any) (err error) {
	rtype := reflect.TypeOf(ptrToI)
	if rtype.Kind() != reflect.Pointer {
		err = fmt.Errorf(`wrong type: %s`, rtype.String())
		return
	}

	elem := rtype.Elem()
	if elem.Kind() != reflect.Interface {
		err = fmt.Errorf(`wrong type: %s`, rtype.String())
		return
	}

	if _, loaded := c.store.LoadOrStore(elem, reflect.ValueOf(ptrToI).Elem()); loaded {
		err = fmt.Errorf(`interface "%s" had been registered.`, elem.String())
		return
	} else {
		log.Debug().Printf(`register interface "%s"`, elem.String())
	}

	return
}

func (c *Container) Fill(v any) (err error) {
	defer func() {
		if err == nil {
			return
		}

		err = fmt.Errorf("failed to fill %#v:\n%w", v, err)
	}()

	if v == nil {
		err = fmt.Errorf("fill should not take a nil.")
		return
	}

	rvalue := reflect.ValueOf(v)
	if rvalue.Kind() != reflect.Pointer {
		err = fmt.Errorf(`fill should always take a pointer as argument.`)
		return
	}

	elem := rvalue.Elem()
	if value, loaded := c.store.Load(elem.Type()); loaded {
		rvalue := reflect.ValueOf(v).Elem()
		rvalue.Set(value.(reflect.Value))
		return
	}

	if elem.Kind() != reflect.Struct {
		err = fmt.Errorf(`type %s not found in this container.`, elem.Type().String())
		return
	}

	for i := 0; i < elem.NumField(); i++ {
		if _, ok := elem.Type().Field(i).Tag.Lookup("inject"); !ok {
			continue
		}
		if err = c.Fill(
			reflect.NewAt(
				elem.Field(i).Type(),
				unsafe.Pointer(elem.Field(i).Addr().Pointer()),
			).Interface(),
		); err != nil {
			err = fmt.Errorf("failed on field %d:\n%w", i, err)
			return
		}
	}

	return nil
}
