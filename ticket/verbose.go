package ticket

// FindingVerbose is a Finding carrying everything it knows, including the two
// fields the recorded contract leaves out.
//
// Finding's own JSON is exactly code, file, ticket, and field, and it stays
// that way. Plan 10 records those four, every fixture sidecar in testdata/ is
// written against them, and a fifth key would rewrite the corpus. That contract
// serves a consumer comparing a report against a recorded expectation, which is
// what the corpus does.
//
// It does not serve a consumer showing the report to somebody. A person reading
// a terminal, or a language model reading a tool result, wants the sentence
// that explains the code and the title of the ticket it names. terva reported
// keeping a parallel struct and copying every finding into it to get them,
// which is a shadow type that goes stale the moment Finding gains a field.
//
// So this is the second serialization, and a caller opts into it. Nothing
// produces it otherwise and the corpus never sees it.
//
// TestFindingVerboseCoversEveryFindingField holds this type to Finding by
// reflection, so a field added to one and not the other fails the suite. That
// is the whole point: moving the shadow type across the boundary would only
// relocate the drift terva was complaining about.
type FindingVerbose struct {
	Code string `json:"code"`
	File string `json:"file"`
	// Ticket and Field stay nullable, matching Finding's contract, so a
	// consumer reading both shapes never has to tell missing from empty.
	Ticket *string `json:"ticket"`
	Field  *string `json:"field"`
	// Message is the human sentence. Title is the title of the ticket the
	// finding names, and is empty when the file did not parse far enough to
	// have one.
	Message string `json:"message"`
	Title   string `json:"title"`
}

// Verbose returns the finding with its message and title carried into the JSON.
func (f Finding) Verbose() FindingVerbose {
	v := FindingVerbose{
		Code:    f.Code,
		File:    f.File,
		Message: f.Message,
		Title:   f.Title,
	}
	if f.Ticket != "" {
		v.Ticket = &f.Ticket
	}
	if f.Field != "" {
		v.Field = &f.Field
	}
	return v
}

// ReportVerbose is a Report whose findings carry their message and title.
type ReportVerbose struct {
	Errors   []FindingVerbose `json:"errors"`
	Warnings []FindingVerbose `json:"warnings"`
}

// Verbose converts every finding in the report.
//
// A nil slice stays nil rather than becoming an empty one, so the verbose
// report renders null exactly where Report does. A consumer that handles one
// shape handles the other.
func (r Report) Verbose() ReportVerbose {
	return ReportVerbose{
		Errors:   verboseFindings(r.Errors),
		Warnings: verboseFindings(r.Warnings),
	}
}

func verboseFindings(in []Finding) []FindingVerbose {
	if in == nil {
		return nil
	}
	out := make([]FindingVerbose, len(in))
	for i, f := range in {
		out[i] = f.Verbose()
	}
	return out
}
