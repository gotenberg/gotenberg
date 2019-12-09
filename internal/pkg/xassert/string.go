package xassert

import (
	"fmt"
	"strings"

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

type ruleStringStartWith struct {
	*baseRuleString
	startWith string
}

func (r ruleStringStartWith) validate() error {
	const op string = "xassert.ruleStringStartWith.validate"
	if strings.HasPrefix(r.value, r.startWith) {
		return nil
	}
	return xerror.Invalid(
		op,
		fmt.Sprintf("'%s' should start with '%s', got '%s'", r.key, r.startWith, r.value),
		nil,
	)
}

/*
StringStartWith returns a RuleString for
validating that a string starts with
given string.
*/
func StringStartWith(startWith string) RuleString {
	return ruleStringStartWith{
		&baseRuleString{},
		startWith,
	}
}

type ruleStringEndWith struct {
	*baseRuleString
	endWith string
}

func (r ruleStringEndWith) validate() error {
	const op string = "xassert.ruleStringEndWith.validate"
	if strings.HasSuffix(r.value, r.endWith) {
		return nil
	}
	return xerror.Invalid(
		op,
		fmt.Sprintf("'%s' should end with '%s', got '%s'", r.key, r.endWith, r.value),
		nil,
	)
}

/*
StringEndWith returns a RuleString for
validating that a string ends with
given string.
*/
func StringEndWith(endWith string) RuleString {
	return ruleStringEndWith{
		&baseRuleString{},
		endWith,
	}
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = RuleString(new(ruleStringOneOf))
	_ = RuleString(new(ruleStringStartWith))
	_ = RuleString(new(ruleStringEndWith))
)
