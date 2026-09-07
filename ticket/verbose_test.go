package ticket

import (
	"encoding/json"
	"reflect"
	"testing"
)

// fieldsMissing names the exported fields of from that to does not also carry
// under the same name. It is the whole mechanism behind the coverage test, and
// TestTheCoverageCheckBites proves it reports a field rather than sitting
// quietly at zero.
func fieldsMissing(from, to reflect.Type) []string {
	have := make(map[string]bool, to.NumField())
	for i := 0; i < to.NumField(); i++ {
		if f := to.Field(i); f.IsExported() {
			have[f.Name] = true
		}
	}
	var missing []string
	for i := 0; i < from.NumField(); i++ {
		f := from.Field(i)
		if f.IsExported() && !have[f.Name] {
			missing = append(missing, f.Name)
		}
	}
	return missing
}

// TestFindingVerboseCoversEveryFindingField is why FindingVerbose is allowed to
// exist. terva's complaint was that a hand-copied parallel struct goes stale
// when Finding gains a field, and a hand-written type on this side of the
// boundary would go stale identically. This is what makes that a build failure
// rather than a promise to remember.
func TestFindingVerboseCoversEveryFindingField(t *testing.T) {
	missing := fieldsMissing(reflect.TypeOf(Finding{}), reflect.TypeOf(FindingVerbose{}))
	if len(missing) > 0 {
		t.Errorf("FindingVerbose is missing %v; add them there and to Finding.Verbose", missing)
	}
}

// TestReportVerboseCoversEveryReportField holds the containing type too, so a
// third slice on Report cannot be dropped on the way out.
func TestReportVerboseCoversEveryReportField(t *testing.T) {
	missing := fieldsMissing(reflect.TypeOf(Report{}), reflect.TypeOf(ReportVerbose{}))
	if len(missing) > 0 {
		t.Errorf("ReportVerbose is missing %v", missing)
	}
}

// TestTheCoverageCheckBites demonstrates the failure the two tests above are
// for, without breaking the real types to do it. A Finding that grew a field is
// simulated here, and the check has to name it.
//
// A coverage assertion that cannot fail is worse than none, because it reads
// like protection.
func TestTheCoverageCheckBites(t *testing.T) {
	type findingThatGrewAField struct {
		Code     string
		File     string
		Ticket   string
		Field    string
		Message  string
		Title    string
		Severity string // the new one
	}
	missing := fieldsMissing(reflect.TypeOf(findingThatGrewAField{}), reflect.TypeOf(FindingVerbose{}))
	if len(missing) != 1 || missing[0] != "Severity" {
		t.Errorf("the coverage check reported %v, want exactly [Severity]", missing)
	}
}

// TestVerboseCarriesMessageAndTitle covers what terva asked for, and pins that
// asking for it did not disturb the four-key contract.
func TestVerboseCarriesMessageAndTitle(t *testing.T) {
	f := Finding{
		Code:    CodeTitleTooLong,
		File:    "tickets/TKT-1.md",
		Ticket:  "TKT-1",
		Field:   "title",
		Message: "the title is 130 characters, and the limit is 120",
		Title:   "A ticket with a very long title",
	}

	plain, err := json.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(plain, &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 4 {
		t.Errorf("Finding marshalled %d keys, want the recorded four: %s", len(got), plain)
	}
	if _, ok := got["message"]; ok {
		t.Error("the contract grew a message key")
	}

	verbose, err := json.Marshal(f.Verbose())
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(verbose, &got); err != nil {
		t.Fatal(err)
	}
	if got["message"] != f.Message {
		t.Errorf("verbose message = %v, want %q", got["message"], f.Message)
	}
	if got["title"] != f.Title {
		t.Errorf("verbose title = %v, want %q", got["title"], f.Title)
	}
	if got["code"] != f.Code || got["ticket"] != f.Ticket || got["field"] != f.Field {
		t.Errorf("verbose lost one of the four contract keys: %s", verbose)
	}
}

// TestVerboseKeepsAbsentFieldsNull pins that a finding about a file rather than
// a field still renders null, so a consumer reading both shapes does not have
// to tell missing from empty in one and not the other.
func TestVerboseKeepsAbsentFieldsNull(t *testing.T) {
	f := Finding{Code: CodeParseError, File: "tickets/broken.md", Message: "unreadable"}
	b, err := json.Marshal(f.Verbose())
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"ticket", "field"} {
		v, present := got[k]
		if !present {
			t.Errorf("%s was omitted; absent scalars are null, never missing", k)
		}
		if v != nil {
			t.Errorf("%s = %v, want null", k, v)
		}
	}
}

// TestReportVerbosePreservesNilSlices keeps the two report shapes renderable
// the same way: Report marshals a nil Warnings as null, so this must too.
func TestReportVerbosePreservesNilSlices(t *testing.T) {
	r := Report{Errors: []Finding{{Code: CodeParseError, File: "a.md", Message: "bad"}}}

	v := r.Verbose()
	if v.Warnings != nil {
		t.Errorf("nil warnings became %v", v.Warnings)
	}
	if len(v.Errors) != 1 || v.Errors[0].Message != "bad" {
		t.Fatalf("errors did not convert: %+v", v.Errors)
	}

	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got["warnings"] != nil {
		t.Errorf("warnings = %v, want null as Report renders it", got["warnings"])
	}
}
