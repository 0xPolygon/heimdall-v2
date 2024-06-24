package types

import "fmt"

// Default parameter values
const (
	DefaultSprintDuration    uint64 = 16
	DefaultSpanDuration      uint64 = 100 * DefaultSprintDuration
	DefaultFirstSpanDuration uint64 = 256
	DefaultProducerCount     uint64 = 4
)

// NewParams creates a new bor Params object
func NewParams(sprintDuration uint64, spanDuration uint64, producerCount uint64) Params {
	return Params{
		SprintDuration: sprintDuration,
		SpanDuration:   spanDuration,
		ProducerCount:  producerCount,
	}
}

// DefaultParams returns default parameters for bor module
func DefaultParams() Params {
	return Params{
		SprintDuration: DefaultSprintDuration,
		SpanDuration:   DefaultSpanDuration,
		ProducerCount:  DefaultProducerCount,
	}
}

// Validate checks that the bor parameters have valid values.
func (p Params) Validate() error {
	if err := validateSprintDuration(p.SprintDuration); err != nil {
		return err
	}

	if err := validateSpanDuration(p.SprintDuration); err != nil {
		return err
	}

	if err := validateProducerCount(p.SprintDuration); err != nil {
		return err
	}

	return nil
}

// validateSprintDuration checks if the sprint duration is valid
func validateSprintDuration(i interface{}) error {
	v, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid sprint duration parameter type: %T", i)
	}

	if v == 0 {
		return fmt.Errorf("invalid sprint duration: %d", v)
	}

	return nil
}

// validateSpanDuration checks if the span duration is valid
func validateSpanDuration(i interface{}) error {
	v, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid span duration parameter type: %T", i)
	}

	if v == 0 {
		return fmt.Errorf("invalid span duration: %d", v)
	}

	return nil
}

// validateProducerCount checks if the producer count is valid
func validateProducerCount(i interface{}) error {
	v, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid producer count parameter type: %T", i)
	}

	if v == 0 {
		return fmt.Errorf("invalid producers count: %d", v)
	}

	return nil
}
