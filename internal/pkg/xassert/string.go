package xassert

import (
	"fmt"

	"github.com/thecodingmachine/gotenberg/internal/pkg/xerror"
)

// RuleString is an interface for
// validating a string.
type RuleString interface {
	with(key, value string)
	validate() error
}

type baseRuleString struct {
	key   string
	value string
}

func (r *baseRuleString) with(key, value string) {
	r.key = key
	r.value = value
}

type ruleStringOneOf struct {
	*baseRuleString
	values []string
}

func (r ruleStringOneOf) validate() error {
	const op string = "xassert.ruleStringOneOf.validate"
	for _, v := range r.values {
		if r.value == v {
			return nil
		}
	}
	return xerror.Invalid(
		op,
		fmt.Sprintf("'%s' should be one of '%v', got '%s'", r.key, r.values, r.value),
		nil,
	)
}

/*
StringOneOf returns a RuleString for
validating that a string is one of given
values.
*/
func StringOneOf(values []string) RuleString {
	return ruleStringOneOf{
		&baseRuleString{},
		values,
	}
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = RuleString(new(ruleStringOneOf))
)
