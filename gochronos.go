package gochronos

import (
	// "fmt"
	"sync"
	"time"
)

const (
	FREQ_SECOND int = 1 + iota
	FREQ_MINUTE
	FREQ_HOUR
	FREQ_DAY
	FREQ_WEEK
	FREQ_MONTH
	FREQ_YEAR
)

// A command that can be sent to a goroutine.
type command int

const (
	// Cancel the goroutine for a scheduled action
	CMD_CANCEL command = 1 + iota
)

// ActionFunc is basically a function to call when time is up, with optional parameters supplied when
// scheduled action was added.
type ActionFunc func(args ...interface{})

// A specification of when to execute an action. This can either be one-off, created by NewOneOff(), or
// recurring, created by NewRecurring().
type TimeSpec struct {
	recurring bool
	when      time.Time

	startTime time.Time
	endTime   time.Time
	frequency int // one of FREQ_ constants
	interval  int
	// byday
	// byhours
	// byminute
	maxNum int
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

	cmdChan chan command
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

// Add a scheduled action to the schedule.
func Add(ts *TimeSpec, f ActionFunc, args ...interface{}) *ScheduledAction {
	sa := NewScheduledAction(ts, f, args)
	AddToSchedule(sa)
	return sa
}

// Remove a scheduled action from the schedule.
func Remove(sa *ScheduledAction) {
	// Tell the timer goroutine to stop. This in turn will trigger the goroutine to remove itself.
	sa.stopTimer()
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
	sc.cmdChan = make(chan command)
	go func() {
		var timer *time.Timer

	loop:
		for t := sc.When.GetNextExec(); !t.IsZero(); {
			d := t.Sub(time.Now())
			if d < 0 {
				d = 0
			}

			// create the time first time around, or reset it if we're re-using it.
			if timer == nil {
				timer = time.NewTimer(d)
			} else {
				timer.Reset(d)
			}

			// wait for either the time, or a command from the command channel
			select {
			case _ = <-timer.C:
				// when timer goes off, we execute the action and repeat the loop
				sc.Action(sc.Parameters...)
			case cmd := <-sc.cmdChan:
				if cmd == CMD_CANCEL {
					timer.Stop()
					break loop
				}
			}
			t = sc.When.GetNextExec()
		}
		remove(sc)
	}()
}

// Stop a scheduled action.
func (sc *ScheduledAction) stopTimer() {
	// send cancel command to the goroutine
	sc.cmdChan <- CMD_CANCEL
}

// Create a new one-off time specification from a Time.
func NewOneOff(t time.Time) *TimeSpec {
	return &TimeSpec{recurring: false, when: t}
}

// Create a new recurring time specification from a map.
func NewRecurring(config map[string]interface{}) *TimeSpec {
	result := &TimeSpec{
		recurring: true,
		interval:  1,
		startTime: time.Time{},
		endTime:   time.Time{},
		frequency: -1,
		maxNum:    -1,
	}

	for k, v := range config {
		switch k {
		case "starttime": // expect time
			result.startTime = v.(time.Time)
		case "frequency": // expect int, which should be a FREQ_* constant
			result.frequency = v.(int)
		case "interval": // expect int: multiplier for frequency e.g. 2 week is a fortnight
			result.interval = v.(int)
		// case "byday": // - (optional) a string or array of strings that define days of the week when the action is to be executed. Valid values are "su","mo","tu","we","th","fr","sa"
		// case "byhours": // byhour - (optional) an int or array of ints that define the hours of the day when the action is to be executed.
		// case "byminute": // - (optional) an int or array of ints that define the minutes of the hours when the action is to be executed
		case "endtime": // expect time
			result.endTime = v.(time.Time)
		case "maxnum": // expect int
			result.maxNum = v.(int)
		}
	}

	// ensure startime and frequency are provided.
	if result.startTime.IsZero() {
		panic("recurring scheduled action must have a start date")
	}
	if result.frequency < FREQ_SECOND || result.frequency > FREQ_YEAR {
		panic("recurring scheduled action must have a frequency")
	}

	return result
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
		// if termination condition is met, return zero time
		if !t.endTime.IsZero() && t.endTime.Before(now) {
			return time.Time{}
		}

		// if start time is in the future, return that
		if t.startTime.After(now) {
			return t.startTime
		}

		// determine period in seconds
		period := 0
		switch t.frequency {
		case FREQ_SECOND:
			period = 1
		case FREQ_MINUTE:
			period = 60
		case FREQ_HOUR:
			period = 3600
		case FREQ_DAY:
			period = 86400
		case FREQ_WEEK:
			period = 604800
		}

		if period > 0 {
			// it's a fixed number of seconds period, which excludes months and years
			period *= t.interval

			// @todo take into account byday, byhour, byminute
			delta := now.Sub(t.startTime) // difference between start and now.
			td := int(delta*time.Second) % period
			prev := time.Unix(now.Unix()-int64(td), 0)
			next := prev.Add(time.Duration(period) * time.Second)
			return next
		}

		// @todo implement month and year
		switch t.frequency {
		case FREQ_MONTH:
		case FREQ_YEAR:
		}

		return time.Time{}
	} else {
		if t.when.Before(now) {
			return time.Time{}
		}
		return t.when
	}
}

// Register an instance of a type that might be used for schedule. This is required if actions
// are being serialised, so that when deserialising, we know how to treat
// func RegisterType(Action) {

// }

// Clear the schedule of all scheduled actions.
// @todo if schedule is already defined and there are executing scheduled actions, terminate them so they're GC'd.
func ClearAll() {
	schedule = make(map[*ScheduledAction]bool)
}
