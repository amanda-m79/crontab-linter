package main

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// Finding is one problem found on a specific line of a crontab file.
type Finding struct {
	Line    int
	Field   string
	Message string
}

func (f Finding) String() string {
	if f.Field != "" {
		return fmt.Sprintf("line %d: %s: %s", f.Line, f.Field, f.Message)
	}
	return fmt.Sprintf("line %d: %s", f.Line, f.Message)
}

type fieldSpec struct {
	label string
	min   int
	max   int
	names map[string]int
}

var monthNames = map[string]int{
	"jan": 1, "feb": 2, "mar": 3, "apr": 4, "may": 5, "jun": 6,
	"jul": 7, "aug": 8, "sep": 9, "oct": 10, "nov": 11, "dec": 12,
}

var dowNames = map[string]int{
	"sun": 0, "mon": 1, "tue": 2, "wed": 3, "thu": 4, "fri": 5, "sat": 6,
}

var fields = []fieldSpec{
	{"minute", 0, 59, nil},
	{"hour", 0, 23, nil},
	{"day-of-month", 1, 31, nil},
	{"month", 1, 12, monthNames},
	{"day-of-week", 0, 7, dowNames},
}

var namedSchedules = map[string]bool{
	"@yearly": true, "@annually": true, "@monthly": true, "@weekly": true,
	"@daily": true, "@midnight": true, "@hourly": true, "@reboot": true,
}

// envVarRe matches crontab lines that set an environment variable
// (e.g. PATH=/usr/bin) rather than schedule a job.
var envVarRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)

func resolveValue(spec fieldSpec, token string) (int, bool) {
	if n, err := strconv.Atoi(token); err == nil {
		return n, true
	}
	if spec.names != nil {
		if n, ok := spec.names[strings.ToLower(token)]; ok {
			return n, true
		}
	}
	return 0, false
}

// lintField checks a single cron field (e.g. "1-5,*/2") against its spec.
func lintField(spec fieldSpec, raw string, lineNum int) []Finding {
	var out []Finding
	seen := map[string]bool{}

	for _, part := range strings.Split(raw, ",") {
		if part == "" {
			out = append(out, Finding{lineNum, spec.label, "empty value in list"})
			continue
		}
		if seen[part] {
			out = append(out, Finding{lineNum, spec.label, fmt.Sprintf("duplicate value %q", part)})
		}
		seen[part] = true

		body, step := part, ""
		if idx := strings.Index(part, "/"); idx >= 0 {
			body, step = part[:idx], part[idx+1:]
		}

		if step != "" {
			n, err := strconv.Atoi(step)
			switch {
			case err != nil || n <= 0:
				out = append(out, Finding{lineNum, spec.label, fmt.Sprintf("step %q must be a positive integer", step)})
			case body == "*" && n == 1:
				out = append(out, Finding{lineNum, spec.label, "step /1 on * is redundant, same as *"})
			case n > spec.max-spec.min+1:
				out = append(out, Finding{lineNum, spec.label, fmt.Sprintf("step %d is larger than the field's range", n)})
			}
		}

		if body == "*" {
			continue
		}

		if strings.Contains(body, "-") {
			bits := strings.SplitN(body, "-", 2)
			start, okStart := resolveValue(spec, bits[0])
			end, okEnd := resolveValue(spec, bits[1])
			if !okStart || !okEnd {
				out = append(out, Finding{lineNum, spec.label, fmt.Sprintf("invalid range %q", body)})
				continue
			}
			if start < spec.min || start > spec.max || end < spec.min || end > spec.max {
				out = append(out, Finding{lineNum, spec.label, fmt.Sprintf("range %q out of bounds %d-%d", body, spec.min, spec.max)})
			}
			if start > end {
				out = append(out, Finding{lineNum, spec.label, fmt.Sprintf("range %q has start greater than end", body)})
			}
			continue
		}

		val, ok := resolveValue(spec, body)
		if !ok {
			out = append(out, Finding{lineNum, spec.label, fmt.Sprintf("invalid value %q", body)})
			continue
		}
		if val < spec.min || val > spec.max {
			out = append(out, Finding{lineNum, spec.label, fmt.Sprintf("value %d out of range %d-%d", val, spec.min, spec.max)})
		}
	}

	return out
}

// lintLine checks one line of a crontab file. It returns nil for blank
// lines, comments, and environment variable assignments.
func lintLine(line string, lineNum int) []Finding {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return nil
	}
	if envVarRe.MatchString(trimmed) {
		return nil
	}

	tokens := strings.Fields(trimmed)
	if len(tokens) == 0 {
		return nil
	}

	if strings.HasPrefix(tokens[0], "@") {
		name := strings.ToLower(tokens[0])
		if !namedSchedules[name] {
			return []Finding{{lineNum, "", fmt.Sprintf("unknown schedule keyword %q", tokens[0])}}
		}
		if len(tokens) < 2 {
			return []Finding{{lineNum, "", "missing command after schedule keyword"}}
		}
		return nil
	}

	if len(tokens) < 6 {
		return []Finding{{lineNum, "", fmt.Sprintf("expected 5 time fields plus a command, found %d fields", len(tokens))}}
	}

	var findings []Finding
	for i, spec := range fields {
		findings = append(findings, lintField(spec, tokens[i], lineNum)...)
	}

	domRestricted := tokens[2] != "*"
	dowRestricted := tokens[4] != "*"
	if domRestricted && dowRestricted {
		findings = append(findings, Finding{lineNum, "",
			"day-of-month and day-of-week are both restricted; most cron daemons treat this as OR, not AND"})
	}

	return findings
}

// LintReader reads a crontab-style file line by line and returns every
// finding, in the order the lines appear.
func LintReader(r io.Reader) []Finding {
	scanner := bufio.NewScanner(r)
	var all []Finding
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		all = append(all, lintLine(scanner.Text(), lineNum)...)
	}
	return all
}
