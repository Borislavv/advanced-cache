package liveness

import "errors"

var TimeoutIsTooShortError = errors.New("liveness probe timeout is too short")
