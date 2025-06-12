package liveness

import "errors"

var TimeoutIsToShortError = errors.New("liveness probe timeout is too short")
