package logging

import (
	"fmt"
	"testing"
)

func TestErrorMessage(t *testing.T) {
	msg := newErrorMessage(fmt.Errorf("just an error"))
	AssertMessageFields(t, msg, StandardVariableFields, map[string]interface{}{
		//pkg/errors errors will yield a backtrace here
		"sous-error-backtrace": "just an error",
		"@loglov3-otl":         "sous-error-v1",
		"sous-error-msg":       "just an error",
	})
}
