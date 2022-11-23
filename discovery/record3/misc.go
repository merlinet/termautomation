package record3

import (
	"regexp"
)

func IsNumeric(value Void) bool {
	switch value.(type) {
	case float64:
		return true
	default:
		return false
	}
}

func IsBool(value Void) bool {
	switch value.(type) {
	case bool:
		return true
	default:
		return false
	}
}

func IsString(value Void) bool {
	switch value.(type) {
	case string:
		return true
	default:
		return false
	}
}

func IsRegexp(value Void) bool {
	switch value.(type) {
	case *regexp.Regexp:
		return true
	default:
		return false
	}
}

func IsList(value Void) bool {
	switch value.(type) {
	case []Void:
		return true
	default:
		return false
	}
}

func IsMap(value Void) bool {
	switch value.(type) {
	case map[Void]Void:
		return true
	default:
		return false
	}
}

func IsNil(value Void) bool {
	switch value.(type) {
	case nil:
		return true
	default:
		return false
	}
}
