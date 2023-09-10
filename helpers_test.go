package edx12

import (
	"os"
	"strings"
	"testing"
)

// failOnErr is a helper function that takes the result of a function that
// only has 1 return value (error), and fails the test if the error is not nil.
// It's intended to reduce boilerplate code in tests.
func failOnErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("%v", err)
	}
}

// replaceNewlines replaces `\r` and `\n` in the given text, so test assets
// can remain somewhat human-readable (one segment per line) without having
// the actual segment terminator set as a newline
func replaceNewlines(t *testing.T, text []byte) string {
	t.Helper()
	var replacer = strings.NewReplacer(
		"\r\n", "",
		"\r", "",
		"\n", "",
	)
	withoutNewlines := replacer.Replace(string(text))
	return withoutNewlines
}

func assertEqual[V comparable](t *testing.T, val V, expected V) {
	t.Helper()
	if val != expected {
		t.Errorf("expected:\n%#v\n\ngot:\n%#v", expected, val)
	}
}

// x835Message test fixture data is from:
// https://x12.org/examples/005010x221/example-02-multiple-claims-single-check
// (with slight modification, as the example had an error in a DTM segment)
func x835Message(t *testing.T) []byte {
	t.Helper()
	file, err := os.ReadFile("testdata/835.txt")
	assertNoError(t, err)
	return file
}

// x271Message test fixture data is from:
// https://x12.org/examples/005010x279/example-1b-response-generic-request-clinic-patients-subscriber-eligibility
func x271Message(t *testing.T) []byte {
	t.Helper()
	file, err := os.ReadFile("testdata/271.txt")
	assertNoError(t, err)
	return file
}

// newSegment creates a new segment from a string, without a spec attached.
// It will split the string using the given elementSeparator, and further
// split any elements that contain the repetitionSeparator into
// RepeatElementNode nodes, and CompositeNode for elements
// containing the componentElementSeparator. All others will be
// added as ElementNode nodes.
func newSegment(
	t *testing.T,
	s string,
	elementSeparator string,
	repetitionSeparator string,
	componentElementSeparator string,
) *X12Node {
	t.Helper()
	elements := strings.Split(s, elementSeparator)
	segmentId := elements[0]
	seg, e := NewNode(SegmentNode, segmentId)
	assertNoError(t, e)

	for _, v := range elements[1:] {
		if repetitionSeparator != "" && strings.Contains(
			v,
			repetitionSeparator,
		) {
			subElems := strings.Split(v, repetitionSeparator)
			repNode, e := NewNode(RepeatElementNode, "", subElems...)
			if e != nil {
				t.Fatalf("%v", e)
			}

			e = seg.Append(repNode)
			if e != nil {
				t.Fatalf("%v", e)
			}
		} else if componentElementSeparator != "" && strings.Contains(
			v,
			componentElementSeparator,
		) {
			subElems := strings.Split(v, componentElementSeparator)
			cmpNode, e := NewNode(CompositeNode, "")
			if e != nil {
				t.Fatalf("%v", e)
			}
			for elemInd := 0; elemInd < len(subElems); elemInd++ {
				subNode, err := NewNode(ElementNode, "", subElems[elemInd])
				if err != nil {
					t.Fatalf("%v", err)
				}
				err = cmpNode.Append(subNode)
				if err != nil {
					t.Fatalf("append error: %v", err)
				}
			}
			e = seg.Append(cmpNode)
			if e != nil {
				t.Fatalf("append error: %v", e)
			}
		} else {
			elemNode, e := NewNode(ElementNode, "", v)
			if e != nil {
				t.Fatalf("%v", e)
			}
			e = seg.Append(elemNode)
			if e != nil {
				t.Fatalf("%v", e)
			}
		}
	}
	return seg
}

func getSpec(
	t *testing.T,
	transactionSetCode string,
	transactionSetVersion string,
) *X12TransactionSetSpec {
	t.Helper()
	spec, err := findTransactionSpec(transactionSetCode, transactionSetVersion)
	assertNoError(t, err)
	if spec == nil {
		t.Fatalf("Expected a spec, got nil")
	}
	if spec.TransactionSetCode != transactionSetCode {
		t.Fatalf(
			"Expected transaction set code to be %v, got %v",
			transactionSetCode,
			spec.TransactionSetCode,
		)
	}

	return spec
}

func x271Spec(t *testing.T) *X12TransactionSetSpec {
	t.Helper()
	return getSpec(t, "271", "005010X279A1")
}

// x271MessageUnmatchedSegment is the same as x271Message, but is missing
// the required NM1 (subscriber) name segment from loop 2000C
func x271MessageUnmatchedSegment(t *testing.T) []byte {
	t.Helper()
	file, err := os.ReadFile("testdata/271_unmatched_segment.txt")
	if err != nil {
		t.Fatalf(
			"unable to open file %s",
			"testdata/271_unmatched_segment.txt",
		)
	}
	return file
}

// x270MessageMismatchedControlNumbers is the same as x270Message, but
// with all mismatched envelope control numbers, including:
// - ST02/SE02
// - GS05/GE02
// - ISA13/IEA02
func x270MessageMismatchedControlNumbers(t *testing.T) []byte {
	t.Helper()
	filename := "testdata/270_mismatched_control_numbers.txt"
	file, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("unable to open file %s", filename)
	}
	return file
}

func assertErrorNotNil(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func assertNotNil(t *testing.T, val interface{}) {
	t.Helper()
	if val == nil {
		t.Fatalf("expected non-nil value, got nil")
	}
}

func assertSliceContains[V comparable](t *testing.T, row []V, expected V) {
	t.Helper()
	if !sliceContains(row, expected) {
		t.Errorf("expected %v to be in slice %v", expected, row)
	}
}

func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// unmarshalText calls UnmarshalText and fails the test if there's an error,
// to reduce boilerplate.
func unmarshalText(t *testing.T, messageText []byte) (msg *Message) {
	t.Helper()
	msg = NewMessage()
	err := UnmarshalText(messageText, msg)
	assertNoError(t, err)
	return msg
}

// x270Message fixture contents are from:
// https://x12.org/examples/005010x279/example-1a-generic-request-clinic-patients-subscriber-eligibility
func x270Message(t *testing.T) []byte {
	t.Helper()
	file, err := os.ReadFile("testdata/270.txt")
	assertNoError(t, err)
	return file
}
