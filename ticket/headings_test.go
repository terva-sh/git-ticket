package ticket

import (
	"strings"
	"testing"
)

// TestSectionHeadingsAgreesWithParseBody is the assertion the warning rests on.
// SectionHeadings predicts where parseBody will split, and a prediction that
// drifts from the parser is worse than no prediction: it would warn about text
// that is fine and stay quiet about text that is not.
//
// Every heading here is non-standard, so parseBody files them all in Extra in
// the order it found them. That makes Extra a direct answer to "what did the
// parser split on", rather than a restatement of the same rule.
func TestSectionHeadingsAgreesWithParseBody(t *testing.T) {
	body := strings.Join([]string{
		"Preamble text.",
		"",
		"## Alpha",
		"",
		"Text under alpha.",
		"",
		"```",
		"## FencedBacktick",
		"```",
		"",
		"~~~",
		"## FencedTilde",
		"~~~",
		"",
		"  ## Indented",
		"",
		"### NotASection",
		"",
		"## Beta",
		"",
		"Text under beta.",
	}, "\n")

	parsed := parseBody(body)
	var fromParser []string
	for _, s := range parsed.Extra {
		fromParser = append(fromParser, s.Heading)
	}

	predicted := SectionHeadings(body)

	if len(predicted) != len(fromParser) {
		t.Fatalf("predicted %v, parser split on %v", predicted, fromParser)
	}
	for i := range predicted {
		if predicted[i] != fromParser[i] {
			t.Errorf("heading %d: predicted %q, parser split on %q",
				i, predicted[i], fromParser[i])
		}
	}
	// Named rather than only compared, so a change that made both sides wrong
	// in the same way still fails.
	if want := []string{"Alpha", "Beta"}; strings.Join(predicted, ",") != strings.Join(want, ",") {
		t.Errorf("predicted %v, want %v", predicted, want)
	}
}

// TestSectionHeadingsFindsNothingInOrdinaryProse keeps the warning quiet on the
// text it will mostly see.
func TestSectionHeadingsFindsNothingInOrdinaryProse(t *testing.T) {
	for _, text := range []string{
		"",
		"One line.",
		"Prose with ## in the middle of a line.",
		"### A subheading is fine.",
		"#### So is a deeper one.",
		"##NoSpace is not a heading either.",
	} {
		if got := SectionHeadings(text); len(got) != 0 {
			t.Errorf("SectionHeadings(%q) = %v, want none", text, got)
		}
	}
}

// TestSectionHeadingsReportsEveryHeading covers the count the warning prints.
func TestSectionHeadingsReportsEveryHeading(t *testing.T) {
	got := SectionHeadings("Prose.\n\n## One\n\nx\n\n## Two\n\ny\n\n## Three\n\nz")
	if want := "One,Two,Three"; strings.Join(got, ",") != want {
		t.Errorf("got %v, want %s", got, want)
	}
}

// TestSectionHeadingsHandlesAnUnclosedFence pins what happens to text that
// opens a code block and never closes it. Everything after it is inside the
// block as far as the parser is concerned, so nothing below can be a heading,
// and the warning has to agree or it would report a section that will not exist.
func TestSectionHeadingsHandlesAnUnclosedFence(t *testing.T) {
	body := "Prose.\n\n```\n## Swallowed\n\n## AlsoSwallowed"

	if got := SectionHeadings(body); len(got) != 0 {
		t.Errorf("SectionHeadings = %v, want none inside an unclosed fence", got)
	}
	if got := parseBody(body); len(got.Extra) != 0 {
		t.Errorf("parseBody found %d sections, so the two disagree", len(got.Extra))
	}
}
