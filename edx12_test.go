package edx12

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestGroupSegmentsBySegmentId(t *testing.T) {
	elementSep := "*"
	repetitionSep := "^"
	componentSep := ":"

	segments := []*X12Node{
		newSegment(
			t,
			"ISA*FOO*BAR",
			elementSep,
			repetitionSep,
			componentSep,
		),
		newSegment(
			t,
			"GS*FOO*BAR",
			elementSep,
			repetitionSep,
			componentSep,
		),
		newSegment(
			t,
			"ST*270*0001",
			elementSep,
			repetitionSep,
			componentSep,
		),
		newSegment(
			t,
			"BHT*0019*00*10001234*20060501*1319",
			elementSep,
			repetitionSep,
			componentSep,
		),
		newSegment(
			t,
			"HL*1**20*1",
			elementSep,
			repetitionSep,
			componentSep,
		),
		newSegment(
			t,
			"NM1*PR*2*UNIFIED INSURANCE CO*****PI*842610001",
			elementSep,
			repetitionSep,
			componentSep,
		),
		newSegment(
			t,
			"SE*3*0001",
			elementSep,
			repetitionSep,
			componentSep,
		),
		newSegment(
			t,
			"ST*270*0002",
			elementSep,
			repetitionSep,
			componentSep,
		),
		newSegment(
			t,
			"BHT*0019*00*10001234*20060501*1319",
			elementSep,
			repetitionSep,
			componentSep,
		),
		newSegment(
			t,
			"HL*1**20*1",
			elementSep,
			repetitionSep,
			componentSep,
		),
		newSegment(
			t,
			"SE*2*0002",
			elementSep,
			repetitionSep,
			componentSep,
		),
		newSegment(
			t,
			"GE*FOO*BAR",
			elementSep,
			repetitionSep,
			componentSep,
		),
		newSegment(
			t,
			"IEA*FOO*BAR",
			elementSep,
			repetitionSep,
			componentSep,
		),
	}

	groupedSegments, err := groupNodesByName(
		"ST",
		"SE",
		segments,
	)
	assertNoError(t, err)
	if len(groupedSegments) != 2 {
		t.Fatalf("Expected 2 groups, got %d", len(groupedSegments))
	}
	if len(groupedSegments[0]) != 5 {
		t.Fatalf(
			"Expected 5 segments in first group, got %d: %v",
			len(groupedSegments[0]),
			groupedSegments,
		)
	}
	if len(groupedSegments[1]) != 4 {
		t.Fatalf(
			"Expected 4 segments in second group, got %d: %v",
			len(groupedSegments[1]),
			groupedSegments,
		)
	}
}

func TestNewGrouping(t *testing.T) {
	segments := []RawSegment{
		{"ISA", "FOO", "BAR"},
		{"GS", "FOO", "BAR"},
		{"ST", "270", "0001"},
		{"BHT", "0019", "00", "10001234", "20060501", "1319"},
		{"HL", "1", "", "20", "1"},
		{
			"NM1",
			"PR",
			"2",
			"UNIFIED INSURANCE CO",
			"",
			"",
			"",
			"",
			"PI",
			"842610001",
		},
		{"SE", "3", "0001"},
		{"ST", "270", "0002"},
		{"BHT", "0019", "00", "10001234", "20060501", "1319"},
		{"HL", "1", "", "20", "1"},
		{
			"NM1",
			"IL",
			"2",
			"UNIFIED INSURANCE CO",
			"",
			"",
			"",
			"",
			"PI",
			"842610001",
		},
		{"SE", "3", "0002"},
		{"GE", "FOO", "BAR"},
		{"IEA", "FOO", "BAR"},
	}

	groups, err := groupByFirstIndexValue(segments, "ST", "SE")
	for _, g := range groups {
		for _, s := range g {
			t.Logf("segments: %#v", s)
		}
		t.Logf("---")
	}
	assertNoError(t, err)
	assertEqual(t, len(groups), 2)

	expectedFirstGroup := segments[2:7]
	firstGroup := groups[0]
	assertEqual(t, len(firstGroup), len(expectedFirstGroup))
	assertEqual(t, firstGroup[0][2], "0001")
	expectedSecondGroup := segments[7:12]

	secondGroup := groups[1]
	assertEqual(t, len(secondGroup), len(expectedSecondGroup))
	assertEqual(t, secondGroup[0][2], "0002")
}

func TestGroupSegmentsMultiTrailer(t *testing.T) {
	elementSep := "*"
	repetitionSep := "^"
	componentSep := ":"
	segments := []*X12Node{
		newSegment(
			t,
			"ISA*FOO*BAR",
			elementSep,
			repetitionSep,
			componentSep,
		),
		newSegment(
			t,
			"GS*FOO*BAR",
			elementSep,
			repetitionSep,
			componentSep,
		),
		newSegment(
			t,
			"ST*270*0001",
			elementSep,
			repetitionSep,
			componentSep,
		),
		newSegment(
			t,
			"BHT*0019*00*10001234*20060501*1319",
			elementSep,
			repetitionSep,
			componentSep,
		),
		newSegment(
			t,
			"HL*1**20*1",
			elementSep,
			repetitionSep,
			componentSep,
		),
		newSegment(
			t,
			"NM1*PR*2*UNIFIED INSURANCE CO*****PI*842610001",
			elementSep,
			repetitionSep,
			componentSep,
		),
		newSegment(
			t,
			"SE*3*0001",
			elementSep,
			repetitionSep,
			componentSep,
		),
		newSegment(
			t,
			"SE*270*0002",
			elementSep,
			repetitionSep,
			componentSep,
		),
		newSegment(
			t,
			"BHT*0019*00*10001234*20060501*1319",
			elementSep,
			repetitionSep,
			componentSep,
		),
		newSegment(
			t,
			"HL*1**20*1",
			elementSep,
			repetitionSep,
			componentSep,
		),
		newSegment(
			t,
			"SE*2*0002",
			elementSep,
			repetitionSep,
			componentSep,
		),
		newSegment(
			t,
			"GE*FOO*BAR",
			elementSep,
			repetitionSep,
			componentSep,
		),
		newSegment(
			t,
			"IEA*FOO*BAR",
			elementSep,
			repetitionSep,
			componentSep,
		),
	}

	_, err := groupNodesByName(
		"ST",
		"SE",
		segments,
	)
	if err == nil {
		t.Fatalf("Expected error grouping segments, got none")
	}
	ne := &NodeError{}
	if !errors.As(err, &ne) {
		t.Fatalf("Expected NodeError, got %T", err)
	}
	if ne.Node.Name != "SE" {
		t.Errorf("Expected NodeError to have name SE, got %s", ne.Node.Name)
	}
}

func TestGroupSegmentsMultiHeader(t *testing.T) {
	elementSep := "*"
	repetitionSep := "^"
	componentSep := ":"
	segments := []*X12Node{
		newSegment(
			t,
			"ISA*FOO*BAR",
			elementSep,
			repetitionSep,
			componentSep,
		),
		newSegment(
			t,
			"GS*FOO*BAR",
			elementSep,
			repetitionSep,
			componentSep,
		),
		newSegment(
			t,
			"ST*270*0001",
			elementSep,
			repetitionSep,
			componentSep,
		),
		newSegment(
			t,
			"BHT*0019*00*10001234*20060501*1319",
			elementSep,
			repetitionSep,
			componentSep,
		),
		newSegment(
			t,
			"ST*1**20*1",
			elementSep,
			repetitionSep,
			componentSep,
		),
		newSegment(
			t,
			"NM1*PR*2*UNIFIED INSURANCE CO*****PI*842610001",
			elementSep,
			repetitionSep,
			componentSep,
		),
		newSegment(
			t,
			"SE*3*0001",
			elementSep,
			repetitionSep,
			componentSep,
		),
		newSegment(
			t,
			"ST*270*0002",
			elementSep,
			repetitionSep,
			componentSep,
		),
		newSegment(
			t,
			"BHT*0019*00*10001234*20060501*1319",
			elementSep,
			repetitionSep,
			componentSep,
		),
		newSegment(
			t,
			"HL*1**20*1",
			elementSep,
			repetitionSep,
			componentSep,
		),
		newSegment(
			t,
			"SE*2*0002",
			elementSep,
			repetitionSep,
			componentSep,
		),
		newSegment(
			t,
			"GE*FOO*BAR",
			elementSep,
			repetitionSep,
			componentSep,
		),
		newSegment(
			t,
			"IEA*FOO*BAR",
			elementSep,
			repetitionSep,
			componentSep,
		),
	}

	_, err := groupNodesByName(
		"ST",
		"SE",
		segments,
	)
	if err == nil {
		t.Fatalf("Expected error grouping segments, got none")
	}
	ne := &NodeError{}
	if !errors.As(err, &ne) {
		t.Fatalf("Expected NodeError, got %T", err)
	}
	if ne.Node.Name != "ST" {
		t.Errorf("Expected NodeError to have name ST, got %s", ne.Node.Name)
	}
}

func TestFindSegments(t *testing.T) {
	type testCase struct {
		Name        string
		ExpectCount int
	}
	testCases := []testCase{
		{
			Name:        stSegmentId,
			ExpectCount: 1,
		},
		{
			Name:        seSegmentId,
			ExpectCount: 1,
		},
		{
			Name:        gsSegmentId,
			ExpectCount: 1,
		},
		{
			Name:        geSegmentId,
			ExpectCount: 1,
		},
		{
			Name:        isaSegmentId,
			ExpectCount: 1,
		},
		{
			Name:        ieaSegmentId,
			ExpectCount: 1,
		},
		{
			Name:        hlSegmentId,
			ExpectCount: 3,
		},
		{
			Name:        "BHT",
			ExpectCount: 1,
		},
		{
			Name:        "NM1",
			ExpectCount: 4,
		},
		{
			Name:        "AEIOU",
			ExpectCount: 0,
		},
	}
	messageText := x271Message(t)
	msg := unmarshalText(t, messageText)
	for _, tc := range testCases {
		t.Run(
			tc.Name,
			func(t *testing.T) {
				segments := msg.SegmentsWithName(tc.Name)
				if len(segments) != tc.ExpectCount {
					t.Fatalf(
						"Expected %d segments, got %d",
						tc.ExpectCount,
						len(segments),
					)
				}
				for _, segment := range segments {
					if segment.Name != tc.Name {
						t.Errorf(
							"Expected segment name %s, got %s",
							tc.Name,
							segment.Name,
						)
					}
				}
			},
		)
	}
}

func TestElementValueValidation(t *testing.T) {
	elemSpec := &X12Spec{
		Name:       "Nm101",
		Usage:      Required,
		Label:      "someName",
		Type:       ElementSpec,
		ValidCodes: []string{"1", "2"},
	}
	var err error
	err = validateElementValue(elemSpec, "3")
	if err == nil {
		t.Errorf("expected error for validCodes, got nil")
	}
	err = validateElementValue(elemSpec, "2")
	assertNoError(t, err)

	elemSpec.ValidCodes = []string{}
	elemSpec.MinLength = 3
	elemSpec.MaxLength = 5
	err = validateElementValue(elemSpec, "aa")
	if err == nil {
		t.Errorf("expected error for minlength, got nil")
	}

	err = validateElementValue(elemSpec, "aaaaaaa")
	if err == nil {
		t.Errorf("expected error for max, got nil")
	}

	err = validateElementValue(elemSpec, "abcd")
	assertNoError(t, err)
}

func TestNewError(t *testing.T) {
	node, err := NewNode(ElementNode, "NM1")
	assertNoError(t, err)
	e := newNodeError(node, ErrElementHasInvalidValue)
	if e == nil {
		t.Fatal("Expected error, got nil")
	}
	if !errors.Is(e, ErrElementHasInvalidValue) {
		t.Errorf(
			"Expected error to be %T, got %T",
			ErrElementHasInvalidValue,
			e,
		)
	}
}

func TestElementNodeWithDefaultValue(t *testing.T) {
	var defaultVal *X12Node
	var expectedDefault string
	var err error

	for _, setRepeatable := range []bool{false, true} {

		elementSpec := &X12Spec{
			Type:      ElementSpec,
			Name:      "NM101",
			Label:     "idQualifier",
			Usage:     Required,
			RepeatMin: 1,
			RepeatMax: 1,
		}

		if setRepeatable {
			elementSpec.RepeatMax = 2
		}

		// validCodes with a single entry should be used as the default
		expectedDefault = "EH"
		elementSpec.ValidCodes = append(elementSpec.ValidCodes, expectedDefault)
		expectedVal := []string{expectedDefault}
		defaultVal, err = defaultElementObj(elementSpec)
		assertNoError(t, err)
		if !reflect.DeepEqual(defaultVal.Value, expectedVal) {
			t.Errorf("Expected %v, got %v", expectedVal, defaultVal.Value)
		}

		// defaultVal should take precedence over validCodes
		expectedDefault = "XX"
		elementSpec.DefaultVal = expectedDefault
		expectedVal[0] = expectedDefault
		defaultVal, err = defaultElementObj(elementSpec)
		assertNoError(t, err)
		if !reflect.DeepEqual(defaultVal.Value, expectedVal) {
			t.Errorf("Expected %v, got %v", expectedVal, defaultVal.Value)
		}

		// No value should be populated if validCodes has more than one entry
		// and defaultVal isn't set
		elementSpec.DefaultVal = ""
		elementSpec.ValidCodes = append(elementSpec.ValidCodes, "1P")

		assertEqual(t, len(elementSpec.ValidCodes), 2)
		if setRepeatable {
			expectedVal = make(
				[]string,
				elementSpec.RepeatMin,
				elementSpec.RepeatMax,
			)
		} else {
			expectedVal = []string{""}
		}

		defaultVal, err = defaultElementObj(elementSpec)
		assertNoError(t, err)

		if !reflect.DeepEqual(defaultVal.Value, expectedVal) {
			t.Errorf(
				"Expected %#v, got: %#v (%v/%v)",
				expectedVal,
				defaultVal.Value,
				elementSpec.IsRepeatable(),
				setRepeatable,
			)
		}

		// Empty validCodes and defaultVal should still have no default value
		elementSpec.ValidCodes = []string{}
		elementSpec.DefaultVal = ""
		defaultVal, err = defaultElementObj(elementSpec)
		assertNoError(t, err)
		if !reflect.DeepEqual(defaultVal.Value, expectedVal) {
			t.Errorf(
				"Expected %#v, got: %#v (%v/%v)",
				expectedVal,
				defaultVal.Value,
				elementSpec.IsRepeatable(),
				setRepeatable,
			)
		}
	}
}

func TestSetMessageHeaderValues(t *testing.T) {
	msgText := x270Message(t)
	msg := unmarshalText(t, msgText)
	type testCase struct {
		Index    int
		Value    string
		Expected string
	}
	testCases := []testCase{
		{
			Index:    isaIndexAuthInfoQualifier,
			Value:    "00",
			Expected: "00",
		},
		{
			Index:    isaIndexAuthInfo,
			Value:    "Foo",
			Expected: "       Foo",
		},
		{
			Index:    isaIndexSecurityInfoQualifier,
			Value:    "01",
			Expected: "01",
		},
		{
			Index:    isaIndexSecurityInfo,
			Value:    "Foo",
			Expected: "       Foo",
		},
		{
			Index:    isaIndexSenderIdQualifier,
			Value:    "01",
			Expected: "01",
		},
		{
			Index:    isaIndexSenderId,
			Value:    "Sender",
			Expected: "         Sender",
		},
		{
			Index:    isaIndexReceiverIdQualifier,
			Value:    "01",
			Expected: "01",
		},
		{
			Index:    isaIndexReceiverId,
			Value:    "Receiver",
			Expected: "       Receiver",
		},
		{
			Index:    isaIndexControlNumber,
			Value:    "11",
			Expected: "000000011",
		},
		{
			Index:    isaIndexControlNumber,
			Value:    "",
			Expected: "000000000",
		},
		{
			Index:    isaIndexAckRequested,
			Value:    "1",
			Expected: "1",
		},
		{
			Index:    isaIndexAckRequested,
			Value:    "0",
			Expected: "0",
		},
		{
			Index:    isaIndexUsageIndicator,
			Value:    "P",
			Expected: "P",
		},
		{
			Index:    isaIndexUsageIndicator,
			Value:    "T",
			Expected: "T",
		},
		{
			Index:    isaIndexComponentElementSeparator,
			Value:    ">",
			Expected: ">",
		},
		{
			Index:    isaIndexRepetitionSeparator,
			Value:    "<",
			Expected: "<",
		},
	}

	for _, tc := range testCases {
		t.Run(
			fmt.Sprintf("ISA%d", tc.Index),
			func(t *testing.T) {
				switch tc.Index {
				case isaIndexAuthInfoQualifier:
					failOnErr(t, msg.SetAuthInfoQualifier(tc.Value))
				case isaIndexAuthInfo:
					failOnErr(t, msg.SetAuthInfo(tc.Value))
				case isaIndexSecurityInfoQualifier:
					failOnErr(t, msg.SetSecurityInfoQualifier(tc.Value))
				case isaIndexSecurityInfo:
					failOnErr(t, msg.SetSecurityInfo(tc.Value))
				case isaIndexSenderIdQualifier:
					failOnErr(t, msg.SetSenderIdQualifier(tc.Value))
				case isaIndexSenderId:
					failOnErr(t, msg.SetSenderId(tc.Value))
				case isaIndexReceiverIdQualifier:
					failOnErr(t, msg.SetReceiverIdQualifier(tc.Value))
				case isaIndexReceiverId:
					failOnErr(t, msg.SetReceiverId(tc.Value))
				case isaIndexControlNumber:
					failOnErr(t, msg.SetControlNumber(tc.Value))
				case isaIndexAckRequested:
					failOnErr(t, msg.SetAckRequested(tc.Value))
				case isaIndexUsageIndicator:
					failOnErr(t, msg.SetUsageIndicator(tc.Value))
				case isaIndexComponentElementSeparator:
					failOnErr(
						t,
						msg.SetComponentElementSeparator(rune(tc.Value[0])),
					)
				case isaIndexRepetitionSeparator:
					failOnErr(t, msg.SetRepetitionSeparator(rune(tc.Value[0])))
				default:
					t.Fatalf("Unexpected index %d- add to switch", tc.Index)
				}

				val := msg.header().Children[tc.Index].Value[0]
				if val != tc.Expected {
					t.Errorf(
						"Expected ISA%02d to be '%s', got '%s'",
						tc.Index,
						tc.Expected,
						val,
					)
				}
			},
		)
	}
}

func TestFormatMessage(t *testing.T) {
	messageText := x270Message(t)
	rawMessage, err := Read(messageText)
	assertNoError(t, err)
	msg, err := rawMessage.Message(context.Background())
	assertNoError(t, err)
	//msg := unmarshalText(t, messageText)
	msgWithoutNewlines := replaceNewlines(t, messageText)
	msgFormat := msg.Format(
		string(rawMessage.SegmentTerminator),
		string(rawMessage.ElementSeparator),
		string(rawMessage.RepetitionSeparator),
		string(rawMessage.ComponentSeparator),
	)
	fmt.Println("Message: ", msgFormat)
	if msgFormat != msgWithoutNewlines {
		t.Errorf(
			"Expected message format to match original message:\nOriginal:  %s\nFormatted: %s\n",
			msgWithoutNewlines,
			msgFormat,
		)
	}
}

func TestCompositeDefault(t *testing.T) {
	segSpec := &X12Spec{Type: SegmentSpec, Name: "NM1"}

	nm101 := &X12Spec{
		ValidCodes: []string{"EH"}, Type: ElementSpec,
		Name:  "NM101",
		Label: "idQualifier",
	}
	appendSeg(t, segSpec, nm101)

	nm102 := &X12Spec{
		Name:  "NM102",
		Type:  ElementSpec,
		Usage: Required,
		Label: "lastName",
	}

	appendSeg(t, segSpec, nm102)

	cSpec := &X12Spec{
		Type:        CompositeSpec,
		Name:        "C040",
		Usage:       Required,
		Label:       "myComposite",
		Description: "some description",
	}

	// Should not return a default, and should return an error
	// if not provided a value
	hi01 := &X12Spec{
		Type:       ElementSpec,
		Name:       "HI01-01",
		Usage:      Required,
		MinLength:  1,
		MaxLength:  3,
		Label:      "qualifierCode",
		ValidCodes: []string{"ABK", "BK"},
	}

	//hi01.dataType = identifierSymbol
	failOnErr(t, appendSpec(cSpec, hi01))
	// Should not return a default, should not return an error if not provided
	hi02 := &X12Spec{Type: ElementSpec}
	hi02.Label = "industryCode"
	hi02.Name = "HI01-02"
	hi02.Usage = NotUsed
	//hi02.dataType = stringSymbol
	failOnErr(t, appendSpec(cSpec, hi02))
	// Should return a default, should not return an error if not explicitly
	// provided, but should return an error if provided with a zero value
	hi03 := &X12Spec{DefaultVal: "hospital"}
	hi03.Label = "facilityType"
	hi03.Name = "HI01-03"
	hi03.Usage = Required
	hi03.Type = ElementSpec
	hi03.MinLength = 1
	hi03.MaxLength = 15
	//hi03.dataType = stringSymbol
	failOnErr(t, appendSpec(cSpec, hi03))
	// Should not return a default, should not error if not provided
	hi04 := &X12Spec{Type: ElementSpec}
	hi04.Label = "facilityName"
	hi04.Name = "HI01-04"
	hi04.Usage = Situational
	hi04.MinLength = 1
	hi04.MaxLength = 35
	//hi04.dataType = stringSymbol
	failOnErr(t, appendSpec(cSpec, hi04))
	// Should not return a default, should return an error if not provided
	hi05 := &X12Spec{Type: ElementSpec}
	hi05.Label = "facilityState"
	hi05.Name = "HI01-05"
	hi05.Usage = Required
	hi05.MinLength = 2
	hi05.MaxLength = 2
	//hi05.dataType = stringSymbol
	hi05.ValidCodes = []string{"GA"}
	failOnErr(t, appendSpec(cSpec, hi05))

	appendSeg(t, segSpec, cSpec)

	generatedDefault, _ := cSpec.defaultObjValue()
	generatedValues := []string{}
	for _, elem := range generatedDefault.Children {
		elemVal := elem.Value
		if len(elemVal) == 0 {
			generatedValues = append(generatedValues, "")
		} else {
			generatedValues = append(generatedValues, elemVal[0])
		}
	}

	expectedValues := []string{"", "", "hospital", "", "GA"}
	if !reflect.DeepEqual(generatedValues, expectedValues) {
		t.Fatalf("Expected: %v, got: %v", expectedValues, generatedValues)
	}

	generatedSegDefault, err := segSpec.defaultObjValue()
	assertNoError(t, err)
	generatedSegPayload, err := payloadFromSegment(generatedSegDefault)
	if err != nil {
		var nodeErr *NodeError
		if errors.As(err, &nodeErr) {
			t.Fatalf(
				"expected no error, got %s with value %#v and spec: %+v",
				nodeErr.Error(),
				nodeErr.Node.Value,
				nodeErr.Node.spec,
			)
		}
	}
	assertNoError(t, err)
	fmt.Printf("payload:\n%v\n", generatedSegPayload)

}

func TestValidateRepeat(t *testing.T) {
	type testCase struct {
		RepeatMin int
		RepeatMax int
		IsValid   bool
	}

	testCases := []testCase{
		{
			RepeatMin: -1,
			IsValid:   false,
		},
		{
			RepeatMin: 0,
			IsValid:   true,
		},
		{
			RepeatMin: 1,
			IsValid:   true,
		},
		{
			RepeatMax: -1,
			IsValid:   false,
		},
		{
			RepeatMax: 0,
			IsValid:   true,
		},
		{
			RepeatMax: 1,
			IsValid:   true,
		},
		{
			RepeatMin: 2,
			RepeatMax: 1,
			IsValid:   false,
		},
		{
			RepeatMin: 2,
			RepeatMax: 0,
			IsValid:   true,
		},
	}

	for i, tc := range testCases {
		t.Run(
			fmt.Sprintf("case %d (%#v)", i, tc),
			func(t *testing.T) {
				s := &X12Spec{
					RepeatMin: tc.RepeatMin,
					RepeatMax: tc.RepeatMax,
				}
				if tc.IsValid == true {
					assertNoError(t, s.validateRepeat())
				} else {
					assertErrorNotNil(t, s.validateRepeat())
				}

			},
		)
	}
}

func TestValidateSpecChecks(t *testing.T) {
	var err error
	var se *SpecError
	// validate that only a SegmentSpec can have segment conditions
	s := &X12Spec{
		Usage: Required,
		Name:  "foo",
		Type:  LoopSpec,
		Syntax: []SegmentCondition{
			{
				Indexes:       []int{1, 2},
				ConditionCode: Exclusion,
			},
		},
	}
	err = s.validateSpec()
	assertErrorNotNil(t, err)
	if !errors.As(err, &se) {
		t.Fatalf("expected SpecError, got %T", err)
	}
	assertStringContains(
		t,
		err.Error(),
		"transaction set/loop spec must have children",
	)

	s.Syntax = nil

	// validate that only a defined spec type can be used (while
	// UnknownSpec is a valid type, it is not a valid spec type)
	s.Type = SpecType(123)
	err = s.validateSpec()
	assertErrorNotNil(t, err)
	if !errors.As(err, &se) {
		t.Fatalf("expected SpecError, got %T", err)
	}
	assertStringContains(
		t,
		err.Error(),
		"[name: foo]: type is required",
	)
	s.Type = SegmentSpec

	// validate that a name must be provided
	s.Name = ""
	s.path = "/foo"
	err = s.validateSpec()
	assertErrorNotNil(t, err)
	if !errors.As(err, &se) {
		t.Fatalf("expected SpecError, got %T", err)
	}
	assertStringContains(
		t,
		err.Error(),
		"[path: /foo]: name is required",
	)

	s.Name = "foo"

	// validate element specs can't have subspecs
	s.Type = ElementSpec
	s.Structure = append(
		s.Structure,
		&X12Spec{Type: ElementSpec, Name: "baz", Usage: NotUsed},
	)
	err = s.validateSpec()
	assertErrorNotNil(t, err)
	if !errors.As(err, &se) {
		t.Fatalf("expected SpecError, got %T", err)
	}
	assertStringContains(
		t,
		err.Error(),
		"[name: foo path: /foo]: element spec cannot have children",
	)

	s.Structure = []*X12Spec{}
	s.Type = SegmentSpec
}

func TestValidateUsage(t *testing.T) {
	s := &X12Spec{
		Usage:       Usage(99),
		Name:        "foo",
		Type:        LoopSpec,
		path:        "/foo",
		Description: "bar",
	}
	assertErrorNotNil(t, s.validateUsage())

	s.Usage = UnknownUsage
	err := s.validateUsage()
	assertErrorNotNil(t, err)

	err = s.validateSpec()
	var se *SpecError
	if !errors.As(err, &se) {
		t.Fatalf("expected SpecError, got %T", err)
	}
	expectedErr := "usage must be one of: REQUIRED, SITUATIONAL, NOT USED"
	//assertEqual(t, err.Error(), expectedErr)
	assertStringContains(t, err.Error(), expectedErr)
}

func TestValidateMinMaxLength(t *testing.T) {
	type testCase struct {
		MinLength int
		MaxLength int
		IsValid   bool
	}

	testCases := []testCase{
		{
			MinLength: -1,
			IsValid:   false,
		},
		{
			MaxLength: -1,
			IsValid:   false,
		},
		{
			MinLength: 10,
			MaxLength: 9,
			IsValid:   false,
		},
		{
			MinLength: 10,
			MaxLength: 50,
			IsValid:   true,
		},
	}

	for i, tc := range testCases {
		t.Run(
			fmt.Sprintf("case %d (%#v)", i, tc),
			func(t *testing.T) {
				s := &X12Spec{
					MinLength: tc.MinLength,
					MaxLength: tc.MaxLength,
					Type:      ElementSpec,
					Name:      "foo",
					Usage:     Required,
				}
				if tc.IsValid == true {
					assertNoError(t, s.validateMinMaxLength())
				} else {
					err := s.validateMinMaxLength()
					assertErrorNotNil(t, err)

					specErr := s.validateSpec()
					assertErrorNotNil(t, specErr)
					var se *SpecError
					if !errors.As(specErr, &se) {
						t.Errorf("expected SpecError, got %T", err)
					}
				}
			},
		)
	}

}

func TestValidateValidCodes(t *testing.T) {
	type testCase struct {
		ValidCodes []string
		DefaultVal string
		MinLength  int
		MaxLength  int
		IsValid    bool
	}

	testCases := []testCase{
		{
			ValidCodes: []string{"GA", "FL"},
			DefaultVal: "MD",
			IsValid:    false,
		},
		{
			ValidCodes: []string{"GA", "FL"},
			DefaultVal: "GA",
			IsValid:    true,
		},
		{
			ValidCodes: []string{"GA", "FL"},
			IsValid:    true,
		},
		{
			ValidCodes: []string{"GA", "FL"},
			MinLength:  3,
			IsValid:    false,
		},
		{
			ValidCodes: []string{"GA", "FL"},
			MaxLength:  1,
			IsValid:    false,
		},
	}

	for i, tc := range testCases {
		t.Run(
			fmt.Sprintf("case %d (%#v)", i, tc),
			func(t *testing.T) {
				s := &X12Spec{
					ValidCodes: tc.ValidCodes,
					DefaultVal: tc.DefaultVal,
					MinLength:  tc.MinLength,
					MaxLength:  tc.MaxLength,
					Type:       ElementSpec,
				}
				if tc.IsValid == true {
					assertNoError(t, s.validateValidCodes())
				} else {
					assertErrorNotNil(t, s.validateValidCodes())
				}
			},
		)
	}

}

func TestSetIndexValue(t *testing.T) {
	elemNode, err := NewNode(ElementNode, "NM1")
	assertNoError(t, err)

	_, err = elemNode.setIndexValue(1, "test")
	assertErrorNotNil(t, err)
	expectedErr := "[type: 'Element' name: 'NM1']: index 1 out of bounds"
	if err.Error() != expectedErr {
		t.Fatalf("Expected error: %s, got: %s", expectedErr, err)
	}
	_, seterr := elemNode.setIndexValue(0, "test2")
	if seterr != nil {
		t.Fatalf("Error setting index value: %s", err)
	}
	assertEqual(t, elemNode.Value[0], "test2")

	elemNode.Type = RepeatElementNode
	_, repErr := elemNode.setIndexValue(5, "test3")
	if repErr != nil {
		t.Fatalf("Error setting index value: %s", err)
	}
	values := elemNode.Value
	expectedValues := []string{"test2", "", "", "", "", "test3"}
	if !reflect.DeepEqual(values, expectedValues) {
		t.Fatalf("Expected values: %v, got: %v", expectedValues, values)
	}

	segNode, err := NewNode(SegmentNode, "NM1")
	assertNoError(t, err)
	_, err = segNode.setIndexValue(2, "test")
	assertErrorNotNil(t, err)

	expectedError := "[type: 'Segment' name: 'NM1']: no spec set"
	if err.Error() != expectedError {
		t.Fatalf(
			"Expected error: %s, got: %s",
			expectedError,
			err,
		)
	}
	var nodeErr *NodeError
	if !errors.As(err, &nodeErr) {
		t.Fatalf("Expected error to be NodeError, got: %T", err)
	}
}

func TestSegmentNodeReplace(t *testing.T) {
	segNode, e := NewNode(SegmentNode, "NM1")
	assertNoError(t, e)

	elemNode, e := NewNode(ElementNode, "NM101")
	assertNoError(t, e)
	failOnErr(t, elemNode.SetValue("1"))
	failOnErr(t, segNode.append(elemNode))

	elemNodeB, e := NewNode(ElementNode, "NM102")
	assertNoError(t, e)
	failOnErr(t, elemNodeB.SetValue("2"))
	failOnErr(t, segNode.append(elemNodeB))

	elemNodeC, e := NewNode(ElementNode, "NM103")
	assertNoError(t, e)
	failOnErr(t, elemNodeC.SetValue("3"))
	failOnErr(t, segNode.append(elemNodeC))

	elemNodeD, e := NewNode(ElementNode, "")
	assertNoError(t, e)
	failOnErr(t, elemNodeD.SetValue("F"))

	err := segNode.replace(2, elemNodeD)
	assertNoError(t, err)
	if segNode.Children[2] != elemNodeD {
		t.Fatalf("Expected node to be replaced")
	}
	for _, err := range segNode.Children {
		if err == elemNodeB {
			t.Fatalf("Expected node to be removed")

		}
	}
}

func TestSpecValidation(t *testing.T) {
	cSpec := &X12Spec{Type: CompositeSpec}
	cSpec.Name = "C040"
	cSpec.Usage = Required
	cSpec.Description = "some description"
	cSpec.Label = "myComposite"

	// Should not return a default, and should return an error
	// if not provided a value
	hi01 := &X12Spec{ValidCodes: []string{"ABK", "BK"}}
	hi01.Label = "qualifierCode"
	hi01.Name = "HI01-01"
	hi01.Usage = Required
	hi01.Type = ElementSpec
	hi01.MinLength = 1
	hi01.MaxLength = 3
	//hi01.dataType = identifierSymbol
	assertNoError(t, appendSpec(cSpec, hi01))

	// Should not return a default, should not return an error if not provided
	hi02 := &X12Spec{}
	hi02.Label = "industryCode"
	hi02.Name = "HI01-02"
	hi02.Usage = NotUsed
	hi02.Type = ElementSpec
	//hi02.dataType = stringSymbol
	assertNoError(t, appendSpec(cSpec, hi02))

	// Should return a default, should not return an error if not explicitly
	// provided, but should return an error if provided with a zero value
	hi03 := &X12Spec{DefaultVal: "hospital"}
	hi03.Label = "facilityType"
	hi03.Name = "HI01-03"
	hi03.Usage = Required
	hi03.Type = ElementSpec
	hi03.MinLength = 1
	hi03.MaxLength = 15
	//hi03.dataType = stringSymbol
	assertNoError(t, appendSpec(cSpec, hi03))

	// Should not return a default, should not error if not provided
	hi04 := &X12Spec{}
	hi04.Label = "facilityName"
	hi04.Name = "HI01-04"
	hi04.Usage = Situational
	hi04.Type = ElementSpec
	hi04.MinLength = 1
	hi04.MaxLength = 35
	//hi04.dataType = stringSymbol
	assertNoError(t, appendSpec(cSpec, hi04))

	// Should not return a default, should return an error if not provided
	hi05 := &X12Spec{}
	hi05.Label = "facilityState"
	hi05.Name = "HI01-05"
	hi05.Usage = Required
	hi05.Type = ElementSpec
	hi05.MinLength = 2
	hi05.MaxLength = 2
	//hi05.dataType = stringSymbol
	assertNoError(t, appendSpec(cSpec, hi05))

	// The default value should remove trailing empty elements, so in this
	// case it should stop at HI03, which is the last element spec with
	// a default value specified
	expectedDefault, err := NewNode(CompositeNode, "", "", "hospital")
	assertNoError(t, err)

	assertNoError(t, expectedDefault.SetSpec(cSpec))

	defaultValue, e := cSpec.defaultObjValue()
	assertNoError(t, e)
	if len(defaultValue.Children) != len(expectedDefault.Children) {
		t.Errorf(
			"Expected default value to be %v, got %v",
			len(expectedDefault.Children),
			len(defaultValue.Children),
		)
	}
	if !reflect.DeepEqual(defaultValue.Value, expectedDefault.Value) {
		t.Errorf(
			"Expected: %#v\ngot: %#v",
			expectedDefault.Value,
			defaultValue.Value,
		)
	}
}

//
//func TestMissingTxnChildren(t *testing.T) {
//	// The given node should either have zero or at least two
//	// children, and neither the first nor last child should be
//	// nil. If this is not the case, an error should be returned.
//	node := &X12Node{Children: []*X12Node{nil}}
//	_, err := createTransactionNode(node)
//	assertErrorNotNil(t, err)
//
//	var ne *NodeError
//	if !errors.As(err, &ne) {
//		t.Errorf("Expected NodeError, got %v", err)
//	}
//
//	node.Children[0] = &X12Node{}
//	node.Children = append(node.Children, nil)
//	_, err = createTransactionNode(node)
//	assertErrorNotNil(t, err)
//
//	var sne *NodeError
//	if !errors.As(err, &sne) {
//		t.Errorf("Expected NodeError, got %v", err)
//	}
//
//	node.Children = []*X12Node{&X12Node{}}
//	_, err = createTransactionNode(node)
//	assertErrorNotNil(t, err)
//	var eee *NodeError
//	if !errors.As(err, &eee) {
//		t.Errorf("Expected NodeError, got %v", err)
//	}
//}

func TestDefaultLoopObj(t *testing.T) {
	spec := X270v005010X279A1
	loopSpec, ok := spec.PathMap["/ST_LOOP/2000A"]
	if !ok {
		t.Fatal("Expected to find loop")
	}
	assertEqual(t, loopSpec.Type, LoopSpec)

	loop, err := loopSpec.defaultObjValue()
	assertNoError(t, err)
	assertNotNil(t, loop)
	assertEqual(t, loop.Name, loopSpec.Name)

	if len(loop.Children) == 0 {
		t.Fatal("Expected loop to have children")
	}
	hlSegment := loop.Children[0]
	assertNotNil(t, hlSegment.spec)
	assertEqual(t, hlSegment.Name, hlSegmentId)

	if len(hlSegment.Children) <= 1 {
		t.Fatal("Expected HL segment to have children")
	}
	randomLoop := &X12Node{Type: LoopNode, Name: "SOMELOOP"}
	replaceErr := hlSegment.replace(0, randomLoop)
	assertErrorNotNil(t, replaceErr)

	assertEqual(t, loop.Children[1].Name, "2100A")
}

func TestValidateElement(t *testing.T) {
	node := &X12Node{Type: LoopNode}
	err := validateElement(node)
	assertErrorNotNil(t, err)

	node.Type = ElementNode

	elementSpec := &X12Spec{
		Type:      ElementSpec,
		Usage:     Required,
		RepeatMin: 2,
		RepeatMax: 4,
	}
	assertNoError(t, node.SetSpec(elementSpec))
	node.Type = ElementNode

	assertErrorNotNil(t, validateElement(node))

	elementSpec.RepeatMin = 0
	elementSpec.RepeatMax = 1
	node.Type = RepeatElementNode

	assertErrorNotNil(t, validateElement(node))
}

func TestValidateSegment(t *testing.T) {
	node := &X12Node{Type: SegmentNode}
	node.Name = "NM1"
	node.Children = append(
		node.Children,
		&X12Node{Type: ElementNode, Value: []string{"ABC"}},
	)
	assertErrorNotNil(t, validateSegmentNode(node))

}

func TestParentLabels(t *testing.T) {
	rootNode := &X12Spec{
		Type:  LoopSpec,
		Name:  "LOOP1",
		Label: "foo",
	}
	middleNode := &X12Spec{
		Type:   LoopSpec,
		Name:   "LOOP2",
		Label:  "bar",
		parent: rootNode,
	}
	bottomNode := &X12Spec{
		Type:   SegmentSpec,
		Name:   "NM1",
		Label:  "baz",
		parent: middleNode,
	}
	var getParentLabels = func(x *X12Spec) []string {
		parentLabels := []string{}
		var currentNode *X12Spec
		currentNode = x.parent
		for currentNode != nil {
			parentLabels = append(parentLabels, currentNode.Label)
			currentNode = currentNode.parent
		}
		return parentLabels
	}
	labels := getParentLabels(bottomNode)
	//labels := bottomNode.parentLabels()
	if !sliceContains(labels, "bar") {
		t.Errorf("Expected %v, got %v", "bar", labels)
	}
	if !sliceContains(labels, "foo") {
		t.Errorf("Expected %v, got %v", "foo", labels)
	}
	assertEqual(t, len(labels), 2)
}

func TestSpecMinMaxLength(t *testing.T) {
	spec := &X12Spec{Type: LoopSpec, MinLength: 1, MaxLength: 2}
	spec.finalize()
	assertEqual(t, spec.MaxLength, 0)
	assertEqual(t, spec.MinLength, 0)
}

func TestNodeTypeJSON(t *testing.T) {
	nodeType := LoopNode
	jsonData, err := json.Marshal(nodeType)
	assertNoError(t, err)
	assertEqual(t, string(jsonData), `"Loop"`)

	var nt NodeType
	data := []byte(`"Loop"`)
	err = json.Unmarshal(data, &nt)
	assertNoError(t, err)
	assertEqual(t, nt, LoopNode)
}

func TestGetSpecs(t *testing.T) {
	assertEqual(t, len(_defaultReader.transactionSpecs), 3)
}

func TestPayloadFromX12Err(t *testing.T) {
	n := &X12Node{
		Type: SegmentNode,
		Name: "NM1",
		Children: []*X12Node{
			{
				Type:  ElementNode,
				Value: []string{"foo"},
			},
			{
				Type:  ElementNode,
				Value: []string{"bar"},
			},
			{
				Type:  ElementNode,
				Value: []string{"baz"},
			},
		},
	}
	_, err := payloadFromX12(n)
	assertErrorNotNil(t, err)
}

func TestReader(t *testing.T) {
	msgText := x270Message(t)
	msg := unmarshalText(t, msgText)
	t.Logf("msg: %#v", msg)

	for _, grp := range msg.functionalGroups {
		for _, txn := range grp.Transactions {
			txnSpec, err := findTransactionSpec(txn)
			assertNoError(t, err)
			e := txn.Transform(txnSpec)
			assertNoError(t, e)
			data, err := txn.JSON(true)
			assertNoError(t, err)
			t.Logf("result:\n%s", string(data))
		}
	}
	errs := msg.Validate()
	assertNoError(t, errs)
}

func TestNodeTypeStr(t *testing.T) {
	type testCase struct {
		Type     NodeType
		Expected string
	}
	var testCases = []testCase{
		{
			Type:     ElementNode,
			Expected: "Element",
		},
		{
			Type:     CompositeNode,
			Expected: "Composite",
		},
		{
			Type:     SegmentNode,
			Expected: "Segment",
		},
		{
			Type:     LoopNode,
			Expected: "Loop",
		},
		{
			Type:     TransactionNode,
			Expected: "TransactionSet",
		},
	}
	for i, tc := range testCases {
		t.Run(
			fmt.Sprintf("Test %d", i), func(t *testing.T) {
				ts := tc.Type.String()
				assertEqual(t, ts, tc.Expected)
			},
		)
	}
}

func appendSeg(t *testing.T, spec *X12Spec, n *X12Spec) {
	t.Helper()
	err := appendSpec(spec, n)
	assertNoError(t, err)
}

func TestSegmentSpecMatch(t *testing.T) {
	segSpec := &X12Spec{
		Type: SegmentSpec,
	}
	segSpec.Name = "NM1"
	segSpec.Usage = Required
	segSpec.Label = "some_name"

	nm101 := &X12Spec{ValidCodes: []string{"EH"}, Type: ElementSpec}
	nm101.Name = "NM101"
	nm101.Label = "idQualifier"
	appendSeg(t, segSpec, nm101)

	nm102 := &X12Spec{Type: ElementSpec}
	nm102.Name = "NM102"
	nm102.Usage = Required
	nm102.Label = "lastName"
	appendSeg(t, segSpec, nm102)

	nm103 := &X12Spec{Type: ElementSpec}
	nm103.Name = "NM103"
	nm103.Usage = Situational
	nm103.Label = "firstName"
	appendSeg(t, segSpec, nm103)

	seg := newSegment(t, "NM1*EH*FOO*BAR", "*", "", "")
	var isMatch bool
	isMatch, _ = segmentMatchesSpec(seg, segSpec)
	if !isMatch {
		elems := []string{}
		for _, element := range seg.elements() {
			elems = append(elems, element.Value[0])
		}
		t.Errorf(
			"Expected segment to match definition (elements: %#v)",
			elems,
		)
	}

	mismatchedSeg := newSegment(t, "NM1*ZZ*FOO*BAR", "*", "", "")

	isMatch, _ = segmentMatchesSpec(mismatchedSeg, segSpec)
	if isMatch {
		t.Errorf(
			"Expected segment to not match definition (elements: %v",
			mismatchedSeg.elements(),
		)
	}

	mismatchedId := newSegment(t, "REF*EH*FOO*BAR", "*", "", "")

	isMatch, _ = segmentMatchesSpec(mismatchedId, segSpec)
	if isMatch {
		t.Errorf(
			"Expected segment to not match definition (elements: %v",
			mismatchedId.elements(),
		)
	}

}

func TestAllowedSpecTypes(t *testing.T) {
	type testCase struct {
		Spec    *X12Spec
		Type    NodeType
		Allowed bool
	}
	var testCases = []testCase{
		{
			Spec:    &X12Spec{Type: LoopSpec},
			Type:    LoopNode,
			Allowed: true,
		},
		{
			Spec:    &X12Spec{Type: SegmentSpec},
			Type:    SegmentNode,
			Allowed: true,
		},
		{
			Spec:    &X12Spec{Type: LoopSpec},
			Type:    SegmentNode,
			Allowed: false,
		},
		{
			Spec:    &X12Spec{Type: SegmentSpec},
			Type:    LoopNode,
			Allowed: false,
		},
	}
	for i, tc := range testCases {
		t.Run(
			fmt.Sprintf("Test %d", i), func(t *testing.T) {
				types, ok := specNodeTypes[tc.Spec.Type]
				if !ok {
					t.Fatalf("No types for spec type: %s", tc.Spec.Type)
				}
				allowed := sliceContains(types, tc.Type)
				assertEqual(t, allowed, tc.Allowed)
			},
		)
	}
}

func TestValidate271(t *testing.T) {
	messageText := string(x271Message(t))
	oldDmg := "DMG*D8*19630519*M~"
	newDmg := "DMG*D8**M~"

	if !strings.Contains(messageText, oldDmg) {
		t.Fatalf("Expected old DMG segment")
	}
	messageText = strings.Replace(
		messageText,
		oldDmg,
		newDmg,
		1,
	)
	msgBytes := []byte(messageText)
	//msg := unmarshalText(t, msgBytes)
	rawMessage, err := Read(msgBytes)
	assertNoError(t, err)

	msg, err := rawMessage.Message(context.Background())
	assertNoError(t, err)

	err = msg.Validate()
	assertErrorNotNil(t, err)
	assertStringContains(
		t,
		err.Error(),
		"[type: 'Segment' name: 'DMG' path: '/X12/GS_LOOP[0]/ST_LOOP[0]/2000A/2000B/2000C/2100C/DMG' position: 13]: segment condition validation failed: when at least one value in [1 2] is present, all must be presen",
	)

}

func Test271UnmatchedSegment(t *testing.T) {
	messageText := x271MessageUnmatchedSegment(t)
	rawMessage, err := Read(messageText)
	txnSegments, e := rawMessage.TransactionSegments()
	assertNoError(t, e)
	assertEqual(t, len(txnSegments), 1)
	totalSegments := len(txnSegments[0])

	assertNoError(t, err)
	ctx := context.Background()
	msg, err := rawMessage.Message(ctx)
	if cause := ctx.Err(); cause != nil {
		t.Fatalf("error in context: %s: %s", cause.Error(), context.Cause(ctx))
	}
	txn := msg.TransactionSets()[0]
	segs := txn.Segments()

	assertEqual(t, len(segs), 9)

	if segs[7].Name != "TRN" {
		t.Errorf("Expected segment TRN, got %s", segs[8].Name)
	}
	if err == nil {
		t.Errorf("expected error")
	}
	unmatched := []*X12Node{}

	for _, n := range txn.segments {
		if !sliceContains(segs, n) {
			unmatched = append(unmatched, n)
		}
	}
	assertEqual(t, len(unmatched), totalSegments-len(segs))
}

func Test271Levels(t *testing.T) {
	messageText := x271Message(t)
	msg := unmarshalText(t, messageText)
	for _, txn := range msg.TransactionSets() {
		parentLevels, e := createHierarchicalLevels(txn.X12Node)
		assertNoError(t, e)
		t.Logf("levels: %v", parentLevels)
	}
}

func TestISATooShort(t *testing.T) {
	messageText := string(x270Message(t))
	messageText = strings.Replace(messageText, "*T*:~", "*:~", 1)
	_, err := Read([]byte(messageText))

	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if !errors.Is(err, ErrInvalidISA) {
		t.Fatalf("Expected ErrInvalidISA, got %v", err)
	}
	expectedError := "parse error: invalid ISA segment: ISA segment too short (expected 17 elements, got 16)"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error message '%s', got: '%s'", expectedError, err)
	}
}

func TestMismatchedControlNumbers(t *testing.T) {
	file := x271MessageMismatchedControlNumbers(t)
	rawMessage, err := Read(file)
	assertNoError(t, err)
	msg, err := rawMessage.Message(context.Background())
	assertNoError(t, err)

	err = msg.Validate()
	if !errors.Is(err, ErrInvalidTransactionEnvelope) {
		t.Fatalf(
			"Expected ErrInvalidTransactionEnvelope, got %v",
			err,
		)
	}
	//msg := unmarshalText(t, file)
	errs := msg.Validate()

	if !errors.Is(errs, ErrInvalidGroupEnvelope) {
		t.Fatalf(
			"Expected ErrInvalidGroupEnvelope, got %v",
			errs,
		)
	}
	if !errors.Is(errs, ErrInvalidInterchangeEnvelope) {
		t.Fatalf(
			"Expected ErrInvalidInterchangeEnvelope, got: %v",
			errs,
		)
	}
}

func TestIsRepeatable(t *testing.T) {
	type testCase struct {
		Usage     Usage
		RepeatMin int
		RepeatMax int
		Expected  bool
	}

	testCases := []testCase{
		{
			Usage:     Required,
			RepeatMin: 0,
			RepeatMax: 1,
			Expected:  false,
		},
		{
			Usage:     Required,
			RepeatMin: 1,
			RepeatMax: 1,
			Expected:  false,
		},
		{
			Usage:     Required,
			RepeatMin: 1,
			RepeatMax: 2,
			Expected:  true,
		},
		{
			Usage:     Required,
			RepeatMin: 1,
			RepeatMax: 0,
			Expected:  true,
		},
		{
			Usage:     Required,
			RepeatMin: 1,
			RepeatMax: 1,
			Expected:  false,
		},
		{
			Usage:     NotUsed,
			RepeatMin: 2,
			RepeatMax: 99,
			Expected:  false,
		},
		{
			Usage:     Situational,
			RepeatMin: 0,
			RepeatMax: 1,
			Expected:  false,
		},
		{
			Usage:     Situational,
			RepeatMin: 1,
			RepeatMax: 1,
			Expected:  false,
		},
		{
			Usage:     Situational,
			RepeatMin: 1,
			RepeatMax: 2,
			Expected:  true,
		},
		{
			Usage:     Situational,
			RepeatMin: 0,
			RepeatMax: 2,
			Expected:  true,
		},
		{
			Usage:     Situational,
			RepeatMin: 9,
			RepeatMax: 0,
			Expected:  true,
		},
		{
			Usage:     Situational,
			RepeatMin: 9,
			RepeatMax: 99,
			Expected:  true,
		},
		{
			Usage:     Situational,
			RepeatMin: 1,
			RepeatMax: 0,
			Expected:  true,
		},
		{
			Usage:     Situational,
			RepeatMin: 1,
			RepeatMax: 1,
			Expected:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(
			fmt.Sprintf("% +v", tc),
			func(t *testing.T) {
				s := &X12Spec{
					Usage:     tc.Usage,
					RepeatMin: tc.RepeatMin,
					RepeatMax: tc.RepeatMax,
				}
				assertEqual(t, s.IsRepeatable(), tc.Expected)
			},
		)
	}
}

func TestSegmentSyntaxConditional(t *testing.T) {
	segmentCondition := SegmentCondition{
		ConditionCode: Conditional,
		Indexes:       []int{1, 3, 5},
	}
	segSpec := &X12Spec{
		Type:   SegmentSpec,
		Name:   "NM1",
		Syntax: []SegmentCondition{segmentCondition},
	}

	type testCase struct {
		Elements      []string
		ErrorExpected bool
	}
	var testCases = []testCase{
		{Elements: []string{"", ""}, ErrorExpected: false},
		{
			Elements: []string{
				"NM1",
				"VAL",
				"",
				"asdf",
				"asdf",
				"asdf",
				"asdf",
			}, ErrorExpected: false,
		},
		{
			Elements:      []string{"NM1", "VAL", "", "", "", "", ""},
			ErrorExpected: true,
		},
		{
			Elements:      []string{"NM1", "VAL", "", "VAL", "", "", ""},
			ErrorExpected: true,
		},
		{
			Elements:      []string{"NM1", "", "", "", "", "VAL", ""},
			ErrorExpected: false,
		},
		{
			Elements:      []string{"NM1", "VAL", "", "", "", "VAL", ""},
			ErrorExpected: true,
		},
		{
			Elements: []string{
				"NM1",
				"",
				"VAL",
				"",
				"VAL",
				"",
				"VAL",
			},
			ErrorExpected: false,
		},
	}
	for i, tc := range testCases {
		t.Run(
			fmt.Sprintf(
				"%d elements: %v / expected: %v",
				i,
				tc.Elements,
				tc.ErrorExpected,
			),
			func(t *testing.T) {
				seg := &X12Node{Type: SegmentNode}
				assertNoError(t, seg.SetSpec(segSpec))
				//assertNoError(t, err)

				for ei := 1; ei < len(tc.Elements); ei++ {
					elemNode := &X12Node{
						Type: ElementNode,
						Name: fmt.Sprintf("NM10%d", ei),
					}
					assertNoError(t, elemNode.SetValue(tc.Elements[ei]))
					assertNoError(t, seg.append(elemNode))
				}
				err := validateSegmentSyntax(seg, &segmentCondition)
				if tc.ErrorExpected && err == nil {
					t.Errorf(
						"(Test case %d) Expected an error for segment %+v, got nil",
						i,
						seg,
					)
				} else if !tc.ErrorExpected && err != nil {
					t.Errorf(
						"(Test case %d) Expected no error for segment %+v, got: %+v",
						i,
						seg,
						err,
					)
				}
			},
		)
	}
}

func TestSegmentSyntaxListValidation(t *testing.T) {
	segmentCondition := SegmentCondition{
		ConditionCode: ListConditional,
		Indexes:       []int{1, 3, 5},
	}
	segSpec := &X12Spec{
		Name:   "NM1",
		Type:   SegmentSpec,
		Syntax: []SegmentCondition{segmentCondition},
	}

	type testCase struct {
		Elements      []string
		ErrorExpected bool
	}
	var testCases = []testCase{
		{
			Elements:      []string{"", ""},
			ErrorExpected: false,
		},
		{
			Elements: []string{
				"NM1",
				"VAL",
				"",
				"asdf",
				"asdf",
				"asdf",
				"asdf",
			},
			ErrorExpected: false,
		},
		{
			Elements:      []string{"NM1", "VAL", "", "", "", "", ""},
			ErrorExpected: true,
		},

		{
			Elements: []string{
				"NM1",
				"VAL",
				"",
				"VAL",
				"",
				"",
				"",
			},
			ErrorExpected: false,
		},
		{
			Elements: []string{
				"NM1",
				"",
				"",
				"",
				"",
				"VAL",
				"",
			},
			ErrorExpected: false,
		},
		{
			Elements: []string{
				"NM1",
				"VAL",
				"",
				"",
				"",
				"VAL",
				"",
			},
			ErrorExpected: false,
		},
		{
			Elements: []string{
				"NM1",
				"",
				"VAL",
				"",
				"VAL",
				"",
				"VAL",
			},
			ErrorExpected: false,
		},
	}

	for i, tc := range testCases {
		t.Run(
			fmt.Sprintf(
				"%d elements: %v / expected: %v",
				i,
				tc.Elements,
				tc.ErrorExpected,
			),
			func(t *testing.T) {
				seg := &X12Node{Type: SegmentNode}
				assertNoError(t, seg.SetSpec(segSpec))

				for ei := 1; ei < len(tc.Elements); ei++ {
					elemNode := &X12Node{
						Type: ElementNode,
						Name: fmt.Sprintf("NM10%d", ei),
					}
					assertNoError(t, elemNode.SetValue(tc.Elements[ei]))
					assertNoError(t, seg.append(elemNode))
				}
				err := validateSegmentSyntax(seg, &segmentCondition)
				if tc.ErrorExpected && err == nil {
					t.Errorf(
						"(Test case %d) Expected an error for segment %+v, got nil",
						i,
						seg,
					)
				} else if !tc.ErrorExpected && err != nil {
					t.Errorf(
						"(Test case %d) Expected no error for segment %+v, got: %+v",
						i,
						seg,
						err,
					)
				}
			},
		)
	}
}

func TestSegmentSyntaxPaired(t *testing.T) {
	segmentCondition := SegmentCondition{
		ConditionCode: PairedOrMultiple,
		Indexes:       []int{1, 3, 5},
	}
	segSpec := &X12Spec{
		Name:   "NM1",
		Type:   SegmentSpec,
		Syntax: []SegmentCondition{segmentCondition},
	}

	type testCase struct {
		Elements      []string
		ErrorExpected bool
	}
	var testCases = []testCase{
		{
			Elements:      []string{"", ""},
			ErrorExpected: false,
		},
		{
			Elements: []string{
				"NM1",
				"VAL",
				"",
				"asdf",
				"asdf",
				"asdf",
				"asdf",
			}, ErrorExpected: false,
		},
		{
			Elements:      []string{"NM1", "VAL", "", "", "", "", ""},
			ErrorExpected: true,
		},
	}
	for i, tc := range testCases {
		t.Run(
			fmt.Sprintf(
				"%d elements: %v / expected: %v",
				i,
				tc.Elements,
				tc.ErrorExpected,
			),
			func(t *testing.T) {
				seg := &X12Node{Type: SegmentNode}
				assertNoError(t, seg.SetSpec(segSpec))

				for ei := 1; ei < len(tc.Elements); ei++ {
					elemNode := &X12Node{
						Type: ElementNode,
						Name: fmt.Sprintf("NM10%d", ei),
					}
					assertNoError(t, elemNode.SetValue(tc.Elements[ei]))

					assertNoError(t, seg.append(elemNode))

				}

				err := validateSegmentSyntax(seg, &segmentCondition)
				if tc.ErrorExpected && err == nil {
					t.Errorf(
						"(Test case %d) Expected an error for segment %+v, got nil",
						i,
						seg,
					)
				} else if !tc.ErrorExpected && err != nil {
					t.Errorf(
						"(Test case %d) Expected no error for segment %+v, got: %+v",
						i,
						seg,
						err,
					)
				}
			},
		)

	}
}

func TestSegmentSyntaxRequired(t *testing.T) {
	segmentCondition := SegmentCondition{
		ConditionCode: RequiredCondition,
		Indexes:       []int{1, 3, 5},
	}
	segSpec := &X12Spec{
		Name:   "NM1",
		Type:   SegmentSpec,
		Syntax: []SegmentCondition{segmentCondition},
	}

	type testCase struct {
		Elements      []string
		ErrorExpected bool
	}
	var testCases = []testCase{
		{Elements: []string{"", ""}, ErrorExpected: true},
		{
			Elements: []string{
				"NM1",
				"VAL",
				"",
				"asdf",
				"asdf",
				"asdf",
				"asdf",
			}, ErrorExpected: false,
		},
		{
			Elements:      []string{"NM1", "VAL", "", "", "", "", ""},
			ErrorExpected: false,
		},
		{
			Elements:      []string{"NM1", "VAL", "", "VAL", "", "", ""},
			ErrorExpected: false,
		},
		{
			Elements:      []string{"NM1", "", "", "", "", "", ""},
			ErrorExpected: true,
		},
		{
			Elements: []string{
				"NM1",
				"",
				"VAL",
				"",
				"VAL",
				"",
				"VAL",
			},
			ErrorExpected: true,
		},
		{
			Elements:      []string{"NM1", "", "", "", "", "VAL", ""},
			ErrorExpected: false,
		},
		{
			Elements: []string{
				"NM1",
				"",
				"VAL",
				"",
				"VAL",
				"",
				"VAL",
			},
			ErrorExpected: true,
		},
	}
	for i, tc := range testCases {
		t.Run(
			fmt.Sprintf(
				"%d elements: %v / expected: %v",
				i,
				tc.Elements,
				tc.ErrorExpected,
			),
			func(t *testing.T) {
				seg := &X12Node{Type: SegmentNode}
				assertNoError(t, seg.SetSpec(segSpec))
				for ei := 1; ei < len(tc.Elements); ei++ {
					elemNode := &X12Node{
						Type: ElementNode,
						Name: fmt.Sprintf("NM10%d", ei),
					}
					assertNoError(t, elemNode.SetValue(tc.Elements[ei]))
					assertNoError(t, seg.append(elemNode))

				}

				err := validateSegmentSyntax(seg, &segmentCondition)
				if tc.ErrorExpected && err == nil {
					t.Errorf(
						"(Test case %d) Expected an error for segment %+v, got nil",
						i,
						seg,
					)
				} else if !tc.ErrorExpected && err != nil {
					t.Errorf(
						"(Test case %d) Expected no error for segment %+v, got: %+v",
						i,
						seg,
						err,
					)
				}
			},
		)
	}
}

func TestSegmentTooManyElements(t *testing.T) {
	segSpec := &X12Spec{Name: "NM1", Type: SegmentSpec}
	seg := &X12Node{Type: SegmentNode}
	assertNoError(t, seg.SetSpec(segSpec))
	segElements := seg.elements()
	if len(segElements) != 1 {
		t.Fatalf("Expected 1 element, got %d", len(segElements))
	}
	if len(segElements[0].Value) != 1 {
		t.Fatalf("Expected 1 value, got %d", len(segElements[0].Value))
	}
	if segElements[0].Value[0] != "NM1" {
		t.Fatalf("Expected value 'NM1', got %q", segElements[0].Value[0])
	}
	ilElem, err := NewNode(ElementNode, "IL")
	assertNoError(t, err)
	elemErr := seg.append(ilElem)
	if elemErr != nil {
		t.Fatalf("Error appending element: %v", elemErr)
	}

	validationErr := seg.Validate()
	if validationErr == nil {
		t.Fatalf("Expected error, got nil")
	}
	expectedErr := "[type: 'Segment' name: 'NM1']: spec defines 0 elements, segment has 1"
	assertStringContains(t, validationErr.Error(), expectedErr)
}

func TestReaderJSON(t *testing.T) {
	msgText := x270Message(t)
	msg := unmarshalText(t, msgText)
	t.Logf("msg: %#v", msg)
	for _, grp := range msg.functionalGroups {
		for _, txn := range grp.Transactions {
			txnSpec, err := findTransactionSpec(txn)
			assertNoError(t, err)
			e := txn.Transform(txnSpec)
			assertNoError(t, e)
			data, err := txn.JSON(true)
			assertNoError(t, err)
			t.Logf("result:\n%s", string(data))

			jsonData, err := json.MarshalIndent(txn, "", "  ")
			assertNoError(t, err)
			t.Logf("json result:\n%s", string(jsonData))

		}
	}
}

func TestPayloadFromX12(t *testing.T) {
	node, err := NewNode(SegmentNode, "NM1")
	assertNoError(t, err)

	// A new segment node should have the name added as the first element,
	// with a spec automatically specifying that as the default valid and
	// as the sole member of ValidCodes
	_, err = payloadFromX12(node)
	assertNoError(t, err)
}

func TestNewSegmentNode(t *testing.T) {
	node, err := NewNode(SegmentNode, "NM1", "NM1", "IL", "1")
	assertNoError(t, err)

	elements := node.elements()
	assertEqual(t, len(elements), 3)

	idValue := elements[0].Value[0]
	assertEqual(t, idValue, "NM1")

	idSpec := elements[0].spec
	assertNotNil(t, idSpec)

	assertEqual(t, idSpec.DefaultVal, "NM1")
	assertEqual(t, len(idSpec.ValidCodes), 1)
	assertSliceContains(t, idSpec.ValidCodes, "NM1")

	qualValue := elements[1].Value[0]
	assertEqual(t, qualValue, "IL")

	postValue := elements[2].Value[0]
	assertEqual(t, postValue, "1")
}

func TestAppendBadNode(t *testing.T) {
	n := &X12Node{Type: SegmentNode}
	childNode := &X12Node{Type: LoopNode}
	assertErrorNotNil(t, n.append(childNode))
}

func TestSetCompositeIndexValue(t *testing.T) {
	n, err := NewNode(CompositeNode, "CLM05")
	assertNoError(t, err)
	_, err = n.setIndexValue(0, "123")
	assertErrorNotNil(t, err)
}

func TestSetNodeValue(t *testing.T) {
	node := &X12Node{Type: LoopNode}
	assertErrorNotNil(t, node.SetValue("foo"))

	node.Type = ElementNode
	assertErrorNotNil(t, node.SetValue("foo", "bar"))

	node.Type = CompositeNode
	cmpSpec := &X12Spec{
		Type: CompositeSpec,
		Structure: []*X12Spec{
			{
				Type:       ElementSpec,
				DefaultVal: "foo",
				Usage:      Required,
			},
			{
				Type:       ElementSpec,
				DefaultVal: "bar",
				Usage:      Required,
			},
		},
	}
	node.spec = cmpSpec
	assertErrorNotNil(t, node.SetValue("foo", "bar", "baz"))

	assertNoError(t, node.SetValue("foo", "bar"))

}

func TestAppendCompositeNode(t *testing.T) {
	n, err := NewNode(SegmentNode, "NM1", "NM1", "IL", "1")
	assertNoError(t, err)
	childNode := &X12Node{Type: CompositeNode}
	assertNoError(t, n.append(childNode))
	assertEqual(t, childNode.Name, "NM103")
}

func TestReader835(t *testing.T) {
	msgText := x835Message(t)
	msg := unmarshalText(t, msgText)
	t.Logf("msg: %#v", msg)

	for _, grp := range msg.functionalGroups {
		for _, txn := range grp.Transactions {
			e := txn.Transform()
			assertNoError(t, e)
			data, err := txn.JSON(true)
			assertNoError(t, err)
			t.Logf("result:\n%s", string(data))
		}
	}
	errs := msg.Validate()
	assertNoError(t, errs)
}

func TestMessageHeader(t *testing.T) {
	msg := unmarshalText(t, x270Message(t))
	assertEqual(t, msg.AuthInfo(), "Authorizat")
	assertEqual(t, msg.SecurityInfoQualifier(), "00")
	assertEqual(t, msg.SecurityInfo(), "Security  ")
	assertEqual(t, msg.SenderIdQualifier(), "ZZ")
	assertEqual(t, msg.SenderId(), "Interchange    ")
	assertEqual(t, msg.ReceiverIdQualifier(), "ZZ")
	assertEqual(t, msg.ReceiverId(), "Interchange Rec")
	assertEqual(t, msg.Date(), "141001")
	assertEqual(t, msg.Time(), "1037")
	assertEqual(t, msg.Version(), "00501")
	assertEqual(t, msg.ControlNumber(), "000031033")
	assertEqual(t, msg.AckRequested(), "0")
	assertEqual(t, msg.UsageIndicator(), "T")
	assertEqual(t, msg.ComponentElementSeparator(), ':')
	assertEqual(t, msg.RepetitionSeparator(), '^')
}

//func TestMarshal271(t *testing.T) {
//	msgText := x271Message(t)
//	msg := Message()
//	err := UnmarshalText([]byte(msgText), msg)
//	assertNoError(t, err)
//	data, err := json.MarshalIndent(msg, "", "  ")
//	assertNoError(t, err)
//	t.Logf("result: %s", string(data))
//}

func TestSegmentTerminatorMatchesElementSeparator(t *testing.T) {
	messageText := x270Message(t)
	messageText[isaByteCount-1] = messageText[isaElementSeparatorIndex]
	//msg := Message()
	_, err := Read(messageText)

	//err := UnmarshalText(messageText, msg)
	assertErrorNotNil(t, err)
	if !errors.Is(err, ErrInvalidISA) {
		t.Fatalf("Expected ErrInvalidISA, got %v", err)
	}
	expectedError := "parse error: invalid ISA segment: segment terminator '*' cannot be the same as the element separator"
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got: '%s'", expectedError, err)
	}
}

func TestSetCompositeSpec(t *testing.T) {
	node, err := NewNode(ElementNode, "C022")
	assertNoError(t, err)
	node.SetValue("foo", "baz")
	spec := &X12Spec{
		Type: CompositeSpec,
		Structure: []*X12Spec{
			{
				Name:       "HI01-01",
				Type:       ElementSpec,
				Usage:      Required,
				DefaultVal: "foo",
				ValidCodes: []string{"foo", "bar", "baz"},
			},
			{
				Name:       "HI01-02",
				Type:       ElementSpec,
				DefaultVal: "bar",
				Usage:      Required,
				ValidCodes: []string{"foo", "bar", "baz"},
			},
		},
	}
	assertNoError(t, node.SetSpec(spec))
	assertEqual(t, node.spec, spec)
	assertEqual(t, len(node.Value), 0)
	assertEqual(t, len(node.Children), 2)
}

func TestSetBadSpecType(t *testing.T) {
	node, err := NewNode(ElementNode, "C022")
	assertNoError(t, err)
	someSpec := &X12Spec{Type: LoopSpec}
	err = node.SetSpec(someSpec)
	assertErrorNotNil(t, err)
	ne := &NodeError{}
	if !errors.As(err, &ne) {
		t.Fatalf("Expected NodeError, got %v", err)
	}
	expectedErr := "[type: 'Element']: spec 'Loop' not allowed for node type 'Element'"
	assertEqual(t, ne.Error(), expectedErr)
}

func TestConvertElement(t *testing.T) {
	converted, err := convertElement("foo", String)
	assertNoError(t, err)
	assertEqual(t, converted, "foo")

	converted, err = convertElement("1.25", Decimal)
	assertNoError(t, err)
	assertEqual(t, converted, 1.25)

	converted, err = convertElement("125", Numeric)
	assertNoError(t, err)
	assertEqual(t, converted, 125)

}

func TestParseDate(t *testing.T) {
	type testCase struct {
		Time       string
		Year       int
		Month      time.Month
		Day        int
		Nanosecond int
	}
	testCases := []testCase{
		{
			Time:  "231031",
			Year:  2023,
			Month: time.October,
			Day:   31,
		},
		{
			Time:  "20231031",
			Year:  2023,
			Month: time.October,
			Day:   31,
		},
	}

	for _, tc := range testCases {
		t.Run(
			fmt.Sprintf("% +v", tc),
			func(t *testing.T) {
				converted, err := parseDate(tc.Time)
				assertNoError(t, err)
				assertEqual(t, converted.IsZero(), false)
				assertEqual(t, converted.Year(), tc.Year)
				assertEqual(t, converted.Month(), tc.Month)
				assertEqual(t, converted.Day(), tc.Day)

			},
		)
	}
}

func TestParseTime(t *testing.T) {
	type testCase struct {
		Time       string
		Hour       int
		Minute     int
		Second     int
		Nanosecond int
	}
	testCases := []testCase{
		{
			Time:   "1030",
			Hour:   10,
			Minute: 30,
		},
		{
			Time:   "103021",
			Hour:   10,
			Minute: 30,
			Second: 21,
		},
		{
			Time:       "10302154",
			Hour:       10,
			Minute:     30,
			Second:     21,
			Nanosecond: 540000000,
		},
		{
			Time:       "1030215",
			Hour:       10,
			Minute:     30,
			Second:     21,
			Nanosecond: 500000000,
		},
	}

	for _, tc := range testCases {
		t.Run(
			fmt.Sprintf("% +v", tc),
			func(t *testing.T) {
				converted, err := parseTime(tc.Time)
				assertNoError(t, err)
				assertEqual(t, converted.IsZero(), false)
				assertEqual(t, converted.Hour(), tc.Hour)
				assertEqual(t, converted.Minute(), tc.Minute)
				assertEqual(t, converted.Second(), tc.Second)
				assertEqual(t, converted.Nanosecond(), tc.Nanosecond)

			},
		)
	}
}

func Test271JSON(t *testing.T) {
	msg := unmarshalText(t, x271Message(t))
	for _, txn := range msg.TransactionSets() {
		txnSpec, err := findTransactionSpec(txn)
		assertNoError(t, err)
		assertNoError(t, txn.Transform(txnSpec))
		data, err := json.MarshalIndent(txn, "", "  ")
		assertNoError(t, err)
		t.Logf("result:\n%s", string(data))
	}

}

func TestDataTypeToString(t *testing.T) {
	type testCase struct {
		dataType DataType
		Expected string
	}
	testCases := []testCase{
		{UnknownDataType, ""},
		{String, "AN"},
		{Numeric, "N"},
		{Decimal, "R"},
		{Date, "DT"},
		{Time, "TM"},
		{Binary, "B"},
		{Identifier, "ID"},
	}

	for _, tc := range testCases {
		t.Run(
			tc.dataType.String(),
			func(t *testing.T) {
				assertEqual(t, tc.dataType.String(), tc.Expected)
				data, err := tc.dataType.MarshalJSON()
				assertNoError(t, err)
				var rv DataType
				err = json.Unmarshal(data, &rv)
				assertNoError(t, err)
				assertEqual(t, tc.dataType, rv)

			},
		)
	}
}

func TestConditionCodeToString(t *testing.T) {
	type testCase struct {
		conditionCode ConditionCode
		Expected      string
	}
	testCases := []testCase{
		{UnknownConditionCode, ""},
		{PairedOrMultiple, "P"},
		{RequiredCondition, "R"},
		{Exclusion, "E"},
		{Conditional, "C"},
		{ListConditional, "L"},
	}

	for _, tc := range testCases {
		t.Run(
			tc.conditionCode.String(),
			func(t *testing.T) {
				assertEqual(t, tc.conditionCode.String(), tc.Expected)
				data, err := tc.conditionCode.MarshalJSON()
				assertNoError(t, err)
				var rv ConditionCode
				err = json.Unmarshal(data, &rv)
				assertNoError(t, err)
				assertEqual(t, tc.conditionCode, rv)

			},
		)
	}
}

func TestAppendSpec(t *testing.T) {
	var err error
	var specErr *SpecError

	parent := &X12Spec{
		Type:  SegmentSpec,
		Name:  "foo",
		Usage: Required,
	}
	child := &X12Spec{
		Type:  SegmentSpec,
		Name:  "bar",
		Usage: Required,
	}

	// can't append segment to segment
	err = appendSpec(parent, child)
	assertErrorNotNil(t, err)
	if !errors.As(err, &specErr) {
		t.Fatalf("Expected SpecError, got %v", err)
	}
	expectedErr := "[name: bar]: cannot append Segment to Segment: must be one of: Element, Composite"
	assertEqual(t, specErr.Error(), expectedErr)

	// parent spec type must be known
	parent.Type = SpecType(99)
	err = appendSpec(parent, child)
	assertErrorNotNil(t, err)
	if !errors.As(err, &specErr) {
		t.Fatalf("Expected SpecError, got %v", err)
	}
	expectedErr = "[name: foo]: unexpected spec type '99'"
	assertEqual(t, specErr.Error(), expectedErr)

	// child spec must have a label
	parent.Type = LoopSpec
	err = appendSpec(parent, child)
	assertErrorNotNil(t, err)
	if !errors.As(err, &specErr) {
		t.Fatalf("Expected SpecError, got %v", err)
	}
	expectedErr = "[name: bar]: spec must have a label"
	assertEqual(t, specErr.Error(), expectedErr)

	parent.Label = "foo"
	child.Label = "bar"

	err = appendSpec(parent, child)
	assertNoError(t, err)

	secondSpec := &X12Spec{
		Type:  SegmentSpec,
		Usage: Required,
		Name:  "baz",
		Label: "bar",
	}
	err = appendSpec(parent, secondSpec)
	assertErrorNotNil(t, err)
	if !errors.As(err, &specErr) {
		t.Fatalf("Expected SpecError, got %v", err)
	}
	expectedErr = "[name: baz label: bar]: spec label 'bar' already exists"
	assertEqual(t, specErr.Error(), expectedErr)

}

func TestRawMessage(t *testing.T) {
	messageText := x271Message(t)
	rawMessage, err := Read(messageText)
	assertNoError(t, err)
	assertNotNil(t, rawMessage)
	msgString := rawMessage.String()
	t.Logf("msg: %s", msgString)
	assertEqual(t, len(rawMessage.segments), 26)
	msgWithoutNewlines := replaceNewlines(t, messageText)
	assertEqual(t, msgString, msgWithoutNewlines)
	assertEqual(t, rawMessage.SegmentTerminator, '~')
	assertEqual(t, rawMessage.ElementSeparator, '*')
	assertEqual(t, rawMessage.ComponentSeparator, ':')
	assertEqual(t, rawMessage.RepetitionSeparator, '>')

	header := rawMessage.Header()
	assertNotNil(t, header)
	assertEqual(t, header.RepetitionSeparator, ">")

	trailer := rawMessage.Trailer()
	assertNotNil(t, trailer)
	assertEqual(t, trailer.ControlNumber, header.ControlNumber)
	assertEqual(t, trailer.SegmentID, ieaSegmentId)

	groupSegments, err := rawMessage.FunctionalGroupSegments()
	assertNoError(t, err)
	assertEqual(t, len(groupSegments), 1)
	assertEqual(t, groupSegments[0][0][0], gsSegmentId)
}

func Test271Stuff(t *testing.T) {
	messageText := x271Message(t)
	rawMessage, err := Read(messageText)
	assertNoError(t, err)
	msg, err := rawMessage.Message(context.Background())
	assertNoError(t, err)

	for _, txn := range msg.TransactionSets() {
		data, err := json.MarshalIndent(txn, "", "  ")
		assertNoError(t, err)
		t.Logf("data: %s", string(data))
	}

}

func slicesEqual(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := 0; i < len(left); i++ {
		val := left[i]
		if !sliceContains(right, val) {
			return false
		}
	}
	return true
}

func walkSpec(s *X12Spec, results *[]*X12Spec) {
	if len(s.ValidCodes) > 0 {
		*results = append(*results, s)
		//results = append(results, s)
	}
	for _, child := range s.Structure {
		walkSpec(child, results)
	}
}

func TestWalkSpec(t *testing.T) {
	withValidCodes := []*X12Spec{}
	dupes := map[string][]*X12Spec{}

	totalItems := []string{}

	walkSpec(X271v005010X279A1.X12Spec, &withValidCodes)
	walkSpec(X270v005010X279A1.X12Spec, &withValidCodes)
	walkSpec(X835v005010X221A1.X12Spec, &withValidCodes)

	byCt := map[string]int{}

	for _, s := range withValidCodes {
		key := strings.Join(s.ValidCodes, ",")
		_, ok := dupes[key]
		if !ok {
			dupes[key] = []*X12Spec{}
		}
		dupes[key] = append(dupes[key], s)
		totalItems = append(totalItems, s.ValidCodes...)

		byCt[key]++
	}

	totalExcludingDupes := [][]string{}
	allDupes := []string{}

	for k, v := range dupes {
		for _, sv := range v {
			t.Logf("%s (%s) (%s)", sv.Name, sv.path, sv.Description)
		}
		totalExcludingDupes = append(totalExcludingDupes, v[0].ValidCodes)
		allDupes = append(allDupes, v[0].ValidCodes...)
		t.Logf("values: %s", k)
		t.Logf("====")
	}

	t.Logf("total items: %d", len(totalItems))
	t.Logf("dupe ct: %d", len(totalExcludingDupes))
	t.Logf("dupe len: %d", len(allDupes))

	var maxFound int
	var maxKey string
	for k, v := range byCt {
		if v > maxFound {
			maxFound = v
			maxKey = k
		}
	}
	t.Logf("max: %d (%s)", maxFound, maxKey)
	for _, v := range dupes[maxKey] {
		t.Logf(" - %s / %s / %s", v.Name, v.path, v.Description)
	}
	//t.Logf("max: %s (%d) \n%#v", maxKey, maxFound, dupes[maxKey][0].ValidCodes)

}

func TestValidateMessage(t *testing.T) {
	type testCase struct {
		Name        string
		Text        []byte
		ExpectError bool
	}
	testCases := []testCase{
		{
			Name:        "270",
			Text:        x270Message(t),
			ExpectError: false,
		},
		{
			Name:        "271",
			Text:        x271Message(t),
			ExpectError: false,
		},
		{
			Name:        "835",
			Text:        x835Message(t),
			ExpectError: false,
		},
		{
			Name:        "271-unmatched",
			Text:        x271MessageUnmatchedSegment(t),
			ExpectError: true,
		},
		{
			Name:        "271-mismatched-control-numbers",
			Text:        x271MessageMismatchedControlNumbers(t),
			ExpectError: true,
		},
	}

	for i, tc := range testCases {
		t.Run(
			fmt.Sprintf("%d-%s", i, tc.Name),
			func(t *testing.T) {
				err := Validate(tc.Text)
				if tc.ExpectError {
					assertErrorNotNil(t, err)
					t.Logf("got error(s): %s", err.Error())
				} else {
					assertNoError(t, err)
				}
			},
		)
	}
}

func TestValidateLoopSpecSegment(t *testing.T) {
	segmentSpec := &X12Spec{
		Type:      SegmentSpec,
		Name:      "HL",
		Usage:     Situational,
		RepeatMax: 1,
		Structure: []*X12Spec{
			{
				Type:        ElementSpec,
				Name:        "HL01",
				Usage:       Required,
				Description: "Hierarchical id number",
				Label:       "hierarchicalIdNumber",
				MinLength:   1,
				MaxLength:   12,
				DataType:    String,
			},
			{
				Type:        ElementSpec,
				Name:        "HL02",
				Usage:       Required,
				Description: "Hierarchical parent id number",
				Label:       "hierarchicalParentIdNumber",
				MinLength:   1,
				MaxLength:   12,
				DataType:    String,
			},
			{
				Type:        ElementSpec,
				Name:        "HL03",
				Usage:       Required,
				Description: "Hierarchical level code",
				Label:       "hierarchicalLevelCode",
				ValidCodes:  []string{"23"},
				MinLength:   1,
				MaxLength:   2,
				DefaultVal:  "23",
				DataType:    Identifier,
			},
			{
				Type:        ElementSpec,
				Name:        "HL04",
				Usage:       Required,
				Description: "Hierarchical child code",
				Label:       "hierarchicalChildCode",
				ValidCodes:  []string{"0"},
				MinLength:   1,
				MaxLength:   1,
				DefaultVal:  "0",
				DataType:    Identifier,
			},
		},
	}
	loopSpec := &X12Spec{
		Type:      LoopSpec,
		Usage:     Required,
		Name:      "2000C",
		Structure: []*X12Spec{segmentSpec},
	}

	err := loopSpec.validateSpec()
	assertErrorNotNil(t, err)
	assertStringContains(
		t,
		err.Error(),
		"first segment in loop must match loop usage (REQUIRED)",
	)
}

func TestSpecTreeString(t *testing.T) {
	s := X271v005010X279A1.treeString("")
	t.Logf("\n%s", s)
}

func TestSpecJSON(t *testing.T) {
	testCases := []*X12TransactionSetSpec{
		X270v005010X279A1,
		X271v005010X279A1,
		X835v005010X221A1,
	}
	for _, tc := range testCases {
		t.Run(
			tc.Key,
			func(t *testing.T) {
				data, err := json.Marshal(tc)
				assertNoError(t, err)
				var newTxnSpec *X12TransactionSetSpec
				err = json.Unmarshal(data, &newTxnSpec)
				assertNoError(t, err)
				assertEqual(t, tc.Key, newTxnSpec.Key)
				assertEqual(t, tc.Description, newTxnSpec.Description)
				assertEqual(t, len(tc.Structure), len(newTxnSpec.Structure))

			},
		)
	}
}

func TestMsgJSON(t *testing.T) {
	msg := unmarshalText(t, x270Message(t))
	data, err := json.MarshalIndent(msg, "", "  ")
	assertNoError(t, err)
	t.Logf("result:\n%s", string(data))
	t.Logf("=======")

	msg = unmarshalText(t, x271Message(t))
	data, err = json.MarshalIndent(msg, "", "  ")
	assertNoError(t, err)
	t.Logf("result:\n%s", string(data))
	t.Logf("=======")

	msg = unmarshalText(t, x835Message(t))
	data, err = json.MarshalIndent(msg, "", "  ")
	assertNoError(t, err)
	t.Logf("result:\n%s", string(data))
}

func TestUnknownTypePayload(t *testing.T) {
	n := &X12Node{}
	payload, err := n.Payload()
	assertErrorNotNil(t, err)
	if payload != nil {
		t.Errorf("Expected nil payload, got %v", payload)
	}

	n.Type = RepeatCompositeNode
	payload, err = n.Payload()
	assertErrorNotNil(t, err)
	if payload != nil {
		t.Errorf("Expected nil payload, got %v", payload)
	}
}

func Test271Missing2100C(t *testing.T) {
	text := x271Missing2100C(t)
	newMessage, err := Read(text)
	assertNoError(t, err)
	msg, err := newMessage.Message(context.Background())
	assertNoError(t, err)

	err = msg.Validate()
	assertErrorNotNil(t, err)

	if !errors.Is(err, ErrRequiredLoopMissing) {
		t.Errorf("Expected ErrRequiredLoopMissing, got %v", err)
	}
	assertStringContains(
		t,
		err.Error(),
		"[loop: 2100C] [description: Subscriber name] [path: /ST_LOOP/2000A/2000B/2000C/2100C]: missing required loop",
	)

	data, _ := json.MarshalIndent(msg, "", "  ")
	t.Logf("%s", string(data))
}

func Test270MissingReceiverName(t *testing.T) {
	text := x270MissingReceiverName(t)
	newMessage, err := Read(text)
	assertNoError(t, err)
	msg, err := newMessage.Message(context.Background())
	assertNoError(t, err)

	err = msg.Validate()
	assertErrorNotNil(t, err)

	if !errors.Is(err, ErrRequiredElementMissing) {
		t.Errorf("Expected ErrRequiredElementMissing, got %v", err)
	}
	assertStringContains(
		t,
		err.Error(),
		"[type: 'Element' name: 'NM103' path: '/X12/GS_LOOP[0]/ST_LOOP[0]/2000A/2000B/2100B/NM1/NM103']: missing required element",
	)

	data, _ := json.MarshalIndent(msg, "", "  ")
	t.Logf("%s", string(data))
}

func TestNewReader(t *testing.T) {
	reader, err := NewReader(
		X271v005010X279A1,
		X270v005010X279A1,
		X835v005010X221A1,
	)
	assertNoError(t, err)
	assertSliceContains(t, reader.transactionSpecs, X270v005010X279A1)
	assertSliceContains(t, reader.transactionSpecs, X271v005010X279A1)
	assertSliceContains(t, reader.transactionSpecs, X835v005010X221A1)
	assertEqual(t, len(reader.transactionSpecs), 3)
	assertSlicesEqual(t, reader.TransactionSpecs(), reader.transactionSpecs)

	type testCase struct {
		Name        string
		Text        []byte
		ExpectError bool
	}
	testCases := []testCase{
		{
			Name:        "270",
			Text:        x270Message(t),
			ExpectError: false,
		},
		{
			Name:        "271",
			Text:        x271Message(t),
			ExpectError: false,
		},
		{
			Name:        "835",
			Text:        x835Message(t),
			ExpectError: false,
		},
		{
			Name:        "271-unmatched",
			Text:        x271MessageUnmatchedSegment(t),
			ExpectError: true,
		},
		{
			Name:        "271-mismatched-control-numbers",
			Text:        x271MessageMismatchedControlNumbers(t),
			ExpectError: true,
		},
	}

	for i, tc := range testCases {
		t.Run(
			fmt.Sprintf("%d-%s", i, tc.Name),
			func(t *testing.T) {

				rawMessage, e := reader.Read(tc.Text)
				assertNoError(t, e)

				msg, e := rawMessage.Message(context.Background())
				if tc.ExpectError {
					if e == nil {
						e = msg.Validate()
					}
					assertErrorNotNil(t, e)
				} else {
					assertNoError(t, e)
					e = msg.Validate()
					assertNoError(t, e)

					messageSegments := msg.Segments()
					assertEqual(
						t,
						len(messageSegments),
						len(rawMessage.segments),
					)
					for ind, seg := range messageSegments {
						assertEqual(t, seg.index, ind)
					}

				}

				//assertNoError(t, e)

				//e = msg.Validate()
				//if tc.ExpectError {
				//	assertErrorNotNil(t, e)
				//} else {
				//	// validates that the order of the segments returned
				//	// reflects the index they were assigned by the
				//	// original Reader
				//	assertNoError(t, e)
				//	messageSegments := msg.Segments()
				//	assertEqual(
				//		t,
				//		len(messageSegments),
				//		len(rawMessage.segments),
				//	)
				//	for ind, seg := range messageSegments {
				//		assertEqual(t, seg.index, ind)
				//	}
				//}
			},
		)
	}
}

func TestPathPayload(t *testing.T) {
	reader, err := NewReader(X271v005010X279A1)
	assertNoError(t, err)
	reader.transactionSpecs = []*X12TransactionSetSpec{}

	type testCase struct {
		Name string
		Text []byte
	}
	testCases := []testCase{
		{
			Name: "270",
			Text: x270Message(t),
		},
		{
			Name: "271",
			Text: x271Message(t),
		},
		{
			Name: "835",
			Text: x835Message(t),
		},
	}

	for i, tc := range testCases {
		t.Run(
			fmt.Sprintf("%d-%s", i, tc.Name),
			func(t *testing.T) {

				rawMessage, e := reader.Read(tc.Text)
				assertNoError(t, e)
				msg, e := rawMessage.Message(context.Background())
				data, err := json.MarshalIndent(msg, "", "  ")
				assertNoError(t, err)
				t.Logf("result:\n%s", string(data))

			},
		)
	}
}

// TestX12TransactionSetSpecJSON validates that we can marshal a spec to
// JSON, then unmarshal it into an equivalent spec
func TestX12TransactionSetSpecJSON(t *testing.T) {
	type testCase struct {
		Spec *X12TransactionSetSpec
	}
	testCases := []testCase{
		{Spec: X270v005010X279A1},
		{Spec: X271v005010X279A1},
		{Spec: X835v005010X221A1},
	}
	for _, tc := range testCases {
		t.Run(
			fmt.Sprintf(tc.Spec.Key), func(t *testing.T) {
				data, err := json.Marshal(tc.Spec)
				assertNoError(t, err)
				t.Logf("%s length: %d", tc.Spec.Key, len(data))

				newSpec := &X12TransactionSetSpec{}
				err = json.Unmarshal(data, newSpec)
				assertNoError(t, err)
				assertEqual(t, newSpec.Key, tc.Spec.Key)
				pathCt := len(newSpec.PathMap)
				if pathCt == 0 {
					t.Errorf("Expected path map to have at least one entry")
				}
				assertEqual(t, len(newSpec.PathMap), len(tc.Spec.PathMap))

				for sk, sv := range newSpec.PathMap {
					originalSpec, ok := tc.Spec.PathMap[sk]
					if !ok {
						t.Errorf(
							"Expected path %s to exist in original spec",
							sk,
						)
						continue
					}
					assertEqual(t, sv.Name, originalSpec.Name)
					assertEqual(t, sv.Type, originalSpec.Type)
					assertEqual(t, sv.Usage, originalSpec.Usage)
					assertEqual(t, sv.Description, originalSpec.Description)
					assertEqual(t, sv.Label, originalSpec.Label)

					assertEqual(t, sv.MinLength, originalSpec.MinLength)
					assertEqual(t, sv.MaxLength, originalSpec.MaxLength)

					assertEqual(t, sv.RepeatMax, originalSpec.RepeatMax)
					assertEqual(t, sv.RepeatMin, originalSpec.RepeatMin)

					assertEqual(t, sv.DefaultVal, originalSpec.DefaultVal)
					assertEqual(t, sv.DataType, originalSpec.DataType)
					assertSlicesEqual(t, sv.ValidCodes, originalSpec.ValidCodes)
					assertEqual(t, sv.path, originalSpec.path)
					assertEqual(t, sv.loopDepth, originalSpec.loopDepth)

					if sv.Type != TransactionSetSpec {
						assertNotNil(t, sv.parent)
					}
				}
			},
		)
	}
}

// TestX12TransactionSetSpecJSON validates that we can marshal a spec to
// JSON, then unmarshal it into an equivalent spec
func TestX12TransactionSetJSONSchema(t *testing.T) {
	type testCase struct {
		Spec *X12TransactionSetSpec
	}
	testCases := []testCase{
		{Spec: X270v005010X279A1},
		{Spec: X271v005010X279A1},
		{Spec: X835v005010X221A1},
	}
	for _, tc := range testCases {
		t.Run(
			fmt.Sprintf(tc.Spec.Key), func(t *testing.T) {
				js := tc.Spec.newSchema()
				data, err := json.MarshalIndent(js, "", "  ")
				assertNoError(t, err)
				if len(data) == 0 {
					t.Fatal("Expected JSON schema to have length > 0")
				}
				//t.Logf("spec length: %d", len(data))
				t.Logf("schema:\n%s", string(data))
			},
		)
	}
}

func TestFindRepeatable(t *testing.T) {
	matching := X271v005010X279A1.findName("2110C")
	assertEqual(t, len(matching), 1)
	s := matching[0]
	assertEqual(t, s.IsRepeatable(), true)

}

func TestFindType(t *testing.T) {
	matching := X271v005010X279A1.findType(LoopSpec)
	if len(matching) == 0 {
		t.Fatalf("Expected to find at least one loop")
	}
	for _, m := range matching {
		assertEqual(t, m.Type, LoopSpec)
	}
}

func TestFindName(t *testing.T) {
	matching := X271v005010X279A1.findName("NM101")
	if len(matching) == 0 {
		t.Fatalf("Expected to find at least one loop")
	}
	for _, m := range matching {
		assertEqual(t, m.Name, "NM101")
	}
}
