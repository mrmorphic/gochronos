package gochronos

import (
	"sync"
	"time"
)

// ActionFunc is basically a function to call when time is up, with optional parameters supplied when
// scheduled action was added.
type ActionFunc func(args ...interface{})

// A specification of when to execute an action. This can either be one-off, created by NewOneOff(), or
// recurring, created by NewRecurring().
type TimeSpec struct {
	recurring bool
	when      time.Time
}

// ScheduledAction represents an action that is scheduled in time. When added to the schedule,
// it will execute in accordance with the time specification.
type ScheduledAction struct {
	// @todo consider sync.Mutex if the goroutine can modify the struct.
	// specification of when the action should trigger
	When *TimeSpec

	// The action to invoke when time is met
	Action ActionFunc

	// Parameters passed to the action.
	Parameters []interface{}
}

// A list of scheduled actions. This is the schedule that is executed.
var schedule map[*ScheduledAction]bool

// This is used to synchronise updates to the schedule across goroutines.
var scheduleLock sync.Mutex

func init() {
	ClearAll()
}

// create a new scheduled action. To add to the schedule, call AddToScheduled, or just Add which creates
// and adds to schedule.
func NewScheduledAction(ts *TimeSpec, f ActionFunc, args []interface{}) *ScheduledAction {
	return &ScheduledAction{When: ts, Action: f, Parameters: args}
}

// Add a scheduled action to the schedule
func AddToSchedule(sa *ScheduledAction) {
	scheduleLock.Lock()

	// add a scheduled action to the list
	schedule[sa] = true

	scheduleLock.Unlock()

	sa.startTimer()
}

func Add(ts *TimeSpec, f ActionFunc, args ...interface{}) *ScheduledAction {
	sa := NewScheduledAction(ts, f, args)
	AddToSchedule(sa)
	return sa
}

func RemoveFromSchedule(sa *ScheduledAction) {
	// Tell the timer goroutine to stop. This in turn will trigger the goroutine to remove itself.
	sa.stopTimer()

	// add a scheduled action to the list
	// create a channel to listen for events from triggers
	// create a goroutine that sends back execution triggers.
	// On closure of a channel, remove the scheduled item
}

// Remove scheduled action from list. This assumes the timer goroutine
// is not going to trigger more events. This can be called by the timer
// goroutines when they reach termination, so locking is required on the structure.
func remove(sa *ScheduledAction) {
	scheduleLock.Lock()

	delete(schedule, sa)

	scheduleLock.Unlock()
}

// @todo SetTimeSpec should cause the timer to re-evaluate if executing
func (sa *ScheduledAction) SetTimeSpec(ts *TimeSpec) {
	sa.When = ts
}

func (sa *ScheduledAction) SetAction(f ActionFunc) {
	sa.Action = f
}

func (sa *ScheduledAction) SetParams(args ...interface{}) {
	sa.Parameters = args
}

// Given a scheduled action, start a goroutine for executing.
func (sc *ScheduledAction) startTimer() {
	go func() {
		for t := sc.When.GetNextExec(); !t.IsZero(); {
			d := t.Sub(time.Now())
			if d < 0 {
				break
			}
			time.Sleep(d)
			sc.Action(sc.Parameters...)
		}
		remove(sc)
	}()
}

func (sc *ScheduledAction) stopTimer() {

}

// create a channel to listen for events from triggers
// create a goroutine that sends back execution triggers.
// On closure of a channel, remove the scheduled item

// starttime - (required) a date/time string parsed by strtotime that determines when the first execution of the action should be.
// frequency - (required) the frequency of execution. Valid values are "hourly", "minutely", "daily", "weekly".
// interval - (optional, default 10) a multiplier for frequency (e.g. "weekly" with interval of 2 is fortnight.)
// byday - (optional) a string or array of strings that define days of the week when the action is to be executed. Valid values are "su","mo","tu","we","th","fr","sa"
// byhour - (optional) an int or array of ints that define the hours of the day when the action is to be executed.
// byminute - (optional) an int or array of ints that define the minutes of the hours when the action is to be executed.

// start time:		time.Time
// frequency:		time.Duration
// start time:		now + time.Duration
// time.Duration
// time.Time
// time.Time

func NewOneOff(t time.Time) *TimeSpec {
	return &TimeSpec{recurring: false, when: t}
}

// Create a new recurring time specification
func NewRecurring(config map[string]interface{}) *TimeSpec {
	return &TimeSpec{recurring: true}
}

// Given the current time, evaluate what the next execution time is according to the time spec.
// Logic is as follows:
// - if timespec is one-off:
//   - if the time is in the past, return the zero value for Time. Past scheduled events are not executed.
//   - otherwise return the time
// - if timespec is recurring:
//   - if termination condition is met, return the zero value for Time.
//   - compute forward from the start date, finding the closest date in the future that meets the spec, and return that.
func (t *TimeSpec) GetNextExec() time.Time {
	now := time.Now()

	if t.recurring {
		// @todo implement
		return time.Time{}
	} else {
		if t.when.Before(now) {
			return time.Time{}
		}
		return t.when
	}
}

// func Replace() {

// }

// Register an instance of a type that might be used for schedule. This is required if actions
// are being serialised, so that when deserialising, we know how to treat
// func RegisterType(Action) {

// }

// Clear the schedule of all scheduled actions
func ClearAll() {
	schedule = make(map[*ScheduledAction]bool)
}
