package asyncmsg

import (
	"errors"
	"fmt"
	"log"
)

var RecoverPanicToErr = true

func PanicValToErr(panicVal interface{}, err *error) {
	if panicVal == nil {
		return
	}
	// case nil
	switch xErr := panicVal.(type) {
	case error:
		*err = xErr
	case string:
		*err = errors.New(xErr)
	default:
		*err = fmt.Errorf("%v", panicVal)
	}
	return
}

func PanicToErr(err *error) {
	if RecoverPanicToErr {
		if x := recover(); x != nil {
			//debug.PrintStack()
			PanicValToErr(x, err)
		}
	}
}

func PrintPanicStack() {
	if x := recover(); x != nil {
		log.Printf("Recovered %v", x)
	}
}
