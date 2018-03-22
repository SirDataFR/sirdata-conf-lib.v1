package sirdataConf

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

const (
	IntUndefined = -1
)

type Checker struct {
	config    interface{}
	tagSpec   func() (string, string)
	verifiers []verifier
}

func (c *Checker) GetTagKey(searched interface{}) (string, bool) {
	return GetTagKey(c.tagSpec, c.config, searched)
}

func (c *Checker) tagKeyOrValue(searched interface{}) string {
	if tag, found := GetTagKey(c.tagSpec, c.config, searched); found {
		return tag
	}
	return fmt.Sprintf(" entry holding value %s", searched)
}

func NewChecker(config interface{}, tagSpec func() (string, string)) Checker {
	value := reflect.ValueOf(config)
	if value.Kind() != reflect.Ptr || value.Elem().Kind() != reflect.Struct {
		panic("Illegal config: must be a pointer to a structure")
	}
	return Checker{config, tagSpec, make([]verifier, 0)}
}

func NewYamlChecker(config interface{}) Checker {
	return NewChecker(config, func() (string, string) { return "yaml", "." })
}

func NewJsonChecker(config interface{}) Checker {
	return NewChecker(config, func() (string, string) { return "json", "." })
}

type verifier interface {
	verify() (bool, []error)
}

func (c *Checker) addVerifier(verifier verifier) {
	c.verifiers = append(c.verifiers, verifier)
}

func (c *Checker) Verify() (bool, []error) {
	ok := true
	errs := make([]error, 0)
	for _, verifier := range c.verifiers {
		if isOk, verifyErrors := verifier.verify(); !isOk {
			ok = false
			errs = append(errs, verifyErrors...)
		}
	}
	if ok {
		return true, nil
	}
	return false, errs
}

type conditionVerifier struct {
	cond Condition
}

func (cv *conditionVerifier) verify() (bool, []error) {
	ok := cv.cond.Evaluate()
	if ok {
		return true, nil
	}
	return false, []error{errors.New(cv.cond.Desc())}
}

type Condition interface {
	Desc() string
	Evaluate() bool
}

type condition struct {
	desc     func() string
	evaluate func() bool
}

func (c condition) Desc() string   { return c.desc() }
func (c condition) Evaluate() bool { return c.evaluate() }

func When(desc func() string, evaluate func() bool) Condition {
	return condition{desc, evaluate}
}

func Or(conds ...Condition) Condition {
	descs, evals := splitConds(conds)
	return condition{
		desc: func() string {
			return strings.Join(evalDescs(descs), " or ")
		},
		evaluate: func() bool {
			for _, eval := range evals {
				if eval() {
					return true
				}
			}
			return false
		},
	}
}

func And(conds ...Condition) Condition {
	descs, evals := splitConds(conds)
	return condition{
		desc: func() string {
			return strings.Join(evalDescs(descs), " and ")
		},
		evaluate: func() bool {
			for _, eval := range evals {
				if !eval() {
					return false
				}
			}
			return true
		},
	}
}

func evalDescs(descs []func() string) []string {
	descsEval := make([]string, len(descs))
	for i, desc := range descs {
		descsEval[i] = desc()
	}
	return descsEval
}

func splitConds(conds []Condition) ([]func() string, []func() bool) {
	descs := make([]func() string, len(conds))
	evals := make([]func() bool, len(conds))
	for index, cond := range conds {
		descs[index] = cond.Desc
		evals[index] = cond.Evaluate
	}
	return descs, evals
}

func (c *Checker) StringEquals(entry *string, value string) Condition {
	return When(
		func() string { return fmt.Sprintf("%s=%s", c.tagKeyOrValue(entry), value) },
		func() bool { return *entry == value })
}

func (c *Checker) StringNotEmpty(entry *string) Condition {
	return When(
		func() string { return fmt.Sprintf("%s is not empty", c.tagKeyOrValue(entry)) },
		func() bool { return *entry != "" })
}

func (c *Checker) StringIn(entry *string, values []string) Condition {
	return When(
		func() string { return fmt.Sprintf("%s in contained in %s", c.tagKeyOrValue(entry), values) },
		func() bool {
			for _, value := range values {
				if value == *entry {
					return true
				}
			}
			return false
		})
}

func (c *Checker) IntEquals(entry *int, value int) Condition {
	return When(
		func() string { return fmt.Sprintf("%s=%d", c.tagKeyOrValue(entry), value) },
		func() bool { return *entry == value })
}

func (c *Checker) BoolEquals(entry *bool, value bool) Condition {
	return When(
		func() string { return fmt.Sprintf("%s=%t", c.tagKeyOrValue(entry), value) },
		func() bool { return *entry == value })
}

func (c *Checker) AddCondition(evaluate func() bool, desc func() string) {
	var verifier verifier = &conditionVerifier{condition{desc, evaluate}}
	c.addVerifier(verifier)
}

func (c *Checker) StringCondition(evaluate func() bool, desc func() string) {
	c.AddCondition(evaluate, desc)
}

func (c *Checker) IntCondition(evaluate func() bool, desc func() string) {
	c.AddCondition(evaluate, desc)
}

func (c *Checker) StringMandatory(entry *string) {
	c.StringCondition(func() bool { return *entry != "" }, c.describeMandatory(entry))
}

func (c *Checker) StringMandatoryWhen(entry *string, when Condition) {
	c.StringCondition(c.evaluateStringMandatoryWhen(entry, when), c.describeWhen(c.describeMandatory(entry), when))
}

func (c *Checker) StringXor(first *string, second *string) {
	c.StringCondition(func() bool { return (*first == "") != (*second == "") }, c.describeXor(first, second))
}

func (c *Checker) StringXorWhen(first *string, second *string, when Condition) {
	c.StringCondition(c.evaluateStringXorWhen(first, second, when), c.describeXor(first, second))
}

func (c *Checker) StringPattern(entry *string, pattern string) error {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	c.StringCondition(
		func() bool { return regex.Match([]byte(*entry)) },
		func() string { return fmt.Sprintf("%s must match pattern %s", c.tagKeyOrValue(entry), pattern) })
	return nil
}

func (c *Checker) EnumOptional(entry *string, values []string, def string) {
	cond := c.enumValueCondition(entry, values)
	c.AddCondition(cond.Evaluate, cond.Desc)
}

func (c *Checker) EnumMandatory(entry *string, values []string) {
	cond := And(
		condition{
			c.describeMandatory(entry),
			func() bool { return *entry != "" },
		},
		c.enumValueCondition(entry, values),
	)
	c.AddCondition(cond.Evaluate, cond.Desc)
}

func (c *Checker) EnumMandatoryWhen(entry *string, values []string, when Condition) {
	cond := And(
		condition{
			c.describeWhen(c.describeMandatory(entry), when),
			c.evaluateStringMandatoryWhen(entry, when),
		},
		c.enumValueCondition(entry, values),
	)
	c.AddCondition(cond.Evaluate, cond.Desc)
}

func (c *Checker) evaluateStringMandatoryWhen(entry *string, when Condition) func() bool {
	// either the entry is not empty, either the "when" condition must be false since it indicates that the
	// entry must be populated in that case
	return func() bool { return *entry != "" || !when.Evaluate() }
}

func (c *Checker) evaluateStringXorWhen(first *string, second *string, when Condition) func() bool {
	return func() bool { return ((*first == "") != (*second == "")) || !when.Evaluate() }
}

func (c *Checker) enumValueCondition(entry *string, values []string) Condition {
	return condition{
		func() string { return fmt.Sprintf("value of %s must be in %s", c.tagKeyOrValue(entry), values) },
		func() bool {
			if *entry == "" {
				return true
			}
			for _, value := range values {
				if value == *entry {
					return true
				}
			}
			return false
		},
	}
}

func (c *Checker) IntMandatory(entry *int) {
	c.IntCondition(func() bool { return *entry != IntUndefined }, c.describeMandatory(entry))
}

func (c *Checker) IntMandatoryWhen(entry *int, when Condition) {
	c.IntCondition(
		func() bool { return *entry != IntUndefined || !when.Evaluate() },
		c.describeWhen(c.describeMandatory(entry), when))
}

func (c *Checker) describeMandatory(searched interface{}) func() string {
	return func() string { return fmt.Sprintf("%s is mandatory", c.tagKeyOrValue(searched)) }
}

func (c *Checker) describeXor(first interface{}, second interface{}) func() string {
	return func() string {
		return fmt.Sprintf("either %s or %s is mandatory, but only one can be set", c.tagKeyOrValue(first), c.tagKeyOrValue(second))
	}
}

func (c *Checker) describeXorWhen(first interface{}, second interface{}, when Condition) func() string {
	return func() string {
		return fmt.Sprintf("%s when %s", c.describeXor(first, second)(), when.Desc())
	}
}

func (c *Checker) describeWhen(desc func() string, when Condition) func() string {
	return func() string { return fmt.Sprintf("%s when %s", desc(), when.Desc()) }
}
