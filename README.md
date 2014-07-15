# About gochronos

gochronos is a go library for creating and executing things to a programmable schedule. gochronos allows changing the schedule within the app at any point,
and is designed for making data-driven scheduling easy.

The schedule persists for the current process; an application that imports
gochronos needs to persist scheduled items itself if they need to persist
beyond application execution.

The module is currently in development. See the "Status" section to see what is currently working and not working.

# Basic Usage

The most basic usage is a one-time scheduled action. The following adds a new
one-time scheduled action, to be executed in 2 hours.

    when := time.Now().Add(2 * time.Hour)
    gochronos.Add(gochronos.NewOneOff(when),
            func(args ...interface{}) {
                s := args[0].(string)
                // do something here
            },
            "somearg")

When adding a scheduled action, you specify the parameters to be passed to the function when it executes. The parameters are optional, but can be useful when there is a generic handler function that can consume the parameters, for different behaviours.

Recurring scheduled actions are also possible, and are fairly flexible in how the occurences are specified.

    timeSpec := gochronos.NewRecurring(map[string]interface{}{
        "starttime": time.Now(),
        "frequency": gochronos.FREQ_MINUTE
    })
    gochronos.Add(timeSpec,
            func(args ...interface{}) {
                // do something here
            })

NewRecurring() generates a recurring time specification. It accepts a
map[string]interface{} which contains entries that specify various properties of the time specification. The above example shows a time specification that starts right now, and executes every hour.

The following properties are currently understood by NewRecurring:

 *  **starttime** - (required) a time.Time value, which is a reference
    time when the first action is executed. This may be in the past; actions
    are triggered at the specified frequency from this time.
 *  **frequency** - (required) one of the gochronos.FREQ_* constants. This
    indicates how frequently the action should occur.
 *  **interval** - (optional, default is 1) a multiplier on frequency. E.g. if
    frequency is FREQ_MINUTE and interval is 3, the action will occur every
    3 minutes.
 *  **endtime** - (optional) a time.Time value, after which no actions should
    occur. The default is no end time, so actions will continue according at
    the required frequency until the program is stopped, or the scheduled
    action removed from the schedule.

gochronos.Add() returns a *ScheduledAction value. This can be used to cancel
it, using Remove(), or change the ScheduledAction properties.

# Persisting the schedule

Currently, the module does not support persisting the schedule, so that a program being restarted can pick up where it left off. This is a planned feature.

# How it Works

Each scheduled action is added to a data structure. A new goroutine is created or each one of them, which determines when it needs to execute it's action, and sleep until that point.

For one-off scheduled items, once the goroutine has executed the action at the appropriate time, the goroutine removes the scheduled action from the schedule, and completes.

The process is almost the same for repeat items, except that after executing the action, it determines if there are more scheduled times to execute, and if so, goes back to sleep until that time.

# Other features for consideration

 *  Logging - although to some degree, this is up to the app, which can wrap
    the action in a logging action.

# Status

## Working

 *  One-time scheduled actions work correctly, and clean up afterwards
 *  One-time scheduled actions are unit tested, including parameters.
 *  Cancellng one-time actions before they execute
 *  Recurring scheduled actions for second, minute, hour, day and week. Only
    'second' is unit tested.

## Not Test

 *  Cancelling recurring actions after at least one iteration

## Not Implemented

 *  Recurring with month or year frequency
 *  Recurring with maxnum
 *  Recurring, by minutes, by hours, by days etc
 *  If scheduled action properties are changed once the goroutine
    is started, changes won't take effect. This requires a command to be
    sent to the goroutine telling it to refresh.