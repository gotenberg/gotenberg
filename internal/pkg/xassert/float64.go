package xassert

import (
	"fmt"

	"github.com/thecodingmachine/gotenberg/internal/pkg/xerror"
)

// RuleFloat64 is an interface for
// validating a float64.
type RuleFloat64 interface {
	with(key string, value float64)
	validate() error
}

type baseRuleFloat64 struct {
	key   string
	value float64
}

func (r *baseRuleFloat64) with(key string, value float64) {
	r.key = key
	r.value = value
}

type ruleFloat64NotInferiorTo struct {
	*baseRuleFloat64
	lowerBound float64
}

func (r ruleFloat64NotInferiorTo) validate() error {
	const op string = "xassert.ruleFloat64NotInferiorTo.validate"
	if r.value < r.lowerBound {
		return xerror.Invalid(
			op,
			fmt.Sprintf("'%s' should be > '%f', got '%f'", r.key, r.lowerBound, r.value),
			nil,
		)
	}
	return nil
}

/*
Float64NotInferiorTo returns a RuleFloat64 for
validating that a float64 is not inferior to
given lower bound.
*/
func Float64NotInferiorTo(lowerBound float64) RuleFloat64 {
	return ruleFloat64NotInferiorTo{
		&baseRuleFloat64{},
		lowerBound,
	}
}

type ruleFloat64NotSuperiorTo struct {
	*baseRuleFloat64
	upperBound float64
}

func (r ruleFloat64NotSuperiorTo) validate() error {
	const op string = "xassert.ruleFloat64NotSuperiorTo.validate"
	if r.value > r.upperBound {
		return xerror.Invalid(
			op,
			fmt.Sprintf("'%s' should be < '%f', got '%f'", r.key, r.upperBound, r.value),
			nil,
		)
	}
	return nil
}

/*
Float64NotSuperiorTo returns a RuleFloat64 for
validating that a float64 is not superior to
given upper bound.
*/
func Float64NotSuperiorTo(upperBound float64) RuleFloat64 {
	return ruleFloat64NotSuperiorTo{
		&baseRuleFloat64{},
		upperBound,
	}
}

// Compile-time checks to ensure type implements desired interfaces.
var (
	_ = RuleFloat64(new(ruleFloat64NotInferiorTo))
	_ = RuleFloat64(new(ruleFloat64NotSuperiorTo))
)
