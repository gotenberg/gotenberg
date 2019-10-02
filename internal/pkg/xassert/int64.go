package xassert

import (
	"fmt"

	"github.com/thecodingmachine/gotenberg/internal/pkg/xerror"
)

// RuleInt64 is an interface for
// validating an int64.
type RuleInt64 interface {
	with(key string, value int64)
	validate() error
}

type baseRuleInt64 struct {
	key   string
	value int64
}

func (r *baseRuleInt64) with(key string, value int64) {
	r.key = key
	r.value = value
}

type ruleInt64NotInferiorTo struct {
	*baseRuleInt64
	lowerBound int64
}

func (r ruleInt64NotInferiorTo) validate() error {
	const op string = "xassert.ruleInt64NotInferiorTo.validate"
	if r.value < r.lowerBound {
		return xerror.Invalid(
			op,
			fmt.Sprintf("'%s' should be > '%d', got '%d'", r.key, r.lowerBound, r.value),
			nil,
		)
	}
	return nil
}

/*
Int64NotInferiorTo returns a RuleInt64 for
validating that an int64 is not inferior to
given lower bound.
*/
func Int64NotInferiorTo(lowerBound int64) RuleInt64 {
	return &ruleInt64NotInferiorTo{
		&baseRuleInt64{},
		lowerBound,
	}
}

type ruleInt64NotSuperiorTo struct {
	*baseRuleInt64
	upperBound int64
}

func (r ruleInt64NotSuperiorTo) validate() error {
	const op string = "xassert.ruleInt64NotSuperiorTo.validate"
	if r.value > r.upperBound {
		return xerror.Invalid(
			op,
			fmt.Sprintf("'%s' should be < '%d', got '%d'", r.key, r.upperBound, r.value),
			nil,
		)
	}
	return nil
}

/*
Int64NotSuperiorTo returns a RuleInt64 for
validating that an int64 is not superior to
given upper bound.
*/
func Int64NotSuperiorTo(upperBound int64) RuleInt64 {
	return ruleInt64NotSuperiorTo{
		&baseRuleInt64{},
		upperBound,
	}
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = RuleInt64(new(ruleInt64NotInferiorTo))
	_ = RuleInt64(new(ruleInt64NotSuperiorTo))
)
