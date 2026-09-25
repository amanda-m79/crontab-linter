package main

import (
	"strings"
	"testing"
)

func findingMessages(findings []Finding) []string {
	var msgs []string
	for _, f := range findings {
		msgs = append(msgs, f.Message)
	}
	return msgs
}

func containsMessage(findings []Finding, substr string) bool {
	for _, f := range findings {
		if strings.Contains(f.Message, substr) {
			return true
		}
	}
	return false
}

func TestLintFieldValid(t *testing.T) {
	cases := []string{"*", "5", "0-59", "*/5", "1,2,3", "9-17", "mon-fri", "jan-dec"}
	specs := map[string]fieldSpec{
		"*":       fields[0],
		"5":       fields[0],
		"0-59":    fields[0],
		"*/5":     fields[0],
		"1,2,3":   fields[0],
		"9-17":    fields[1],
		"mon-fri": fields[4],
		"jan-dec": fields[3],
	}
	for _, raw := range cases {
		spec := specs[raw]
		got := lintField(spec, raw, 1)
		if len(got) != 0 {
			t.Errorf("lintField(%q) on field %q = %v, want no findings", raw, spec.label, got)
		}
	}
}

func TestLintFieldEmptyValue(t *testing.T) {
	got := lintField(fields[0], "5,,10", 3)
	if !containsMessage(got, "empty value in list") {
		t.Errorf("expected empty value finding, got %v", findingMessages(got))
	}
}

func TestLintFieldDuplicateValue(t *testing.T) {
	got := lintField(fields[0], "5,10,5", 1)
	if !containsMessage(got, `duplicate value "5"`) {
		t.Errorf("expected duplicate value finding, got %v", findingMessages(got))
	}
}

func TestLintFieldStepNonPositive(t *testing.T) {
	for _, raw := range []string{"*/0", "*/-1", "*/foo"} {
		got := lintField(fields[0], raw, 1)
		if !containsMessage(got, "must be a positive integer") {
			t.Errorf("lintField(%q) = %v, want a positive-integer step finding", raw, findingMessages(got))
		}
	}
}

func TestLintFieldStepRedundantOnStar(t *testing.T) {
	got := lintField(fields[0], "*/1", 1)
	if !containsMessage(got, "redundant") {
		t.Errorf("expected redundant */1 finding, got %v", findingMessages(got))
	}
}

func TestLintFieldStepTooLarge(t *testing.T) {
	got := lintField(fields[0], "*/100", 1)
	if !containsMessage(got, "larger than the field's range") {
		t.Errorf("expected step-too-large finding, got %v", findingMessages(got))
	}
}

func TestLintFieldInvalidRange(t *testing.T) {
	got := lintField(fields[0], "a-b", 1)
	if !containsMessage(got, `invalid range "a-b"`) {
		t.Errorf("expected invalid range finding, got %v", findingMessages(got))
	}
}

func TestLintFieldRangeOutOfBounds(t *testing.T) {
	got := lintField(fields[0], "10-99", 1)
	if !containsMessage(got, "out of bounds") {
		t.Errorf("expected out-of-bounds range finding, got %v", findingMessages(got))
	}
}

func TestLintFieldRangeBackwards(t *testing.T) {
	got := lintField(fields[0], "10-5", 1)
	if !containsMessage(got, "start greater than end") {
		t.Errorf("expected backwards range finding, got %v", findingMessages(got))
	}
}

func TestLintFieldInvalidValue(t *testing.T) {
	got := lintField(fields[0], "notanumber", 1)
	if !containsMessage(got, `invalid value "notanumber"`) {
		t.Errorf("expected invalid value finding, got %v", findingMessages(got))
	}
}

func TestLintFieldValueOutOfRange(t *testing.T) {
	got := lintField(fields[0], "99", 1)
	if !containsMessage(got, "out of range 0-59") {
		t.Errorf("expected out-of-range finding, got %v", findingMessages(got))
	}
}

func TestLintFieldNamedValues(t *testing.T) {
	if got := lintField(fields[3], "Jan", 1); len(got) != 0 {
		t.Errorf("lintField(month, Jan) = %v, want no findings", findingMessages(got))
	}
	if got := lintField(fields[4], "sun", 1); len(got) != 0 {
		t.Errorf("lintField(day-of-week, sun) = %v, want no findings", findingMessages(got))
	}
	got := lintField(fields[3], "smarch", 1)
	if !containsMessage(got, `invalid value "smarch"`) {
		t.Errorf("expected invalid value finding for bad month name, got %v", findingMessages(got))
	}
}

func TestLintLineBlankAndComment(t *testing.T) {
	for _, line := range []string{"", "   ", "# a comment", "  # indented comment"} {
		if got := lintLine(line, 1); got != nil {
			t.Errorf("lintLine(%q) = %v, want nil", line, got)
		}
	}
}

func TestLintLineEnvVar(t *testing.T) {
	if got := lintLine("PATH=/usr/bin:/bin", 1); got != nil {
		t.Errorf("lintLine(env var) = %v, want nil", got)
	}
}

func TestLintLineNamedScheduleValid(t *testing.T) {
	if got := lintLine("@daily /usr/bin/backup.sh", 1); got != nil {
		t.Errorf("lintLine(@daily) = %v, want nil", got)
	}
}

func TestLintLineNamedScheduleUnknown(t *testing.T) {
	got := lintLine("@fortnightly /usr/bin/backup.sh", 1)
	if !containsMessage(got, "unknown schedule keyword") {
		t.Errorf("expected unknown schedule keyword finding, got %v", findingMessages(got))
	}
}

func TestLintLineNamedScheduleMissingCommand(t *testing.T) {
	got := lintLine("@daily", 1)
	if !containsMessage(got, "missing command after schedule keyword") {
		t.Errorf("expected missing command finding, got %v", findingMessages(got))
	}
}

func TestLintLineTooFewFields(t *testing.T) {
	got := lintLine("* * * * /usr/bin/cmd.sh", 1)
	if !containsMessage(got, "expected 5, 6, or 7 time fields plus a command, found 4 fields") {
		t.Errorf("expected too-few-fields finding, got %v", findingMessages(got))
	}
}

func TestLintLineMissingCommand(t *testing.T) {
	got := lintLine("* * * * *", 1)
	if !containsMessage(got, "missing command after time fields") {
		t.Errorf("expected missing-command finding, got %v", findingMessages(got))
	}
}

func TestLintLineQuartzSixFieldsValid(t *testing.T) {
	got := lintLine("0 0 12 * * ? /usr/bin/noon.sh", 1)
	if len(got) != 0 {
		t.Errorf("lintLine(quartz 6-field) = %v, want no findings", findingMessages(got))
	}
}

func TestLintLineQuartzSevenFieldsValid(t *testing.T) {
	got := lintLine("0 0 12 * * ? 2030 /usr/bin/noon.sh", 1)
	if len(got) != 0 {
		t.Errorf("lintLine(quartz 7-field) = %v, want no findings", findingMessages(got))
	}
}

func TestLintLineQuartzSeconds(t *testing.T) {
	got := lintLine("99 0 12 * * ? /usr/bin/noon.sh", 1)
	if !containsMessage(got, "value 99 out of range 0-59") {
		t.Errorf("expected seconds out-of-range finding, got %v", findingMessages(got))
	}
}

func TestLintLineQuartzDowNamesOneIndexed(t *testing.T) {
	got := lintLine("0 0 12 ? * SUN /usr/bin/noon.sh", 1)
	if len(got) != 0 {
		t.Errorf("lintLine(quartz SUN) = %v, want no findings", findingMessages(got))
	}
}

func TestLintLineQuartzDomDowBothRestricted(t *testing.T) {
	got := lintLine("0 0 12 15 * MON /usr/bin/noon.sh", 1)
	if !containsMessage(got, "Quartz requires one of them to be ?") {
		t.Errorf("expected quartz dom/dow ambiguity finding, got %v", findingMessages(got))
	}
}

func TestLintLineQuartzQuestionMarkOnlyOnDomDow(t *testing.T) {
	got := lintLine("0 0 ? * * ? /usr/bin/noon.sh", 1)
	if !containsMessage(got, `invalid value "?"`) {
		t.Errorf("expected invalid value finding for ? on hour field, got %v", findingMessages(got))
	}
}

func TestLintLineValid(t *testing.T) {
	got := lintLine("0 3 * * * /usr/bin/backup.sh", 1)
	if len(got) != 0 {
		t.Errorf("lintLine(valid) = %v, want no findings", findingMessages(got))
	}
}

func TestLintLineDomDowBothRestricted(t *testing.T) {
	got := lintLine("30 4 15 * 1 /usr/bin/ambiguous.sh", 1)
	if !containsMessage(got, "treat this as OR, not AND") {
		t.Errorf("expected dom/dow ambiguity finding, got %v", findingMessages(got))
	}
}

func TestLintLinePropagatesFieldFindings(t *testing.T) {
	got := lintLine("99 5 * * * /usr/bin/broken.sh", 7)
	if !containsMessage(got, "value 99 out of range 0-59") {
		t.Errorf("expected propagated field finding, got %v", findingMessages(got))
	}
	for _, f := range got {
		if f.Line != 7 {
			t.Errorf("finding %v has line %d, want 7", f, f.Line)
		}
	}
}
