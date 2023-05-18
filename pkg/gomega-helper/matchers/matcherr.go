package matchers

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/onsi/gomega/format"
)

type MatchErrMatcher struct {
	Expected interface{}
}

func (matcher *MatchErrMatcher) Match(
	actual interface{},
) (
	success bool, err error,
) {
	if actual == nil {
		success = matcher.Expected == nil
		return
	}

	if matcher.Expected == nil {
		success = false
		return
	}

	if _, ok := actual.(error); !ok {
		success = false
		err = fmt.Errorf("Expected an error.  Got:\n%s",
			format.Object(actual, 1),
		)
		return
	}

	actualErr := actual.(error)
	expected := matcher.Expected

	if _, ok := expected.(error); !ok {
		return false, fmt.Errorf(
			"MatchErr must be passed an error. Got:\n%s",
			format.Object(expected, 1))
	}

	if errors.Is(actualErr, expected.(error)) {
		return true, nil
	}

	rets := reflect.ValueOf(errors.As).Call(
		[]reflect.Value{
			reflect.ValueOf(actualErr),
			reflect.NewAt(
				reflect.ValueOf(expected).Type(),
				reflect.ValueOf(expected).UnsafePointer(),
			),
		},
	)
	if len(rets) == 1 && rets[0].Bool() == true {
		return true, nil
	}

	return false, nil
}

func (matcher *MatchErrMatcher) FailureMessage(actual interface{}) (message string) {
	return format.Message(actual, "to match error", matcher.Expected)
}

func (matcher *MatchErrMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return format.Message(actual, "not to match error", matcher.Expected)
}
