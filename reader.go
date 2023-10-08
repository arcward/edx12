package edx12

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

var ParseError = errors.New("parse error")
var _defaultReader = &Reader{}
var defaultTransactionSpecs = []*X12TransactionSetSpec{
	X270v005010X279A1,
	X271v005010X279A1,
	X835v005010X221A1,
}

type RawSegment []string

func (s RawSegment) ID() string {
	if len(s) == 0 {
		return ""
	}
	return s[0]
}

func (r RawMessage) newSegmentNodes() ([]*X12Node, error) {
	nodes := make([]*X12Node, 0, len(r.segments))
	errs := []error{}
	errFmt := "segment %d, element %d: %w"
	for ind, rawSegment := range r.segments {
		segID := rawSegment.ID()
		segNode := &X12Node{
			Type:     SegmentNode,
			Children: make([]*X12Node, 0, len(rawSegment)),
			index:    ind,
			Name:     segID,
		}
		nodes = append(nodes, segNode)

		idNode, _ := NewNode(ElementNode, segID, segID)
		segNode.Children = append(segNode.Children, idNode)
		idNode.Parent = segNode
		idSpec := &X12Spec{
			Type:        ElementSpec,
			Usage:       Required,
			RepeatMin:   1,
			RepeatMax:   1,
			ValidCodes:  []string{segID},
			Description: "Segment ID",
			DefaultVal:  segID,
		}
		if err := idNode.SetSpec(idSpec); err != nil {
			errs = append(errs, newNodeError(idNode, err))
		}
		if len(rawSegment) == 1 {
			continue
		}

		for elemInd := 1; elemInd < len(rawSegment); elemInd++ {
			// Segment ID is the first element, any element after that
			// gets an index, so the first element value being `REF` means
			// the next element gets named `REF01`, ...
			elemNode := &X12Node{
				Parent: segNode,
				Type:   ElementNode,
				Name:   fmt.Sprintf("%s%02d", segID, elemInd),
			}
			segNode.Children = append(segNode.Children, elemNode)
			v := rawSegment[elemInd]
			var containsRepetitionSeparator bool
			var containsComponentSeparator bool

			if segID != isaSegmentId {
				containsRepetitionSeparator = strings.Contains(
					v,
					string(r.RepetitionSeparator),
				)
				containsComponentSeparator = strings.Contains(
					v,
					string(r.ComponentSeparator),
				)
			}

			switch {
			case containsRepetitionSeparator:
				elemNode.Type = RepeatElementNode
				elemNode.Value = strings.Split(
					v,
					string(r.RepetitionSeparator),
				)
			case containsComponentSeparator:
				elemNode.Type = CompositeNode
				values := strings.Split(
					v,
					string(r.ComponentSeparator),
				)

				if e := populateCompositeNode(
					elemNode,
					values...,
				); e != nil {
					errs = append(
						errs,
						newNodeError(
							segNode,
							fmt.Errorf(errFmt, ind, elemInd, e),
						),
					)
				}
			default:
				elemNode.Value = make([]string, 1, 1)
				elemNode.Value[0] = v
			}
		}
		var controlSpec *X12Spec
		switch segID {
		case isaSegmentId:
			controlSpec = isaHeaderSpec
		case ieaSegmentId:
			controlSpec = ieaTrailerSpec
		case gsSegmentId:
			controlSpec = gsHeaderSpec
		case geSegmentId:
			controlSpec = geTrailerSpec
		case stSegmentId:
			controlSpec = stHeaderSpec
		case seSegmentId:
			controlSpec = seTrailerSpec
		}
		if controlSpec != nil {
			if err := segNode.SetSpec(controlSpec); err != nil {
				errs = append(errs, newNodeError(segNode, err))
			}
		}
	}
	return nodes, errors.Join(errs...)
}

//func NewRawMessage(data []byte) (RawMessage, error) {
//	msg := RawMessage{}
//	if err := msg.Read(data); err != nil {
//		return msg, err
//	}
//	return msg, nil
//}

type groupIndex struct {
	Start int
}

// groupByFirstIndexValue takes a slice of slices, and returns a slice of slices
// where each slice's first element starts with the given startValue, and
// the last slice's first element ends with the given endValue.
// The returned slice of slices will contain all segments between them,
// inclusive
func groupByFirstIndexValue(
	segments []RawSegment,
	startValue string,
	endValue string,
) (segmentGroups [][]RawSegment, err error) {
	var indexes []*groupIndex
	var currentIndex *groupIndex
	for ind, segment := range segments {
		segmentName := segment[0]
		if segmentName == startValue {
			if currentIndex != nil {
				return segmentGroups, fmt.Errorf(
					"found '%v' before '%v'",
					startValue,
					endValue,
				)
			}
			currentIndex = &groupIndex{Start: ind}
			indexes = append(indexes, currentIndex)
		} else if segmentName == endValue {
			if currentIndex == nil {
				return segmentGroups, fmt.Errorf(
					"unexpected state- found segment Trailer %v, but not currently in a segment group (already have: %v)",
					segmentName,
					segmentGroups,
				)
			}
			segmentGroups = append(
				segmentGroups,
				segments[currentIndex.Start:ind+1],
			)
			currentIndex = nil
		}
	}
	return segmentGroups, err
}

type RawMessage struct {
	SegmentTerminator   rune
	ElementSeparator    rune
	RepetitionSeparator rune
	ComponentSeparator  rune
	text                string
	segments            []RawSegment
	transactionSetSpecs []*X12TransactionSetSpec
	reader              *Reader
}

func (r *RawMessage) String() string {
	var b strings.Builder
	for _, segment := range r.segments {
		for i, element := range segment {
			if i > 0 {
				b.WriteRune(r.ElementSeparator)
			}
			b.WriteString(element)
		}
		b.WriteRune(r.SegmentTerminator)
	}
	return b.String()
}

func (r *RawMessage) Segments() []RawSegment {
	return r.segments
}

// FunctionalGroupSegments a slice of slices, where each slice begins with a GS
// segment and ends with a GE segment
func (r *RawMessage) FunctionalGroupSegments() ([][]RawSegment, error) {
	return groupByFirstIndexValue(r.segments, gsSegmentId, geSegmentId)
}

// TransactionSegments returns a slice of slices, where each slice begins
// with an ST segment and ends with an SE segment
func (r *RawMessage) TransactionSegments() ([][]RawSegment, error) {
	return groupByFirstIndexValue(r.segments, stSegmentId, seSegmentId)
}

// UnmarshalText implements the encoding.TextUnmarshaler interface
func (r *RawMessage) UnmarshalText(data []byte) error {
	if r.reader == nil {
		r.reader = _defaultReader
	}
	rawMessage, err := r.reader.Read(data)
	*r = *rawMessage
	return err
}

// Reader returns the Reader that was used to read/create the RawMessage,
// if any.
func (r *RawMessage) Reader() *Reader {
	return r.reader
}

// Message creates a new Message node from the RawMessage segments.
// All TransactionSetNode nodes will attempt to have a TransactionSetSpec
// set, if one matches. Either the default set of specs will be used, or
// the ones attached to the Reader that was used to read the RawMessage.
// Additional specs can be provided, which will take precedence.
func (r *RawMessage) Message(
	ctx context.Context,
	transactionSetSpecs ...*X12TransactionSetSpec,
) (*Message, error) {
	for _, txnSpec := range transactionSetSpecs {
		if err := txnSpec.Validate(); err != nil {
			return nil, err
		}
	}
	if r.reader != nil {
		for _, txnSpec := range r.reader.transactionSpecs {
			transactionSetSpecs = append(transactionSetSpecs, txnSpec)
		}
	}

	msg := &Message{
		X12Node: &X12Node{
			Type: MessageNode,
			Name: "X12",
		},
		rawMessage: r,
	}

	var validationErrors []error
	var isaNode *X12Node
	var ieaNode *X12Node

	segmentNodes, err := r.newSegmentNodes()
	msg.segments = segmentNodes
	msg.Children = []*X12Node{}
	if err != nil {
		validationErrors = append(validationErrors, err)
	}
	if len(segmentNodes) > 0 {
		isaNode = segmentNodes[0]
		ieaNode = segmentNodes[len(segmentNodes)-1]
	} else {
		validationErrors = append(
			validationErrors,
			fmt.Errorf("expected at least one segment, got zero"),
		)
		err = errors.Join(validationErrors...)
		return msg, err
	}
	msg.Header = isaNode
	msg.Trailer = ieaNode

	for _, n := range segmentNodes {
		n.Parent = msg.X12Node
	}

	// Group segments by ISA/IEA, then group the segments by GS/GE
	groupedEnvelope, err := groupNodesByName(
		isaSegmentId,
		ieaSegmentId,
		segmentNodes,
	)
	if err != nil {
		validationErrors = append(validationErrors, err)
	}
	if len(groupedEnvelope) == 0 {
		validationErrors = append(
			validationErrors,
			fmt.Errorf("expected at least one functional group, got zero"),
		)
	}

	newMsgChildren := make(
		[]*X12Node,
		0,
		len(groupedEnvelope)+2,
	)
	newMsgChildren = append(newMsgChildren, isaNode)
	groupedSegments, err := groupNodesByName(
		gsSegmentId, geSegmentId, groupedEnvelope[0],
	)
	if err != nil {
		validationErrors = append(validationErrors, err)
	}

	msg.functionalGroups = make([]*FunctionalGroupNode, 0, len(groupedSegments))
	// Create functional groups from gsSegmentId/geSegmentId-bounded segments
	for _, functionalSegments := range groupedSegments {
		fGroup := &FunctionalGroupNode{
			X12Node: &X12Node{
				Type: GroupNode,
				Name: functionalGroupName,
			},
		}
		msg.functionalGroups = append(msg.functionalGroups, fGroup)
		newMsgChildren = append(newMsgChildren, fGroup.X12Node)
		fGroup.Header = functionalSegments[0]
		fGroup.Trailer = functionalSegments[len(functionalSegments)-1]
		fGroup.Children = append(fGroup.Children, fGroup.Header)

		transactionSegments, grpErr := groupNodesByName(
			stSegmentId,
			seSegmentId,
			functionalSegments,
		)
		if grpErr != nil {
			validationErrors = append(validationErrors, grpErr)
		}

		for _, stSegments := range transactionSegments {
			t := &TransactionSetNode{
				segments: stSegments,
				X12Node: &X12Node{
					Type:   TransactionNode,
					Name:   transactionSetName,
					Parent: fGroup.X12Node,
				},
			}
			fGroup.Transactions = append(fGroup.Transactions, t)
			fGroup.Children = append(fGroup.Children, t.X12Node)
			t.Group = fGroup
			t.Header = stSegments[0]
			for i := 1; i < len(t.Header.Children); i++ {
				switch i {
				case stIndexTransactionSetCode:
					t.TransactionSetCode = t.Header.Children[i].Value[0]
				case stIndexVersionCode:
					t.VersionCode = t.Header.Children[i].Value[0]
				case stIndexControlNumber:
					t.ControlNumber = t.Header.Children[i].Value[0]
				}
			}

			t.Trailer = stSegments[len(stSegments)-1]
			t.Children = append(t.Children, t.Header)

			for ti := 1; ti < len(stSegments); ti++ {
				t.Children = append(t.Children, stSegments[ti])
				stSegments[ti].Parent = t.X12Node
			}
			var levels []*HierarchicalLevel
			levels, err = createHierarchicalLevels(t.X12Node)
			t.HierarchicalLevels = levels
			if err != nil {
				validationErrors = append(
					validationErrors,
					newNodeError(t.X12Node, err),
				)
			}
		}
		fGroup.Children = append(fGroup.Children, fGroup.Trailer)
		for gi := 0; gi < len(fGroup.Children); gi++ {
			fGroup.Children[gi].Parent = fGroup.X12Node
		}

		if ctx.Err() != nil {
			break
		}
	}

	newMsgChildren = append(newMsgChildren, ieaNode)
	msg.Children = newMsgChildren
	for i := 0; i < len(msg.Children); i++ {
		msg.Children[i].Parent = msg.X12Node
	}
	validationErrors = append(validationErrors, msg.X12Node.setPath())

	txnSets := msg.TransactionSets()

	for i := 0; i < len(txnSets); i++ {
		var txn *TransactionSetNode
		txn = txnSets[i]
		if err = txn.TransformWithContext(
			ctx,
			transactionSetSpecs...,
		); err != nil {
			validationErrors = append(validationErrors, err)
		}
	}

	validationErrors = append(validationErrors, msg.X12Node.detectCycles())
	err = errors.Join(validationErrors...)
	return msg, err
}

type InterchangeHeader struct {
	SegmentID                 string `json:"-"`                            // ISA segment ID
	AuthInfoQualifier         string `json:"authorizationQualifier"`       // ISA01
	AuthInfo                  string `json:"authorizationInformation"`     // ISA02
	SecurityInfoQualifier     string `json:"securityInformationQualifier"` // ISA03
	SecurityInfo              string `json:"securityInformation"`          // ISA04
	SenderIDQualifier         string `json:"senderIdQualifier"`            // ISA05
	SenderID                  string `json:"senderId"`                     // ISA06
	ReceiverIDQualifier       string `json:"receiverIdQualifier"`          // ISA07
	ReceiverID                string `json:"receiverId"`                   // ISA08
	Date                      string `json:"date"`                         // ISA09
	Time                      string `json:"time"`                         // ISA10
	RepetitionSeparator       string `json:"repetitionSeparator"`          // ISA11
	Version                   string `json:"controlVersionNumber"`         // ISA12
	ControlNumber             string `json:"controlNumber"`                // ISA13
	AckRequested              string `json:"acknowledgmentRequested"`      // ISA14
	UsageIndicator            string `json:"usageIndicator"`               // ISA15
	ComponentElementSeparator string `json:"componentElementSeparator"`    // ISA16
}

type InterchangeTrailer struct {
	SegmentID                        string `json:"-"`                    // IEA segment ID
	NumberOfIncludedFunctionalGroups string `json:"functionalGroupCount"` // IEA01
	ControlNumber                    string `json:"controlNumber"`        // IEA02
}

type FunctionalGroupHeader struct {
	SegmentID                            string `json:"-"`                        // GS segment ID
	IdentifierCode                       string `json:"functionalIdentifierCode"` // GS01
	ApplicationSenderCode                string `json:"applicationSenderCode"`    // GS02
	ApplicationReceiverCode              string `json:"applicationReceiverCode"`  // GS03
	Date                                 string `json:"date"`                     // GS04
	Time                                 string `json:"time"`                     // GS05
	ControlNumber                        string `json:"controlNumber"`            // GS06
	ResponsibleAgencyCode                string `json:"responsibleAgencyCode"`    // GS07
	VersionReleaseIndustryIdentifierCode string `json:"versionCode"`              // GS08
}

type FunctionalGroupTrailer struct {
	SegmentID                       string `json:"-"' `                  // GE segment ID
	NumberOfTransactionSetsIncluded string `json:"transactionSetCount" ` // GE01
	ControlNumber                   string `json:"controlNumber"`        // GE02
}

// Header returns the ISA segment as an InterchangeHeader struct, or
// nil if the message has no ISA segment
func (r *RawMessage) Header() *InterchangeHeader {
	if len(r.segments) == 0 {
		return nil
	}
	seg := r.segments[0]
	if len(seg) == 0 || seg[0] != isaSegmentId {
		return nil
	}
	isaSegment := make([]string, isaElementCount, isaElementCount)
	copy(isaSegment, seg)
	return &InterchangeHeader{
		isaSegment[isaIndexSegmentId],
		isaSegment[isaIndexAuthInfoQualifier],
		isaSegment[isaIndexAuthInfo],
		isaSegment[isaIndexSecurityInfoQualifier],
		isaSegment[isaIndexSecurityInfo],
		isaSegment[isaIndexSenderIdQualifier],
		isaSegment[isaIndexSenderId],
		isaSegment[isaIndexReceiverIdQualifier],
		isaSegment[isaIndexReceiverId],
		isaSegment[isaIndexDate],
		isaSegment[isaIndexTime],
		isaSegment[isaIndexRepetitionSeparator],
		isaSegment[isaIndexVersion],
		isaSegment[isaIndexControlNumber],
		isaSegment[isaIndexAckRequested],
		isaSegment[isaIndexUsageIndicator],
		isaSegment[isaIndexComponentElementSeparator],
	}
}

// Trailer returns the IEA segment as an InterchangeTrailer struct
func (r *RawMessage) Trailer() *InterchangeTrailer {
	if len(r.segments) == 0 {
		return nil
	}
	seg := r.segments[len(r.segments)-1]
	if len(seg) == 0 || seg[0] != ieaSegmentId {
		return nil
	}
	ieaSegment := make([]string, 3, 3)
	copy(ieaSegment, seg)
	return &InterchangeTrailer{
		ieaSegment[0],
		ieaSegment[1],
		ieaSegment[2],
	}
}

// textCleanup removes any characters that are not in the extended character
// set, or the segment terminator, from the given string
func textCleanup(msg string, segmentTerminator rune) string {
	msg = strings.TrimSpace(msg)

	var keepChars []string

	for _, c := range strings.Split(extendedCharacterSet, "") {
		keepChars = append(keepChars, c)
	}
	switch segmentTerminator {
	case '\n':
		keepChars = append(keepChars, "\n")
	case '\r':
		keepChars = append(keepChars, "\r")
	case '\t':
		keepChars = append(keepChars, "\t")
	}

	msgChars := strings.Split(msg, "")
	var b strings.Builder
	for _, c := range msgChars {
		if sliceContains(keepChars, c) {
			b.WriteString(c)
		}
	}
	return b.String()
}

// Read parses the given byte slice into a RawMessage, using the default
// Reader with default X12TransactionSetSpec instances.
func Read(data []byte) (*RawMessage, error) {
	return _defaultReader.Read(data)
}

// Reader is used to read X12 messages into RawMessage instances.
type Reader struct {
	transactionSpecs   []*X12TransactionSetSpec
	transactionSpecsMu sync.RWMutex
}

func (r *Reader) TransactionSpecs() []*X12TransactionSetSpec {
	r.transactionSpecsMu.RLock()
	defer r.transactionSpecsMu.RUnlock()
	return r.transactionSpecs
}

// Read parses the given byte slice into a RawMessage, using the given Reader
// instance.
func (r *Reader) Read(data []byte) (rawMessage *RawMessage, err error) {
	if !utf8.Valid(data) {
		return rawMessage, fmt.Errorf("message text is not valid UTF-8")
	}

	readErrors := []error{}

	var addError = func(err error) {
		if err != nil {
			readErrors = append(readErrors, err)
		}
	}

	text := bytes.TrimLeftFunc(data, unicode.IsSpace)
	runes := bytes.Runes(text)
	if len(runes) < isaByteCount {
		addError(
			fmt.Errorf(
				"%w: message too short to accommodate ISA segment (expected at least %d bytes, got %d)",
				ParseError,
				isaByteCount,
				len(runes),
			),
		)
		return rawMessage, errors.Join(readErrors...)
	}
	rawMessage = &RawMessage{reader: r}

	isaLine := runes[:isaByteCount]
	rawMessage.SegmentTerminator = isaLine[isaByteCount-1]
	rawMessage.ElementSeparator = isaLine[isaElementSeparatorIndex]

	if rawMessage.SegmentTerminator == rawMessage.ElementSeparator {
		addError(
			fmt.Errorf(
				"%w: %w: segment terminator %q cannot be the same as the element separator",
				ParseError, ErrInvalidISA, rawMessage.SegmentTerminator,
			),
		)
		return rawMessage, errors.Join(readErrors...)
	}

	msgText := textCleanup(string(runes), rawMessage.SegmentTerminator)
	rawMessage.text = msgText
	lines := strings.Split(msgText, string(rawMessage.SegmentTerminator))
	rawMessage.segments = make([]RawSegment, 0, len(lines))
	if len(lines) == 0 {
		addError(fmt.Errorf("%w: no segments found", ParseError))
		return rawMessage, errors.Join(readErrors...)
	}

	headerLine := make([]string, isaElementCount, isaElementCount)
	headerElems := strings.Split(lines[0], string(rawMessage.ElementSeparator))

	if len(headerElems) < isaElementCount {
		addError(
			fmt.Errorf(
				"%w: %w: ISA segment too short (expected %d elements, got %d)",
				ParseError,
				ErrInvalidISA,
				isaElementCount,
				len(headerElems),
			),
		)
	}
	copy(headerLine, headerElems)
	rawMessage.segments = append(rawMessage.segments, headerLine)

	componentSeparator := headerLine[isaIndexComponentElementSeparator]
	if len(componentSeparator) != 1 {
		addError(
			fmt.Errorf(
				"%w: %w: invalid component separator '%s' (expected a single character)",
				ParseError,
				ErrInvalidISA,
				componentSeparator,
			),
		)
		return rawMessage, errors.Join(readErrors...)
	}
	rawMessage.ComponentSeparator = rune(componentSeparator[0])

	repetitionSeparator := headerLine[isaIndexRepetitionSeparator]
	if len(repetitionSeparator) != 1 {
		addError(
			fmt.Errorf(
				"%w: %w: invalid repetition separator '%s' (expected a single character)",
				ParseError, ErrInvalidISA, repetitionSeparator,
			),
		)
		return rawMessage, errors.Join(readErrors...)
	}
	rawMessage.RepetitionSeparator = rune(repetitionSeparator[0])

	uq := uniqueElements(
		[]string{
			string(rawMessage.RepetitionSeparator),
			string(rawMessage.ComponentSeparator),
			string(rawMessage.ElementSeparator),
			string(rawMessage.SegmentTerminator),
		},
	)
	if len(uq) != 4 {
		addError(
			fmt.Errorf(
				"%w: separators must be unique (got %v)",
				ParseError, uq,
			),
		)
	}

	for i := 1; i < len(lines); i++ {
		line := lines[i]
		if strings.TrimSpace(line) == "" {
			continue
		}
		seg := strings.Split(line, string(rawMessage.ElementSeparator))
		if len(seg) == 0 {
			addError(
				fmt.Errorf(
					"%w: segment %d is empty",
					ParseError, i,
				),
			)
			return rawMessage, errors.Join(readErrors...)
		}
		rawMessage.segments = append(rawMessage.segments, seg)
	}
	return rawMessage, errors.Join(readErrors...)
}

// NewReader creates a new Reader with the given transaction set specs. The
// specs will be validated, and used to transform ST/SE-bounded transaction
// sets if RawMessage.Message() is called on the result of any
// Reader.Read() call.
func NewReader(transactionSetSpecs ...*X12TransactionSetSpec) (*Reader, error) {
	addSpecs := []*X12TransactionSetSpec{}
	specErrors := []error{}
	for _, txnSpec := range transactionSetSpecs {
		if err := txnSpec.Validate(); err != nil {
			specErrors = append(
				specErrors,
				fmt.Errorf("%s: %w", txnSpec.Key, err),
			)
			continue
		}
		addSpecs = append(addSpecs, txnSpec)
	}
	return &Reader{transactionSpecs: addSpecs}, errors.Join(specErrors...)
}

func init() {
	_defaultReader.transactionSpecsMu.Lock()
	defer _defaultReader.transactionSpecsMu.Unlock()
	for _, txnSpec := range defaultTransactionSpecs {
		if err := txnSpec.Validate(); err != nil {
			panic(
				fmt.Sprintf(
					"unable to validate initial spec %s: %s",
					txnSpec.Key,
					err.Error(),
				),
			)
		}
		_defaultReader.transactionSpecs = append(
			_defaultReader.transactionSpecs,
			txnSpec,
		)
	}
}
