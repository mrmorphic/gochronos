package gochronos

import (
	"testing"
	"time"
)

func TestAdd(t *testing.T) {
	count := 0
	param1 := ""
	param2 := 0

	// Add a new one-off action. The action will count the number of times executed, and will set
	// properties based on parameters.
	Add(NewOneOff(time.Now().Add(time.Second)),
		func(args ...interface{}) {
			param1 = args[0].(string)
			param2 = args[1].(int)
			count++
		},
		"test", 5)

	// kill all scheduled actions
	time.Sleep(time.Second * 3)

	if count != 1 {
		t.Errorf("Expected one-off action to be executed exactly once, was executed %d times", count)
	}

	if param1 != "test" {
		t.Errorf("Expected first parameter to be 'test', was actually %s", param1)
	}

	if param2 != 5 {
		t.Errorf("Expected second parameter to be 5, was actually %d", param2)
	}

	if len(schedule) > 0 {
		t.Errorf("Expected schedule to empty, contains %d item(s)", len(schedule))
	}

	ClearAll()
}
