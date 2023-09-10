package edx12

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestLoadSingleSpec(t *testing.T) {
	clearRegisteredSpecs()
	assertEqual(t, len(transactionSpecs), 0)
	specs, err := loadTransactionSpecFiles(&x12Specs)
	assertNoError(t, err)
	assertEqual(t, len(specs), 3)
	specFiles, err := getAllFilenames(&x12Specs)
	assertNoError(t, err)

	for _, s := range specFiles {
		xspec := &X12TransactionSetSpec{}
		content, err := x12Specs.ReadFile(s)
		assertNoError(t, err)
		err = json.Unmarshal(content, xspec)
		assertNoError(t, err)
	}

}

func TestLoadSpecs(t *testing.T) {
	clearRegisteredSpecs()
	assertEqual(t, len(transactionSpecs), 0)
	specs, err := loadTransactionSpecFiles(&x12Specs)
	assertNoError(t, err)
	assertEqual(t, len(specs), 3)
	//for _, s := range specs {
	//	t.Logf("spec: %s: %#v", s.TransactionSetCode, s)
	//for _, subSpec := range s.Structure {
	//	t.Logf("subspec: %s: %#v", subSpec.Name, subSpec)
	//}
	//newData, err := json.MarshalIndent(s, "", "  ")
	//if err != nil {
	//	t.Fatalf("expected no error, got %s", err.Error())
	//}
	//t.Logf("pretty:\n%s", string(newData))

	//for k, v := range s.PathMap {
	//	t.Logf("path: %s: [%s] %s", k, v.Type, v.Name)
	//}
	//}
}

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
				segments := msg.findSegments(tc.Name)
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
	e := newError(node, ErrElementHasInvalidValue)
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
		defaultVal, err = elementNodeWithDefaultValue(elementSpec)
		assertNoError(t, err)
		if !reflect.DeepEqual(defaultVal.Value, expectedVal) {
			t.Errorf("Expected %v, got %v", expectedVal, defaultVal.Value)
		}

		// defaultVal should take precedence over validCodes
		expectedDefault = "XX"
		elementSpec.DefaultVal = expectedDefault
		expectedVal[0] = expectedDefault
		defaultVal, err = elementNodeWithDefaultValue(elementSpec)
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

		defaultVal, err = elementNodeWithDefaultValue(elementSpec)
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
		defaultVal, err = elementNodeWithDefaultValue(elementSpec)
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
			Index:    isaIndexAuthInfo,
			Value:    "Foo",
			Expected: "       Foo",
		},
		{
			Index:    isaIndexSecurityInfo,
			Value:    "Foo",
			Expected: "       Foo",
		},
		{
			Index:    isaIndexSenderId,
			Value:    "Sender",
			Expected: "         Sender",
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
				case isaIndexAuthInfo:
					failOnErr(t, msg.SetAuthInfo(tc.Value))
				case isaIndexSecurityInfo:
					failOnErr(t, msg.SetSecurityInfo(tc.Value))
				case isaIndexSenderId:
					failOnErr(t, msg.SetSenderId(tc.Value))
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
	msg := unmarshalText(t, messageText)
	msgFormat := msg.Format()
	msgWithoutNewlines := replaceNewlines(t, messageText)

	fmt.Println("Message: ", msgFormat)
	if msgFormat != msgWithoutNewlines {
		t.Errorf(
			"Expected message format to match original message:\nOriginal: %s\nFormatted:%s\n",
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

func TestValidateUnknownUsage(t *testing.T) {
	s := &X12Spec{
		Usage: Usage(99),
	}
	assertErrorNotNil(t, s.validateUsage())

	s.Usage = UnknownUsage
	assertErrorNotNil(t, s.validateUsage())
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
				}
				if tc.IsValid == true {
					assertNoError(t, s.validateMinMaxLength())
				} else {
					assertErrorNotNil(t, s.validateMinMaxLength())
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
		t.Fatalf("Expected error to be SpecErr")
	}
}

func TestSegmentNodeReplace(t *testing.T) {
	segNode, e := NewNode(SegmentNode, "NM1")
	assertNoError(t, e)

	elemNode, e := NewNode(ElementNode, "NM101")
	assertNoError(t, e)
	failOnErr(t, elemNode.SetValue("1"))
	failOnErr(t, segNode.Append(elemNode))

	elemNodeB, e := NewNode(ElementNode, "NM102")
	assertNoError(t, e)
	failOnErr(t, elemNodeB.SetValue("2"))
	failOnErr(t, segNode.Append(elemNodeB))

	elemNodeC, e := NewNode(ElementNode, "NM103")
	assertNoError(t, e)
	failOnErr(t, elemNodeC.SetValue("3"))
	failOnErr(t, segNode.Append(elemNodeC))

	elemNodeD, e := NewNode(ElementNode, "")
	assertNoError(t, e)
	failOnErr(t, elemNodeD.SetValue("F"))

	err := segNode.Replace(2, elemNodeD)
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

func TestMissingTxnChildren(t *testing.T) {
	// The given node should either have zero or at least two
	// children, and neither the first nor last child should be
	// nil. If this is not the case, an error should be returned.
	node := &X12Node{Children: []*X12Node{nil}}
	_, err := createTransactionNode(node)
	assertErrorNotNil(t, err)

	var ne *NodeError
	if !errors.As(err, &ne) {
		t.Errorf("Expected NodeError, got %v", err)
	}

	node.Children[0] = &X12Node{}
	node.Children = append(node.Children, nil)
	_, err = createTransactionNode(node)
	assertErrorNotNil(t, err)

	var sne *NodeError
	if !errors.As(err, &sne) {
		t.Errorf("Expected NodeError, got %v", err)
	}

	node.Children = []*X12Node{&X12Node{}}
	_, err = createTransactionNode(node)
	assertErrorNotNil(t, err)
	var eee *NodeError
	if !errors.As(err, &eee) {
		t.Errorf("Expected NodeError, got %v", err)
	}
}

func TestDefaultLoopObj(t *testing.T) {
	spec := getSpec(t, "270", "005010X279A1")
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
	assertNotNil(t, hlSegment.Spec)
	assertEqual(t, hlSegment.Name, hlSegmentId)

	if len(hlSegment.Children) <= 1 {
		t.Fatal("Expected HL segment to have children")
	}
	randomLoop := &X12Node{Type: LoopNode, Name: "SOMELOOP"}
	replaceErr := hlSegment.Replace(0, randomLoop)
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
	node := &X12Node{Type: ElementNode}
	assertErrorNotNil(t, validateSegmentNode(node))

	node.Type = SegmentNode
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
		Parent: rootNode,
	}
	bottomNode := &X12Spec{
		Type:   SegmentSpec,
		Name:   "NM1",
		Label:  "baz",
		Parent: middleNode,
	}
	labels := bottomNode.parentLabels()
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
	assertErrorNotNil(t, spec.validateMinMaxLength())
}

func TestSpecValidCodes(t *testing.T) {
	spec := &X12Spec{Type: LoopSpec, ValidCodes: []string{"foo", "bar"}}
	assertErrorNotNil(t, spec.validateValidCodes())
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
	assertEqual(t, len(TransactionSpecs()), 3)
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
	f := msg.Format()
	origText := replaceNewlines(t, msgText)
	t.Logf("formatted: %s", f)
	assertEqual(t, f, origText)

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
		for _, element := range seg.Elements() {
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
			mismatchedSeg.Elements(),
		)
	}

	mismatchedId := newSegment(t, "REF*EH*FOO*BAR", "*", "", "")

	isMatch, _ = segmentMatchesSpec(mismatchedId, segSpec)
	if isMatch {
		t.Errorf(
			"Expected segment to not match definition (elements: %v",
			mismatchedId.Elements(),
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
				types := tc.Spec.allowedNodeTypes()
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
	msg := unmarshalText(t, msgBytes)

	targetSpec := x271Spec(t)

	txns := msg.TransactionSets()
	if len(txns) != 1 {
		t.Fatalf("No segments found")
	}
	targetTransaction := txns[0]
	targetTransaction.TransactionSpec = targetSpec
	err := targetTransaction.Transform()

	expectedErr := "[segment: DMG] [path: /X12/GS_LOOP[0]/ST_LOOP[0]/2000A/2000B/2000C/2100C/DMG]: segment condition validation failed: when at least one value in [1 2] is present, all must be present"

	if err == nil {
		t.Fatalf("Expected error:\n%q\ngot: nil", expectedErr)
	} else if err.Error() != expectedErr {
		t.Errorf("Expected error:\n%q\ngot:\n%s", expectedErr, err.Error())
	}

	validationErrors := txns[0].Validate()
	if validationErrors == nil {
		t.Fatalf("Expected validation errors")
	}
	fmt.Printf("validation errors:\n%v", validationErrors)

}

func Test271UnmatchedSegment(t *testing.T) {
	messageText := x271MessageUnmatchedSegment(t)
	msg := unmarshalText(t, messageText)
	targetSpec := x271Spec(t)

	txn := msg.TransactionSets()[0]
	txn.TransactionSpec = targetSpec
	err := txn.Transform()
	segs := txn.Segments()
	if len(segs) != 8 {
		t.Errorf("Expected 8 segments, got %d", len(segs))
	}
	if segs[7].Name != "TRN" {
		t.Errorf("Expected segment TRN, got %s", segs[8].Name)
	}
	if err == nil {
		t.Errorf("expected error")
	}
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
	msg := NewMessage()
	err := UnmarshalText([]byte(messageText), msg)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if !errors.Is(err, ErrInvalidISA) {
		t.Fatalf("Expected ErrInvalidISA, got %v", err)
	}
	expectedError := "[type: 'Segment' name: 'ISA']: invalid ISA segment: expected 17 elements, got 16"
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got: '%s'", expectedError, err)
	}
}

func TestMismatchedControlNumbers(t *testing.T) {
	file := x270MessageMismatchedControlNumbers(t)
	msg := unmarshalText(t, file)
	errs := msg.Validate()
	if !errors.Is(errs, ErrInvalidTransactionEnvelope) {
		t.Fatalf(
			"Expected ErrInvalidTransactionEnvelope, got %v",
			errs,
		)
	}
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
	s := &X12Spec{Usage: Required}
	assertEqual(t, s.IsRepeatable(), false)

	s.Usage = Required
	assertEqual(t, s.IsRepeatable(), false)

	s.RepeatMin = 1
	assertEqual(t, s.IsRepeatable(), false)

	s.RepeatMax = 2
	assertEqual(t, s.IsRepeatable(), true)

	s.RepeatMin = 0
	assertEqual(t, s.IsRepeatable(), true)

	s.RepeatMax = 1
	assertEqual(t, s.IsRepeatable(), false)

	s.RepeatMax = 2
	s.Usage = NotUsed
	assertEqual(t, s.IsRepeatable(), false)

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
					assertNoError(t, seg.Append(elemNode))
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
					assertNoError(t, seg.Append(elemNode))
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

					assertNoError(t, seg.Append(elemNode))

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
					assertNoError(t, seg.Append(elemNode))

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
	segElements := seg.Elements()
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
	elemErr := seg.Append(ilElem)
	if elemErr != nil {
		t.Fatalf("Error appending element: %v", elemErr)
	}

	validationErr := seg.Validate()
	if validationErr == nil {
		t.Fatalf("Expected error, got nil")
	}
	expectedErr := "[segment: NM1] [path: ]: spec defines 0 elements, segment has 1"
	if validationErr.Error() != expectedErr {
		t.Errorf(
			"Expected error %q, got %q",
			expectedErr,
			validationErr.Error(),
		)
	}
}

func TestReaderJSON(t *testing.T) {
	msgText := x270Message(t)
	msg := unmarshalText(t, msgText)
	t.Logf("msg: %#v", msg)
	f := msg.Format()
	origText := replaceNewlines(t, msgText)
	t.Logf("formatted: %s", f)
	assertEqual(t, f, origText)

	for _, grp := range msg.functionalGroups {
		for _, txn := range grp.Transactions {
			e := txn.Transform()
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

	elements := node.Elements()
	assertEqual(t, len(elements), 3)

	idValue := elements[0].Value[0]
	assertEqual(t, idValue, "NM1")

	idSpec := elements[0].Spec
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
	assertErrorNotNil(t, n.Append(childNode))
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
	node.Spec = cmpSpec
	assertErrorNotNil(t, node.SetValue("foo", "bar", "baz"))

	assertNoError(t, node.SetValue("foo", "bar"))

}

func TestAppendCompositeNode(t *testing.T) {
	n, err := NewNode(SegmentNode, "NM1", "NM1", "IL", "1")
	assertNoError(t, err)
	childNode := &X12Node{Type: CompositeNode}
	assertNoError(t, n.Append(childNode))
	assertEqual(t, childNode.Name, "NM103")
}

func TestReader835(t *testing.T) {
	msgText := x835Message(t)
	msg := unmarshalText(t, msgText)
	t.Logf("msg: %#v", msg)
	f := msg.Format()
	origText := replaceNewlines(t, msgText)
	t.Logf("formatted: %s", f)
	assertEqual(t, f, origText)

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
