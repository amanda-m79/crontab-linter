# crontab-linter

Cron syntax is easy to get subtly wrong: a step value that's larger than
the field range, a range written backwards, a day-of-month and
day-of-week combination that behaves as OR instead of AND when you
expected AND. None of that fails loudly - the crontab loads fine, and
the job just runs at the wrong time (or never). This is a small
command-line tool that reads a crontab-style file and reports problems
by line number, before they hit a scheduler.

## Usage

```
go run . mycrontab
```

Given a file like this:

```
# mycrontab
PATH=/usr/bin:/bin

0 3 * * * /usr/bin/backup.sh
*/1 * * * * /usr/bin/heartbeat.sh
99 5 * * * /usr/bin/broken.sh
0 9-17 * * mon-fri /usr/bin/office-hours.sh
30 4 15 * 1 /usr/bin/ambiguous.sh
0 0 1 13 * /usr/bin/bad-month.sh
* * * * *
```

it prints:

```
line 5: minute: step /1 on * is redundant, same as *
line 6: minute: value 99 out of range 0-59
line 8: day-of-month and day-of-week are both restricted; most cron daemons treat this as OR, not AND
line 9: month: value 13 out of range 1-12
line 10: expected 5 time fields plus a command, found 5 fields
```

Findings are printed in the order the lines appear in the file, and a
line can produce more than one finding if several fields have problems.

Exit code is 1 if any findings were reported, 0 if the file is clean.
That makes it usable as a pre-commit or CI check on crontab files kept
in version control.

## What it checks today

- each of the 5 standard time fields (minute, hour, day-of-month,
  month, day-of-week) against its valid range, or the 6/7 Quartz
  fields (seconds, minute, hour, day-of-month, month, day-of-week,
  year) when the line has that many
- named months (`jan`-`dec`) and weekdays (`sun`-`sat`, or Quartz's
  1-indexed `SUN`-`SAT`)
- `@daily`, `@hourly`, `@reboot` and the other shorthand schedules
- malformed or backwards ranges (`5-1`)
- step values that are zero, negative, non-numeric, or larger than the
  field's range
- the redundant `*/1` form
- duplicate values in a comma list
- lines with too few fields, or a schedule keyword with no command
- day-of-month and day-of-week both being restricted at once, which is
  a common source of confusion: most cron implementations treat that
  combination as OR rather than AND, and Quartz requires one of the
  two to be `?` for exactly this reason

Whether a line is standard cron or Quartz is inferred from how many
leading tokens look like schedule fields rather than the start of a
command - a token containing `.` or more than one `/` is treated as
the command. This is a heuristic: a command with no extension and no
path separator can fool it.

Lines starting with `#` are treated as comments, and lines that look
like `NAME=value` are treated as environment variable assignments, both
of which are skipped.

## What it doesn't check yet

There's no validation of the command field itself (path existence,
quoting, etc.), and no support for the `L`, `W`, and `#`
day-of-month/day-of-week extensions some schedulers add.

## License

MIT, see LICENSE.
