package validator

import (
	"errors"
	"strconv"
)

var (
	StatusCodeIsNotNumericError = errors.New("metricsservice.Metrics{}.IncStatus(path string, method string, " +
		"-> status string) an HTTP status code accepts only codes which represents as numeric string")
	StatusCodeIsOutOfRangeError = errors.New("metricsservice.Metrics{}.IncStatus(path string, method string, " +
		"-> status string) an HTTP status code accepts only codes in range from 100 to 599")
)

func ValidateStrStatusCode(status string) error {
	if statusInt, err := strconv.Atoi(status); err != nil {
		return StatusCodeIsNotNumericError
	} else if statusInt < 100 || statusInt > 599 {
		return StatusCodeIsOutOfRangeError
	}
	return nil
}
