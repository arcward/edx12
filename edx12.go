package edx12

import (
	"container/list"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	isaDefaultVersion = "05010"
)

var (
	ErrRequiredElementMissing     = errors.New("missing required element")
	ErrRequiredCompositeMissing   = errors.New("missing required composite")
	ErrInvalidElementSpecType     = errors.New("unexpected type for element definition")
	ErrRequiredSegmentMissing     = errors.New("missing required segment")
	ErrElementHasInvalidValue     = errors.New("element has invalid value")
	ErrElementHasBadLength        = errors.New("bad element value length")
	ErrRequiredLoopMissing        = errors.New("missing required loop")
	ErrUnexpectedPayloadKey       = errors.New("unexpected key in payload")
	SegmentConditionErr           = errors.New("segment condition validation failed")
	ErrTooFewLoops                = errors.New("number of loops is less than the minimum allowed")
	ErrTooManyLoops               = errors.New("number of loops is greater than the maximum allowed")
	ErrTooFewSegments             = errors.New("number of segments is less than the minimum allowed")
	ErrTooManySegments            = errors.New("number of segments is greater than the maximum allowed")
	ErrInvalidISA                 = errors.New("invalid ISA segment")
	ErrInvalidTransactionEnvelope = errors.New("invalid ST/SE transaction set envelope")
	ErrInvalidGroupEnvelope       = errors.New("invalid GS/GE transaction set envelope")
	ErrInvalidInterchangeEnvelope = errors.New("invalid ISA/IEA interchange envelope")
	ErrMissingSpec                = errors.New("X12Node does not have a spec")
	ErrNotUsed                    = errors.New("X12Node spec indicates NOT_USED, but node has a value")
)

func newError(node *X12Node, err error) error {
	return &NodeError{
		Node: node,
		Err:  err,
	}
}

type NodeError struct {
	Node *X12Node
	Err  error
}

func (e *NodeError) Error() string {
	var b strings.Builder

	if nodeType := e.Node.Type.String(); nodeType != "" {
		_, _ = fmt.Fprintf(&b, "type: '%s' ", nodeType)
	}
	if nodeName := e.Node.Name; nodeName != "" {
		_, _ = fmt.Fprintf(&b, "name: '%s' ", nodeName)
	}
	if nodePath := e.Node.Path; nodePath != "" {
		_, _ = fmt.Fprintf(&b, "path: '%s' ", nodePath)
	}
	bs := strings.TrimSpace(b.String())
	if bs == "" {
		return e.Err.Error()
	}
	bs = fmt.Sprintf("[%s]: %s", bs, e.Err)
	return bs
}

func (e *NodeError) Unwrap() error {
	return e.Err
}

func newSpecErr(e error, spec *X12Spec) error {
	return &SpecErr{
		Spec: spec,
		Err:  e,
	}
}

type SpecErr struct {
	Spec *X12Spec
	Err  error
}

func (e *SpecErr) Error() string {
	var b strings.Builder

	if specName := e.Spec.Name; specName != "" {
		_, _ = fmt.Fprintf(&b, "name: %s ", specName)
	}
	if specPath := e.Spec.Path; specPath != "" {
		_, _ = fmt.Fprintf(&b, "path: %s ", specPath)
	}
	if specDesc := e.Spec.Description; specDesc != "" {
		_, _ = fmt.Fprintf(&b, "description: '%s' ", specDesc)
	}

	bs := strings.TrimSpace(b.String())
	if bs == "" {
		return e.Err.Error()
	}
	bs = fmt.Sprintf("[%s]: %s", bs, e.Err.Error())
	return bs
}

func (e *SpecErr) Unwrap() error {
	return e.Err
}

type X12Node struct {
	Type       NodeType
	Name       string
	Path       string
	Occurrence int
	Parent     *X12Node
	Children   []*X12Node
	Spec       *X12Spec
	Value      []string
	sync.Mutex
}

func (n *X12Node) Elements() []*X12Node {
	elementNodes := []*X12Node{}
	for i := 0; i < len(n.Children); i++ {
		child := n.Children[i]
		switch child.Type {
		case ElementNode:
			elementNodes = append(elementNodes, child)
		default:
			for j := 0; j < len(child.Children); j++ {
				elementNodes = append(
					elementNodes,
					child.Children[j].Elements()...,
				)
			}
		}
	}
	return elementNodes
}

func (n *X12Node) Validate() error {
	if n.Spec == nil {
		return newError(n, ErrMissingSpec)
	}
	validationErrors := []error{}

	if n.Spec.NotUsed() && n.Length() > 0 {
		validationErrors = append(validationErrors, newError(n, ErrNotUsed))
	} else if n.Spec.Required() && n.Length() == 0 {
		switch n.Spec.Type {
		case ElementSpec:
			validationErrors = append(
				validationErrors,
				newError(n, ErrRequiredElementMissing),
			)
		case CompositeSpec:
			validationErrors = append(
				validationErrors,
				newError(n, ErrRequiredCompositeMissing),
			)
		case SegmentSpec:
			validationErrors = append(
				validationErrors,
				newError(n, ErrRequiredSegmentMissing),
			)
		case LoopSpec:
			validationErrors = append(
				validationErrors,
				newError(n, ErrRequiredLoopMissing),
			)
		default:
			validationErrors = append(
				validationErrors,
				newError(n, errors.New("missing required node")),
			)
		}
	}

	switch n.Type {
	case SegmentNode:
		validationErrors = append(validationErrors, validateSegmentNode(n))
	case ElementNode, RepeatElementNode:
		validationErrors = append(validationErrors, validateElement(n))
	case CompositeNode:
		validationErrors = append(validationErrors, validateComposite(n))
	case LoopNode:
		validationErrors = append(validationErrors, validateLoop(n))
	}
	return errors.Join(validationErrors...)
}

func (n *X12Node) ChildTypes() []NodeType {
	return nodeChildTypes[n.Type]
}

func (n *X12Node) Length() int {
	switch n.Type {
	case ElementNode:
		val := n.Value
		if len(val) == 0 {
			return 0
		}
		return len(val[0])
	case RepeatElementNode:
		return len(n.Value)
	default:
		return len(n.Children)
	}
}

func (n *X12Node) AllowedSpecTypes() []SpecType {
	switch n.Type {
	case MessageNode:
		return []SpecType{}
	case GroupNode:
		return []SpecType{}
	case TransactionNode:
		return []SpecType{TransactionSetSpec}
	case LoopNode:
		return []SpecType{LoopSpec}
	case SegmentNode:
		return []SpecType{SegmentSpec}
	case CompositeNode:
		return []SpecType{CompositeSpec}
	case RepeatElementNode:
		return []SpecType{ElementSpec}
	case ElementNode:
		return []SpecType{ElementSpec}
	default:
		return []SpecType{}
	}
}

func (n *X12Node) SetSpec(s *X12Spec) error {
	var specErrors []error
	n.Name = s.Name
	if n.Type == UnknownNode {
		switch s.Type {
		case LoopSpec:
			n.Type = LoopNode
		case SegmentSpec:
			n.Type = SegmentNode
		case CompositeSpec:
			n.Type = CompositeNode
		case ElementSpec:
			if s.IsRepeatable() {
				n.Type = RepeatElementNode
			} else {
				n.Type = ElementNode
			}
		case TransactionSetSpec:
			n.Type = TransactionNode
		}
	}

	if !sliceContains(s.allowedNodeTypes(), n.Type) {
		specErrors = append(
			specErrors,
			fmt.Errorf(
				"spec '%s' not allowed for node type '%s'",
				s.Type,
				n.Type,
			),
		)
		return errors.Join(specErrors...)
	}

	switch s.Type {
	case ElementSpec:
		if s.IsRepeatable() {
			n.Type = RepeatElementNode
			if len(n.Value) == 0 {
				// if a value hasn't already been set, assign the value
				// based on the defined capacity of the element
				if err := s.validateRepeat(); err != nil {
					return err
				}
				rmin := s.RepeatMin
				if rmin == 0 {
					rmin = 1
				}

				if s.RepeatMax > 0 {
					n.Value = make([]string, rmin, s.RepeatMax)
				} else {
					n.Value = make([]string, rmin)
				}
			}
		} else {
			n.Type = ElementNode
			// Ignore values of length 1 to avoid overwriting an existing value
			if len(n.Value) > 1 {
				specErrors = append(
					specErrors,
					fmt.Errorf(
						"element '%s' is not repeatable but has %d values",
						n.Name,
						len(n.Value),
					),
				)
			} else if len(n.Value) == 0 {
				n.Value = make([]string, 1, 1)
				//log.Printf("set %#v value to: %#v", x, n.Value)
			}
		}

	case CompositeSpec:
		if n.Type != CompositeNode {
			n.Type = CompositeNode
			values := n.Value
			if len(values) > 0 {
				n.Children = make([]*X12Node, len(values))
			} else {
				n.Children = []*X12Node{}
			}
			for ci := 0; ci < len(values); ci++ {
				n.Children[ci] = &X12Node{
					Type:   ElementNode,
					Name:   fmt.Sprintf("%s%02d", n.Name, ci),
					Value:  []string{values[ci]},
					Parent: n,
				}
			}
			n.Value = []string{}
		}
		specStructure := s.Structure
		for i := 0; i < len(s.Structure); i++ {
			eSpec := specStructure[i]
			if eSpec.Type != ElementSpec {
				return newSpecErr(
					fmt.Errorf(
						"composite spec contains non-element spec '%s'",
						eSpec.Name,
					), s,
				)
			}

			var newElemNode *X12Node
			var newElemNodeErr error
			if i < len(n.Children) {

				newElemNode, newElemNodeErr = NewNode(
					ElementNode,
					"",
					n.Children[i].Value...,
				)
				if newElemNodeErr != nil {
					specErrors = append(
						specErrors,
						newSpecErr(newElemNodeErr, eSpec),
					)
				}
				if err := newElemNode.SetSpec(eSpec); err != nil {
					specErrors = append(specErrors, err)
				}
				n.Children[i] = newElemNode
				newElemNode.Parent = n
			} else {
				newElemNode, newElemNodeErr = eSpec.defaultObjValue()
				if newElemNodeErr != nil {
					specErrors = append(
						specErrors,
						newSpecErr(newElemNodeErr, eSpec),
					)
				}
				n.Children = append(n.Children, newElemNode)
				newElemNode.Parent = n
			}
		}
		//se := x.setPath()
		//if se != nil {
		//	specErrors = append(specErrors, newError(x, se))
		//}
	case SegmentSpec:
		if n.Type != SegmentNode {
			return fmt.Errorf(
				"cannot set segment spec on node of type %s",
				n.Type,
			)
		}

		if len(n.Children) == 0 {
			n.Children = make([]*X12Node, len(s.Structure)+1)
			idNode, err := NewNode(ElementNode, s.Name, s.Name)
			if err != nil {
				specErrors = append(specErrors, newSpecErr(err, s))
			}
			idNode.Parent = n
			n.Children[0] = idNode
		}
		elementInd := 0
		specStructure := s.Structure
		for i := 0; i < len(specStructure); i++ {
			childSpec := specStructure[i]
			elementInd = i + 1
			if elementInd < len(n.Children) {
				childNode := n.Children[elementInd]
				if childNode == nil {
					childNode = &X12Node{}
				}
				err := childNode.SetSpec(childSpec)
				specErrors = append(specErrors, err)
				n.Children[elementInd] = childNode
			} else {
				//childSpec := xSpec.structure[i]
				switch childSpec.Type {
				case ElementSpec, CompositeSpec:
					childNode := &X12Node{}
					if err := childNode.SetSpec(childSpec); err != nil {
						specErrors = append(specErrors, err)
					}
					n.Children = append(n.Children, childNode)
				default:
					return newSpecErr(
						fmt.Errorf(
							"expected %s or %s, got %s",
							elementTypeName,
							compositeTypeName,
							childSpec.Type,
						), s,
					)
				}
			}
		}
	case LoopSpec:
		specStructure := s.Structure
		n.Spec = s
		for i := 1; i < len(specStructure); i++ {
			if i < len(n.Children) {
				childNode := n.Children[i]
				err := childNode.SetSpec(specStructure[i-1])
				specErrors = append(specErrors, err)
				n.Children[i] = childNode
			}
		}
		return errors.Join(specErrors...)
	case TransactionSetSpec:
		specStructure := s.Structure
		for i := 1; i < len(specStructure); i++ {
			if i < len(n.Children) {
				childNode := n.Children[i]
				err := childNode.SetSpec(specStructure[i-1])
				specErrors = append(specErrors, err)
				n.Children[i] = childNode
			}
		}
	default:
		return fmt.Errorf("unknown spec type '%s'", s.Type)
	}

	n.Spec = s
	return errors.Join(specErrors...)
}

func (n *X12Node) SetValue(value ...string) error {
	var errs []error
	if n.Type != ElementNode && n.Type != RepeatElementNode && n.Type != CompositeNode {
		errs = append(
			errs,
			fmt.Errorf("cannot set value for node type '%s'", n.Type),
		)
		return errors.Join(errs...)
	}
	switch n.Type {
	case ElementNode:
		if len(value) > 1 {
			errs = append(
				errs,
				fmt.Errorf(
					"cannot set multiple values for element '%s'",
					n.Name,
				),
			)
			return errors.Join(errs...)
		}
		elemArr := make([]string, 1, 1)
		copy(elemArr, value)
		n.Value = elemArr
	case RepeatElementNode:
		if len(value) > 0 {
			n.Value = removeTrailingEmptyElements(value)
		} else {
			n.Value = []string{}
		}

	case CompositeNode:
		var maxLen int
		elementCt := len(value)
		if n.Spec != nil {
			maxLen = len(n.Spec.Structure)
			if elementCt > maxLen {
				errs = append(
					errs, newError(
						n, fmt.Errorf(
							"composite node has %d children, but %d values were given - %d have been ignored",
							maxLen, len(value), elementCt-maxLen,
						),
					),
				)
				return errors.Join(errs...)
			}
		}
		if maxLen == 0 {
			maxLen = elementCt
		}
		values := make([]string, maxLen, maxLen)
		copy(values, value)
		for i := 0; i < maxLen; i++ {
			if i >= len(n.Children) {
				newChildNode, err := NewNode(
					ElementNode,
					fmt.Sprintf("%s%02d", n.Name, i),
				)
				if err != nil {
					errs = append(errs, newError(n, err))
					return errors.Join(errs...)
				}
				if err = n.Append(newChildNode); err != nil {
					errs = append(errs, newError(n, err))
					return errors.Join(errs...)
				}
				continue
			}
			if e := n.Children[i].SetValue(values[i]); e != nil {
				errs = append(errs, newError(n.Children[i], e))
			}
		}

	}
	return errors.Join(errs...)
}

func NewNode(nodeType NodeType, name string, values ...string) (
	node *X12Node,
	err error,
) {
	node = &X12Node{Type: nodeType, Name: name}
	switch nodeType {
	case ElementNode:
		if len(values) > 1 {
			return node, errors.New("cannot set >1 value on ElementNode, only RepeatElementNode")
		}
		node.Value = make([]string, 1, 1)
		copy(node.Value, values)
	case RepeatElementNode:
		if len(values) > 0 {
			node.Value = make([]string, len(values))
			copy(node.Value, values)
		} else {
			node.Value = make([]string, 1)
		}
	case SegmentNode:
		segmentID, _ := NewNode(ElementNode, name, name)
		node.Children = []*X12Node{segmentID}
		segmentID.Parent = node
		idSpec := &X12Spec{
			Type:        ElementSpec,
			Usage:       Required,
			RepeatMin:   1,
			RepeatMax:   1,
			ValidCodes:  []string{name},
			Description: "Segment ID",
			DefaultVal:  name,
		}
		if err := segmentID.SetSpec(idSpec); err != nil {
			return node, err
		}

		if len(values) > 0 {
			if values[0] != name {
				err = fmt.Errorf(
					"segment name '%s' does not match value '%s'",
					name,
					values[0],
				)
				return node, err
			}
			for i := 1; i < len(values); i++ {
				elementID := fmt.Sprintf("%s%02d", name, i+1)
				childNode, err := NewNode(ElementNode, elementID, values[i])
				if err != nil {
					return node, err
				}
				node.Children = append(
					node.Children,
					childNode,
				)
			}
		}
	}
	return node, err
}

func (n *X12Node) Append(node *X12Node) error {
	var nodeErrors []error

	if !sliceContains(n.ChildTypes(), node.Type) {
		nodeErrors = append(
			nodeErrors,
			fmt.Errorf(
				"node type '%s' not allowed as child of '%s'",
				node.Type,
				n.Type,
			),
		)
		return errors.Join(nodeErrors...)
	}
	if node.Name == "" && node.Spec == nil {
		switch n.Type {
		case SegmentNode:
			if len(n.Children) == 1 {
				node.Name = n.Name
			} else {
				node.Name = fmt.Sprintf(
					"%s%02d",
					n.Name,
					len(n.Children),
				)
			}
		case CompositeNode:
			node.Name = fmt.Sprintf(
				"%s%02d",
				n.Name,
				len(n.Children),
			)
		}
	}

	if node.Name == "" && node.Spec != nil {
		node.Name = node.Spec.Name
	}
	node.Parent = n
	n.Children = append(n.Children, node)
	return nil

}

type ISAHeader struct {
	AuthorizationQualifier       string    `validate:"oneof=00 03" json:"authorization_qualifier"`
	AuthorizationInformation     string    `json:"authorization_information"`
	SecurityInformationQualifier string    ` validate:"oneof=00 01" json:"security_information_qualifier"`
	SecurityInformation          string    `json:"security_information"`
	SenderIdQualifier            string    `validate:"oneof=01 14 20 27 28 29 30 33 ZZ" json:"sender_id_qualifier"`
	SenderId                     string    `json:"sender_id"`
	ReceiverIdQualifier          string    `validate:"oneof=01 14 20 27 28 29 30 33 ZZ" json:"receiver_id_qualifier"`
	ReceiverId                   string    `json:"receiver_id"`
	DateTime                     time.Time `json:"date_time"`
	RepetitionSeparator          rune      `validate:"required" json:"repetition_separator"`
	ControlVersionNumber         string    `json:"control_version_number"`
	ControlNumber                string    `json:"control_number"`
	AcknowledgmentRequested      string    `validate:"oneof=0 1" json:"acknowledgment_requested"`
	UsageIndicator               string    `validate:"oneof=P T" json:"usage_indicator"`
	ComponentElementSeparator    rune      `validate:"required" json:"component_element_separator"`
	*X12Node
}

func (i ISAHeader) String() string {
	s, err := json.Marshal(i)
	if err != nil {
		log.Printf("err: %v", err)
		return fmt.Sprintf("%#v", i)
	}
	return string(s)
}

//
//func NewISAHeader() ISAHeader {
//	return ISAHeader{
//		AuthorizationQualifier:       "00",
//		AuthorizationInformation:     "",
//		SecurityInformationQualifier: "00",
//		SecurityInformation:          "",
//		SenderIdQualifier:            "01",
//		SenderId:                     "",
//		ReceiverIdQualifier:          "01",
//		ReceiverId:                   "",
//		DateTime:                     time.Now(),
//		ControlVersionNumber:         isaDefaultVersion,
//		ControlNumber:                "",
//		RepetitionSeparator:          '^',
//		ComponentElementSeparator:    ':',
//		AcknowledgmentRequested:      "0",
//		UsageIndicator:               "P",
//	}
//}

type Message struct {
	Header            *X12Node
	Trailer           *X12Node
	SegmentTerminator rune `json:"segment_terminator"`
	ElementSeparator  rune `json:"element_separator"`
	functionalGroups  []*FunctionalGroupNode
	*X12Node
}

func (n *Message) TransactionSets() []*TransactionSetNode {
	transactionSets := []*TransactionSetNode{}
	for _, fg := range n.functionalGroups {
		for _, t := range fg.Transactions {
			transactionSets = append(transactionSets, t)
		}
	}
	return transactionSets
}

func (n *Message) numberOfFunctionalGroups() (int, error) {
	elem := n.Trailer.Children[ieaIndexFunctionalGroupCount]
	ct := elem.Value[0]

	return strconv.Atoi(ct)
}

func (n *Message) MarshalText() ([]byte, error) {
	return []byte(n.Format()), nil
}

func (n *Message) Validate() error {
	validationErrors := []error{}
	validationErrors = append(validationErrors, n.Header.Validate())
	validationErrors = append(validationErrors, n.Trailer.Validate())

	isaControlNum := n.ControlNumber()
	ieaControlNum := n.IEAControlNumber()
	if isaControlNum != ieaControlNum {
		validationErrors = append(
			validationErrors,
			fmt.Errorf(
				"%w: ISA control number %s does not match IEA control number %s",
				ErrInvalidInterchangeEnvelope,
				isaControlNum,
				ieaControlNum,
			),
		)
	}

	groupCount := len(n.functionalGroups)
	trailerCt, e := n.numberOfFunctionalGroups()
	validationErrors = append(validationErrors, e)
	if e == nil && trailerCt != groupCount {
		validationErrors = append(
			validationErrors,
			fmt.Errorf(
				"%w: expected %d functional groups, got %d",
				ErrInvalidInterchangeEnvelope,
				trailerCt,
				groupCount,
			),
		)
	}

	for g := 0; g < len(n.functionalGroups); g++ {
		validationErrors = append(
			validationErrors,
			n.functionalGroups[g].Validate(),
		)
	}
	return errors.Join(validationErrors...)
}

// ControlNumber returns `ISA13`
func (n *Message) ControlNumber() string {
	return n.headerElementValue(isaIndexControlNumber)
}

// SetControlNumber sets the control number for the message to the
// given value. This updates both ISA and IEA segments.
func (n *Message) SetControlNumber(value string) error {
	targetLength := isaLenControlNumber
	if len(value) > targetLength {
		return fmt.Errorf(
			"string too long (length: %d, max: %d)",
			len(value),
			targetLength,
		)
	}
	v := strings.TrimSpace(value)
	if v == "" {
		v = "0"
	}

	vi, e := strconv.Atoi(v)
	if e != nil {
		return e
	}
	paddedVal := fmt.Sprintf("%0*s", targetLength, strconv.Itoa(vi))
	header := n.header()
	if header != nil {
		if err := header.Children[isaIndexControlNumber].SetValue(paddedVal); err != nil {
			return err
		}
	}

	trailer := n.Trailer
	if trailer != nil {
		return trailer.Children[ieaIndexControlNumber].SetValue(paddedVal)
	}
	return nil
}

// IEAControlNumber returns `IEA02`. The value can
// be set via SetControlNumber.
func (n *Message) IEAControlNumber() string {
	return n.Trailer.Children[ieaIndexControlNumber].Value[0]
}

func (n *Message) Format() string {
	return n.X12Node.Format(
		string(n.SegmentTerminator),
		string(n.ElementSeparator),
		string(n.RepetitionSeparator()),
		string(n.ComponentElementSeparator()),
	)
}

// header returns the ISA segment
func (n *Message) header() *X12Node {
	if len(n.Children) == 0 {
		return nil
	}
	return n.Children[0]
}

// headerElementValue gets the value of the ISA header segment at
// the given index
func (n *Message) headerElementValue(index int) string {
	return n.header().Children[index].Value[0]
}

// ComponentElementSeparator returns `ISA16`
func (n *Message) ComponentElementSeparator() rune {
	var r rune
	for _, c := range n.headerElementValue(isaIndexComponentElementSeparator) {
		r = c
		break
	}
	return r
}

func charRemover(char rune) walkNodeFunc {
	return func(n *X12Node) error {
		var b strings.Builder
		for i := 0; i < len(n.Value); i++ {
			val := n.Value[i]
			if !strings.ContainsRune(val, char) {
				continue
			}
			for _, r := range val {
				if r == char {
					b.WriteString("")
				} else {
					b.WriteRune(r)
				}
			}
			n.Value[i] = b.String()
			b.Reset()
		}
		return nil
	}
}

// RepetitionSeparator returns `ISA11`
func (n *Message) RepetitionSeparator() rune {
	var r rune
	for _, c := range n.headerElementValue(isaIndexRepetitionSeparator) {
		r = c
		break
	}
	return r
}

func (n *X12Node) removeChar(char rune) error {
	return n.Walk(charRemover(char))
}

// SetRepetitionSeparator sets the given rune as the repetition delimiter.
// When set, this value is stripped from all element values in the message.
func (n *Message) SetRepetitionSeparator(value rune) error {
	if e := n.removeChar(value); e != nil {
		return e
	}
	return n.Header.Children[isaIndexRepetitionSeparator].SetValue(string(value))
}

// SetComponentElementSeparator sets `ISA16`. When set, this value is
// stripped from all element values in the message.
func (n *Message) SetComponentElementSeparator(value rune) error {
	if e := n.removeChar(value); e != nil {
		return e
	}
	return n.Header.Children[isaIndexComponentElementSeparator].SetValue(string(value))
}

type FunctionalGroupNode struct {
	Header                *X12Node
	Trailer               *X12Node
	IdentifierCode        string // GS01
	ReceiverCode          string // GS02
	SenderCode            string // GS03
	Date                  string // GS04
	Time                  string // GS05
	ControlNumber         string // GS06
	ResponsibleAgencyCode string // GS07
	Version               string // GS08
	Transactions          []*TransactionSetNode
	Message               *Message
	*X12Node
}

func populateCompositeNode(node *X12Node, values ...string) error {
	var errs []error
	elementDiff := len(values) - len(node.Children)
	if elementDiff > 0 {
		for i := 0; i < elementDiff; i++ {
			elemName := ""
			if node.Name != "" {
				elemName = fmt.Sprintf("%s-%02d", node.Name, i+1)
			}
			eNode, e := NewNode(ElementNode, elemName)
			if e != nil {
				return e
			}
			node.Children = append(node.Children, eNode)
		}
	}
	elementCt := len(node.Children)

	for i := 0; i < elementCt; i++ {
		if i > len(values) {
			if e := node.Children[i].SetValue(""); e != nil {
				errs = append(errs, e)
			}
			continue
		}
		if e := node.Children[i].SetValue(values[i]); e != nil {
			errs = append(
				errs,
				fmt.Errorf("error setting value at %d: %w", i, e),
			)
		}
	}
	return errors.Join(errs...)
}

// setIndexValue will set the `value` at the given `index` for the node.
//
// If this is called on a SegmentNode, the index is the index of the
// child ElementNode or RepeatElementNode. For ElementNode, the
// provided value will be set on the first index of ElementNode's value.
// For RepeatElementNode, the provided value will be set on first index,
// and any other indexes will be set to an empty string.
//
// If this is called on a CompositeNode, the index is the index of the
// child ElementNode, and the provided value will be set on the
// first index of ElementNode's value.
//
// It will return an error if called on any NodeType other than SegmentNode,
// CompositeNode, RepeatElementNode or ElementNode.
func (n *X12Node) setIndexValue(index int, value string) (
	node *X12Node,
	err error,
) {
	var nodeErrors []error
	switch n.Type {
	case ElementNode:
		if index != 0 {
			return node, newError(
				n,
				fmt.Errorf("index %d out of bounds", index),
			)
		}
		return nil, n.SetValue(value)
	case RepeatElementNode:
		val := n.Value
		for i := len(val); i < index+1; i++ {
			val = append(val, "")
		}
		val[index] = value
		n.Value = val
		return nil, nil
	case CompositeNode:
		if n.Spec == nil {
			return nil, newError(n, fmt.Errorf("no spec set"))
		}
		node = n.Children[index]
		if e := node.SetValue(value); e != nil {
			nodeErrors = append(nodeErrors, e)
		}
		return node, errors.Join(nodeErrors...)
	case SegmentNode:
		if n.Spec == nil {
			return nil, newError(
				n,
				fmt.Errorf("no spec set"),
			)
		}
		if e := n.Children[index].SetValue(value); e != nil {
			nodeErrors = append(nodeErrors, e)
		}
		return n.Children[index], errors.Join(nodeErrors...)
	default:
		return nil, newError(n, fmt.Errorf("invalid node type %s", n.Type))

	}
}

// groupNodesByName will return a slice of slices of nodes, where each
// slice of nodes is a group of nodes where the first node's
// X12Node.Name is equal to segmentHeaderId, and the last node's
// X12Node.Name is equal to segmentTrailerId, with all nodes in between.
// Use this to separate a list of SegmentNode nodes into groups bounded
// by ISA/IEA, GS/GE, ST/SE, etc.
// An error will be returned if segmentHeaderId is seen more than once
// before segmentTrailerId is seen, or if segmentTrailerId is seen before
// segmentHeaderId.
func groupNodesByName(
	segmentHeaderId string,
	segmentTrailerId string,
	segments []*X12Node,
) ([][]*X12Node, error) {
	var segmentGroups [][]*X12Node
	var lastSeen bool
	var currentGroup []*X12Node

	for _, segment := range segments {
		if segment.Name == segmentHeaderId {
			if len(currentGroup) > 0 {
				return nil, newError(
					segment, fmt.Errorf(
						"found %s segment before %s segment",
						segmentHeaderId,
						segmentTrailerId,
					),
				)
			}
			currentGroup = append(currentGroup, segment)
			lastSeen = true
		} else if segment.Name == segmentTrailerId {
			if len(currentGroup) > 0 {
				currentGroup = append(currentGroup, segment)
				lastSeen = true
				segmentGroups = append(segmentGroups, currentGroup)
				currentGroup = []*X12Node{}
			} else {
				return nil, newError(
					segment, fmt.Errorf(
						"unexpected state- found segment Trailer %v, but not currently in a segment group (already have: %v)",
						segment.Name,
						segmentGroups,
					),
				)
			}
		} else {
			if lastSeen == true && segment.Name != segmentHeaderId && len(currentGroup) == 0 {
				return segmentGroups, nil
			} else if len(currentGroup) > 0 {
				currentGroup = append(currentGroup, segment)
			}
		}
	}
	return segmentGroups, nil
}

func createFunctionalGroupNode(node *X12Node) (
	groupNode *FunctionalGroupNode,
	err error,
) {
	var errs []error
	if node == nil {
		node = &X12Node{
			Type: GroupNode,
			Name: functionalGroupName,
		}
	}
	node.Type = GroupNode
	node.Name = functionalGroupName

	groupNode = &FunctionalGroupNode{X12Node: node}

	gsSpec, err := groupHeaderSpec()
	if err != nil {
		return groupNode, err
	}
	geSpec, err := groupTrailerSpec()
	if err != nil {
		return groupNode, err
	}

	if len(groupNode.Children) >= 2 {
		if groupNode.Children[0] == nil {
			return groupNode, newError(
				groupNode.X12Node,
				fmt.Errorf("no header node found"),
			)
		}
		if groupNode.Children[len(groupNode.Children)-1] == nil {
			return groupNode, newError(
				groupNode.X12Node,
				fmt.Errorf("no trailer node found"),
			)
		}
	}

	if len(node.Children) == 0 {
		headerNode, err := NewNode(SegmentNode, gsSegmentId)
		groupNode.Header = headerNode
		headerNode.Parent = groupNode.X12Node
		if err != nil {
			return groupNode, err
		}

		trailerNode, err := NewNode(SegmentNode, geSegmentId)
		groupNode.Trailer = trailerNode
		trailerNode.Parent = groupNode.X12Node
		if err != nil {
			return groupNode, err
		}

		node.Children = []*X12Node{headerNode, trailerNode}
	} else if len(node.Children) >= 2 {
		groupNode.Header = node.Children[0]
		groupNode.Trailer = node.Children[len(node.Children)-1]
	} else {
		errs = append(
			errs, newError(
				node,
				fmt.Errorf("group node must have at least 2 children"),
			),
		)
	}

	err = groupNode.Header.SetSpec(gsSpec)
	if err != nil {
		return groupNode, err
	}

	err = groupNode.Trailer.SetSpec(geSpec)
	if err != nil {
		return groupNode, err
	}

	gsHeader := groupNode.Children[0]

	for i := 1; i < len(gsHeader.Children); i++ {
		switch i {
		case gsIndexFunctionalIdentifierCode:
			groupNode.IdentifierCode = gsHeader.Children[i].Value[0]
		case gsIndexSenderCode:
			groupNode.SenderCode = gsHeader.Children[i].Value[0]
		case gsIndexReceiverCode:
			groupNode.ReceiverCode = gsHeader.Children[i].Value[0]
		case gsIndexDate:
			groupNode.Date = gsHeader.Children[i].Value[0]
		case gsIndexTime:
			groupNode.Time = gsHeader.Children[i].Value[0]
		case gsIndexControlNumber:
			groupNode.ControlNumber = gsHeader.Children[i].Value[0]
		case gsIndexResponsibleAgencyCode:
			groupNode.ResponsibleAgencyCode = gsHeader.Children[i].Value[0]
		case gsIndexVersion:
			groupNode.Version = gsHeader.Children[i].Value[0]
		}
	}
	for i := 1; i < len(groupNode.Children); i++ {
		childNode := node.Children[i]
		if childNode.Type != TransactionNode {
			continue
		}
		txnNode, err := createTransactionNode(childNode)
		if err != nil {
			errs = append(errs, newError(childNode, err))
		}
		groupNode.Transactions = append(groupNode.Transactions, txnNode)
	}
	return groupNode, errors.Join(errs...)
}

// TransactionSetNode represents a transaction set, bounded by ST/SE segments.
type TransactionSetNode struct {
	TransactionSetCode string `json:"transactionSetCode"` // ST01
	ControlNumber      string `json:"controlNumber"`      // ST02
	VersionCode        string `json:"version"`            // ST03
	Header             *X12Node
	Trailer            *X12Node
	// HierarchicalLevels points to HL segments with their parent/child
	// relationships.
	HierarchicalLevels []*HierarchicalLevel
	TransactionSpec    *X12TransactionSetSpec
	Group              *FunctionalGroupNode
	*X12Node
}

func (txn *TransactionSetNode) MarshalJSON() (data []byte, err error) {
	payload, err := txn.Payload()
	if err != nil {
		return data, err
	}
	payload["transactionSetCode"] = txn.TransactionSetCode
	payload["controlNumber"] = txn.ControlNumber
	payload["version"] = txn.VersionCode
	return json.Marshal(payload)
}

type TransactionIdentifier struct {
	TransactionSetCode string `json:"transaction_set_code"`
	Version            string `json:"version"`
}

//func findTransactionIdentifier(nodes ...*X12Node) (txnID TransactionIdentifier, err error) {
//	var headerNode *X12Node
//
//	for i := 0; i < len(nodes); i++ {
//		n := nodes[i]
//		if n.Name == stSegmentId {
//			headerNode = n
//			break
//		}
//
//		segs := n.findSegments(stSegmentId)
//		if len(segs) > 0 {
//			headerNode = segs[0]
//			break
//		}
//	}
//
//	if headerNode == nil {
//		return txnID, fmt.Errorf("no header node found")
//	}
//
//
//}

// Transform takes a flat list of Segment instances for the current
// TransactionSetNode, and maps them to
// the TransactionSetSpec structure, identifying which segments
// map to which SegmentSpec instances, and which LoopSpec instances those
// segments belong to. A TransactionSet is created, with the hierarchy
// of loop and Segment instances matched to the transaction set.
func (txn *TransactionSetNode) Transform() error {
	if txn.TransactionSpec == nil {
		spec, err := findTransactionSpec(
			txn.TransactionSetCode,
			txn.VersionCode,
		)
		if err != nil {
			return err
		}
		txn.TransactionSpec = spec
		txn.X12Node.Spec = spec.X12Spec
	}
	var validationErrors []error
	segmentQueue := &segmentDeque{segments: list.New()}
	for _, segment := range txn.Segments() {
		segmentQueue.Append(segment)
	}
	segmentCount := segmentQueue.segments.Len()

	loopVisitor := &loopTransformer{message: txn.Group.Message}
	loopVisitor.createdLoop = txn.X12Node
	loopVisitor.LoopSpec = txn.TransactionSpec.X12Spec
	loopVisitor.SegmentQueue = segmentQueue

	for i := 0; i < len(txn.Children); i++ {
		childNode := txn.Children[i]
		childNode.Parent = nil
	}
	txn.Children = []*X12Node{}

	// We set the parent to nil here and re-set it after transforming
	// the loop, to avoid re-setting the paths of the entire tree very
	// time we append a new loop or node. X12Node.Append() will not
	// update the path of a TransactionNode or GroupNode if the parent
	// is nil, so each TransactionSet child hierarchy will still have
	// the correct base path, and the nil parent will prevent the
	// update from moving up the tree and down into all the other
	// transactions
	txnParent := txn.Parent
	txn.Parent = nil
	if e := loopVisitor.cascade(); e != nil {
		return e
	}

	loopVisitor.updateLoop()

	validationErrors = append(validationErrors, loopVisitor.ErrorLog...)
	matchedSegments := txn.Segments()
	unmatchedSegments := []*X12Node{}

	if segmentQueue.Length() > 0 {
		for i := 0; i < segmentQueue.Length(); i++ {
			orphanSegment := segmentQueue.PopLeft()
			unmatchedSegments = append(unmatchedSegments, orphanSegment)
			segmentQueue.Append(orphanSegment)
		}
	}
	if len(unmatchedSegments) > 0 {
		breakingSegment := unmatchedSegments[0]
		breakingSegmentInd := segmentCount - len(unmatchedSegments)
		var remainingSegments []*X12Node
		if len(unmatchedSegments) > 1 {
			remainingSegments = unmatchedSegments[1:]
		}

		segErr := newError(
			breakingSegment,
			fmt.Errorf("unable to match to a SegmentSpec"),
		)
		var sArr []interface{}
		for _, s := range remainingSegments {
			sArr = append(sArr, s.Children)
		}
		remErr := fmt.Errorf("remaining segments: %q", sArr)
		if segmentCount != (len(matchedSegments) + len(unmatchedSegments)) {
			validationErrors = append(
				validationErrors,
				fmt.Errorf("lost a segment somewhere"),
			)
		}
		validationErrors = append(
			validationErrors, fmt.Errorf(
				"unable to match %d (of %d) segments, starting at index %d: %w: %w",
				len(unmatchedSegments),
				segmentCount,
				breakingSegmentInd,
				segErr,
				remErr,
			),
		)
	}
	rootNode := getRootNode(txn.X12Node)
	err := rootNode.setPath()
	validationErrors = append(validationErrors, err)

	err = txn.Validate()
	validationErrors = append(validationErrors, err)
	txn.Parent = txnParent
	if loopVisitor.abort && loopVisitor.abortReason != nil {
		return newError(
			txn.X12Node, fmt.Errorf(
				"%w: %w: %w",
				ErrTransactionAborted,
				loopVisitor.abortReason,
				errors.Join(validationErrors...),
			),
		)
	}
	return errors.Join(validationErrors...)
}

func getRootNode(x *X12Node) *X12Node {
	if x.Parent == nil {
		return x
	}
	return getRootNode(x.Parent)
}

// HierarchicalLevel represents HL segments and their parent/child
// relationships
type HierarchicalLevel struct {
	id          string // HL01
	parentId    string // HL02
	levelCode   string // HL03
	childCode   string // HL04
	Node        *X12Node
	ParentLevel *HierarchicalLevel
	ChildLevels []*HierarchicalLevel
}

// hasChildLevels returns true if HL04 (childCode) indicates there
// should be child HL segments which reference this level's id
func (h *HierarchicalLevel) hasChildLevels() bool {
	if h.childCode == "1" {
		return true
	}
	return false
}

// hasParentLevel returns true if HL02 (parentId) is populated,
// indicating that there should be a correlating HL segment
// with HL01 == parentId
func (h *HierarchicalLevel) hasParentLevel() bool {
	if h.parentId == "" {
		return false
	}
	return true
}

// newHierarchicalLevel creates a HierarchicalLevel from the given X12Node
func newHierarchicalLevel(node *X12Node) (*HierarchicalLevel, error) {
	hlNode := &HierarchicalLevel{Node: node}

	elements := node.Children
	for i := 0; i < len(elements); i++ {
		switch i {
		case hlIndexHierarchicalId:
			hlNode.id = elements[i].Value[0]
		case hlIndexParentId:
			hlNode.parentId = elements[i].Value[0]
		case hlIndexLevelCode:
			hlNode.levelCode = elements[i].Value[0]
		case hlIndexChildCode:
			hlNode.childCode = elements[i].Value[0]
		}
	}
	return hlNode, nil
}

// createHierarchicalLevels creates HierarchicalLevel objects for each
// HL segment in the given transaction set, and returns a slice containing
// all top-level HL segments (those with no parent HL segment).
// HierarchicalLevel objects will also be correlated with their parent
// and child levels, setting ParentLevel and ChildLevels accordingly.
// Returns an error if an HL segment references a parent ID which has
// no corresponding HierarchicalLevel.id, or if childCode indicates
// child levels but no HL segments are found which reference this level's id
// to parentId
func createHierarchicalLevels(transactionSetNode *X12Node) (
	parentLevels []*HierarchicalLevel,
	err error,
) {
	if transactionSetNode.Type != TransactionNode {
		return parentLevels, newError(
			transactionSetNode,
			fmt.Errorf(
				"expected node type to be %s, got %s",
				TransactionNode,
				transactionSetNode.Type,
			),
		)
	}

	hlSegments := transactionSetNode.findSegments(hlSegmentId)
	if len(hlSegments) == 0 {
		return parentLevels, nil
	}
	levels := map[string]*HierarchicalLevel{}

	for i := 0; i < len(hlSegments); i++ {
		hlNode, e := newHierarchicalLevel(hlSegments[i])
		if e != nil {
			return parentLevels, e
		}
		_, exists := levels[hlNode.id]
		if exists {
			return parentLevels, newError(
				hlNode.Node,
				fmt.Errorf(
					"HL segment with id '%s' already exists",
					hlNode.id,
				),
			)
		}
		levels[hlNode.id] = hlNode
		if !hlNode.hasParentLevel() {
			parentLevels = append(parentLevels, hlNode)
		}
	}

	for _, hlNode := range levels {
		if hlNode.hasParentLevel() {
			parentNode, ok := levels[hlNode.parentId]
			if !ok {
				log.Printf(
					"parent node with id '%s' not found",
					hlNode.parentId,
				)
				//return parentLevels, nil
				// TODO: uncomment
				return parentLevels, newError(
					hlNode.Node,
					fmt.Errorf(
						"parent node with id '%s' not found",
						hlNode.parentId,
					),
				)
			}
			hlNode.ParentLevel = parentNode
			parentNode.ChildLevels = append(parentNode.ChildLevels, hlNode)
			if !parentNode.hasChildLevels() {
				log.Printf(
					"parent node with id '%s' does not have child levels",
					hlNode.parentId,
				)
				//return parentLevels, nil
				return parentLevels, newError(
					hlNode.Node,
					fmt.Errorf(
						"parent node with id '%s' does not have child levels",
						hlNode.parentId,
					),
				)
			}
		}
	}
	return parentLevels, nil
}

// Walk walks the X12Node tree, calling walkFunc for each node
func (n *X12Node) Walk(walkFunc walkNodeFunc) error {
	return walkNodes(n, walkFunc)
}

type walkNodeFunc func(x *X12Node) error

func walkNodes(node *X12Node, walkFunc walkNodeFunc) (err error) {
	errs := []error{}

	errs = append(errs, walkFunc(node))

	for _, child := range node.Children {
		e := walkNodes(child, walkFunc)
		if e != nil {
			errs = append(errs, newError(child, e))
		}
	}
	err = errors.Join(errs...)
	return err
}

func (n *X12Node) findSegments(name string) []*X12Node {
	var segments []*X12Node
	n.Walk(
		func(nx *X12Node) error {
			if nx.Type == SegmentNode && nx.Name == name {
				segments = append(segments, nx)
			}
			return nil
		},
	)
	return segments
}

func createTransactionNode(node *X12Node) (
	txnNode *TransactionSetNode,
	err error,
) {
	if node == nil {
		node = &X12Node{}
	}
	node.Type = TransactionNode
	node.Name = transactionSetName

	txnNode = &TransactionSetNode{X12Node: node}

	stSpec, err := transactionHeaderSpec()
	if err != nil {
		return txnNode, err
	}
	seSpec, err := transactionTrailerSpec()
	if err != nil {
		return txnNode, err
	}

	if len(txnNode.Children) >= 2 {
		if txnNode.Children[0] == nil {
			return txnNode, newError(
				txnNode.X12Node,
				fmt.Errorf("first child node is nil"),
			)
		}
		if txnNode.Children[len(txnNode.Children)-1] == nil {
			return txnNode, newError(
				txnNode.X12Node,
				fmt.Errorf("last child node is nil"),
			)
		}
	}

	if len(txnNode.Children) == 0 {
		stNode, err := NewNode(SegmentNode, stSegmentId)
		if err != nil {
			return txnNode, err
		}

		txnNode.Header = stNode
		stNode.Parent = txnNode.X12Node

		seNode, err := NewNode(SegmentNode, seSegmentId)
		if err != nil {
			return txnNode, err
		}

		txnNode.Trailer = seNode
		seNode.Parent = txnNode.X12Node

		txnNode.Children = []*X12Node{stNode, seNode}
	} else if len(txnNode.Children) >= 2 {
		txnNode.Header = txnNode.Children[0]
		txnNode.Trailer = txnNode.Children[len(txnNode.Children)-1]
	} else {
		return txnNode, newError(
			txnNode.X12Node,
			fmt.Errorf(
				"expected zero or at least 2 children, got %d",
				len(txnNode.Children),
			),
		)
	}
	err = txnNode.Header.SetSpec(stSpec)
	if err != nil {
		return txnNode, newError(txnNode.Header, err)
	}

	err = txnNode.Trailer.SetSpec(seSpec)
	if err != nil {
		return txnNode, newError(txnNode.Trailer, err)
	}

	stHeader := txnNode.Children[0]

	for i := 1; i < len(stHeader.Children); i++ {
		switch i {
		case stIndexTransactionSetCode:
			txnNode.TransactionSetCode = stHeader.Children[i].Value[0]
		case stIndexVersionCode:
			txnNode.VersionCode = stHeader.Children[i].Value[0]
		case stIndexControlNumber:
			txnNode.ControlNumber = stHeader.Children[i].Value[0]
		}
	}

	levels, err := createHierarchicalLevels(node)
	txnNode.HierarchicalLevels = levels
	if err != nil {
		err = newError(node, err)
	}
	return txnNode, err
}

func (x *X12Spec) defaultObjValue() (n *X12Node, err error) {
	if x.Type == UnknownSpec {
		return n, errors.New("unknown spec type")
	}

	n = &X12Node{}
	err = n.SetSpec(x)
	if err != nil {
		return n, err
	}

	switch x.Type {
	case SegmentSpec:
		return defaultSegmentObj(x)
	case LoopSpec:
		return defaultLoopObj(x)
	case CompositeSpec:
		return defaultCompositeObj(x)
	case ElementSpec:
		return elementNodeWithDefaultValue(x)
	}
	return n, err
}

func defaultCompositeObj(x *X12Spec) (cmpNode *X12Node, err error) {
	var errs []error
	cmpNode, err = NewNode(CompositeNode, x.Name)
	if err != nil {
		return cmpNode, err
	}
	if err = cmpNode.SetSpec(x); err != nil {
		return cmpNode, err
	}

	errs = append(errs, err)
	for i := 0; i < len(x.Structure); i++ {
		childStruct := x.Structure[i]
		if childStruct.Required() {
			elemDefault, e := elementNodeWithDefaultValue(childStruct)
			errs = append(errs, e)
			e = cmpNode.Replace(i, elemDefault)
			errs = append(errs, e)
		}

	}
	return cmpNode, errors.Join(errs...)
}

// Replace replaces the node at the given index with the given node.
// The paths of the current node's children will be updated, including
// the replacement node. The node being replaced will be detached
// from the current node, and its paths (including children) will
// be updated.
// It will return an error if the given node's NodeType is not present
// in ChildTypes()
func (n *X12Node) Replace(index int, node *X12Node) error {
	if index < 0 || index >= len(n.Children) {
		return errors.New("index out of range")
	}

	if !sliceContains(n.ChildTypes(), node.Type) {
		return newError(
			n,
			fmt.Errorf(
				"cannot append node of type %s (must be one of: %v)",
				node.Type, n.ChildTypes(),
			),
		)
	}

	currentNode := n.Children[index]
	currentNode.Parent = nil
	n.Children[index] = node
	node.Parent = n
	if node.Name == "" {
		if node.Spec == nil {
			switch n.Type {
			case SegmentNode:
				if node.Type == ElementNode && index == 0 {
					node.Name = n.Name
				} else {
					node.Name = fmt.Sprintf("%s%02d", n.Name, len(n.Children)-1)
				}
			case CompositeNode:
				node.Name = fmt.Sprintf("%s%02d", n.Name, len(n.Children)-1)
			}
		} else {
			node.Name = node.Spec.Name
		}
	}
	if e := currentNode.setPath(); e != nil {
		return e
	}
	return n.setChildPaths()
}

func defaultLoopObj(x *X12Spec) (loopVal *X12Node, err error) {
	loopVal, err = NewNode(LoopNode, x.Name)
	if err != nil {
		return loopVal, err
	}

	if err = loopVal.SetSpec(x); err != nil {
		return loopVal, err
	}

	for _, childSpec := range x.Structure {
		if !childSpec.Required() {
			continue
		}
		defaultVal, e := childSpec.defaultObjValue()
		if e != nil {
			return loopVal, e
		}
		loopVal.Children = append(loopVal.Children, defaultVal)
	}
	return loopVal, err
}

func defaultSegmentObj(x *X12Spec) (n *X12Node, err error) {
	if x.Type != SegmentSpec {
		return nil, errors.New("spec is not a segment")
	}
	n, err = NewNode(SegmentNode, x.Name)
	if err != nil {
		return n, err
	}
	if err = n.SetSpec(x); err != nil {
		log.Fatalf("uhhsfddfd %s", err.Error())
		return n, err
	}
	errs := []error{}
	for i := 0; i < len(x.Structure); i++ {
		childSpec := x.Structure[i]
		if childSpec.Required() {
			defaultVal, e := childSpec.defaultObjValue()
			if e != nil {
				log.Printf("failed on %s: %s", childSpec.Name, e.Error())
			}

			errs = append(errs, e)
			elemInd := i + 1
			if elemInd == len(n.Children) {
				n.Children = append(n.Children, defaultVal)
			} else {
				n.Children[i+1] = defaultVal
			}
		}
	}
	return n, errors.Join(errs...)
}

// elementNodeWithDefaultValue creates a new X12Node with type ElementNode from
// the given ElementSpec. X12Node.value is set to the default value for the
// element, if there is one. If there is no default value, an empty string
// is set as the first value - for RepeatElementNode, an empty array will
// be set.
func elementNodeWithDefaultValue(e *X12Spec) (
	elementNode *X12Node,
	err error,
) {
	elementNode = &X12Node{Type: ElementNode}
	if err = elementNode.SetSpec(e); err != nil {
		return elementNode, err
	}

	defaultVal := defaultElementString(e)
	if defaultVal != "" {
		err = elementNode.SetValue(defaultVal)
	}
	return elementNode, err
}

// defaultElementString returns the default value for an element spec,
// determined first by ElementSpec.defaultVal and then by ElementSpec.validCodes
// (if there is only one entry). If there is no default value, an empty string
// is returned.
func defaultElementString(e *X12Spec) string {
	if len(e.DefaultVal) > 0 {
		return e.DefaultVal
	}

	if len(e.ValidCodes) == 1 {
		return e.ValidCodes[0]
	}
	return ""
}

func NewMessage() *Message {
	n := &X12Node{Type: MessageNode, Name: "X12"}
	return &Message{X12Node: n}
}

func (n *X12Node) Segments() []*X12Node {
	segs := []*X12Node{}

	for _, child := range n.Children {
		if child == nil {
			log.Printf("nil child under %s", n.Name)
		}
		if child.Type == SegmentNode {
			segs = append(segs, child)
		} else {
			segs = append(segs, child.Segments()...)
		}
	}
	return segs
}

func (n *X12Node) Format(
	segmentTerminator string,
	elementSeparator string,
	repetitionSeparator string,
	componentElementSeparator string,
) string {
	segLines := []string{}

	switch n.Type {
	case CompositeNode:
		elems := []string{}

		for _, cmpChild := range n.Children {
			elemVal := cmpChild.Value
			if len(elemVal) == 0 {
				elems = append(elems, "")
			} else {
				elems = append(elems, elemVal[0])
			}
		}

		elems = removeTrailingEmptyElements(elems)
		elemStr := strings.Join(elems, componentElementSeparator)
		return elemStr
	case SegmentNode:
		var elems []string

		for _, segChild := range n.Children {
			switch segChild.Type {
			case ElementNode:
				elemVal := segChild.Value
				if len(elemVal) == 0 {
					elems = append(elems, "")
				} else {
					elems = append(elems, elemVal[0])
				}
			case RepeatElementNode:
				elemVal := segChild.Value
				repVals := removeTrailingEmptyElements(elemVal)
				if len(repVals) == 0 {
					elems = append(elems, "")
				} else if len(repVals) == 1 {
					elems = append(elems, repVals[0])
				} else {
					elems = append(
						elems,
						strings.Join(repVals, repetitionSeparator),
					)
				}
			case CompositeNode:
				cmpStr := segChild.Format(
					segmentTerminator,
					elementSeparator,
					repetitionSeparator,
					componentElementSeparator,
				)
				elems = append(elems, cmpStr)
			}
		}
		elems = removeTrailingEmptyElements(elems)
		elemStr := strings.Join(elems, elementSeparator)
		return elemStr
	}

	for _, seg := range n.Segments() {
		segStr := seg.Format(
			segmentTerminator,
			elementSeparator,
			repetitionSeparator,
			componentElementSeparator,
		)
		segLines = append(segLines, segStr)
	}

	s := strings.Join(segLines, segmentTerminator)
	s += segmentTerminator
	return s
}

func validateElement(element *X12Node) error {
	var validationErrors []error

	if element.Type != ElementNode && element.Type != RepeatElementNode {
		validationErrors = append(
			validationErrors,
			fmt.Errorf("expected element, got %s", element.Type),
		)
		return errors.Join(validationErrors...)
	}
	if element.Spec == nil {
		validationErrors = append(
			validationErrors,
			newError(element, fmt.Errorf("nil spec")),
		)
		return errors.Join(validationErrors...)
	}

	if element.Spec.IsRepeatable() && element.Type != RepeatElementNode {
		validationErrors = append(
			validationErrors,
			fmt.Errorf("expected repeat element, got %s", element.Type),
		)
	} else if !element.Spec.IsRepeatable() && element.Type != ElementNode {
		validationErrors = append(
			validationErrors,
			fmt.Errorf("expected element, got %s", element.Type),
		)
	}

	switch element.Type {
	case ElementNode:
		elemVal := ""
		if len(element.Value) == 1 {
			elemVal = element.Value[0]
		}
		elementErrors := validateElementValue(element.Spec, elemVal)
		validationErrors = append(validationErrors, elementErrors)
	case RepeatElementNode:
		v := element.Value
		if element.Spec.IsRepeatable() {
			if element.Spec.RepeatMax != 0 && len(v) > element.Spec.RepeatMax {
				validationErrors = append(
					validationErrors,
					fmt.Errorf(
						"element %s has too many occurrences, max %d",
						element.Spec.Name,
						element.Spec.RepeatMax,
					),
				)
			}
			if element.Spec.RepeatMin != 0 && len(v) < element.Spec.RepeatMin {
				validationErrors = append(
					validationErrors,
					fmt.Errorf(
						"element %s has too few occurrences, min %d",
						element.Spec.Name,
						element.Spec.RepeatMin,
					),
				)
			}
		}
		for i, repElem := range v {
			elementErrors := validateElementValue(element.Spec, repElem)
			if elementErrors != nil {
				validationErrors = append(
					validationErrors,
					fmt.Errorf(
						"element %s, Occurrence %d: %s",
						element.Spec.Name,
						i+1,
						elementErrors,
					),
				)
			}
		}
	}
	return errors.Join(validationErrors...)
}

// validateElementValue will evaluate the given string against
// ElementSpec, and return an error if it's invalid according
// to the spec.
func validateElementValue(spec *X12Spec, element string) error {
	if spec.Type != ElementSpec {
		return fmt.Errorf("expected element spec, got %s", spec.Type)
	}
	var validationErrors []error

	// If a value has been provided and validCodes is defined, check that
	// it's in the allowed list
	if element != "" && len(spec.ValidCodes) > 0 {
		if !sliceContains(spec.ValidCodes, element) {
			validationErrors = append(
				validationErrors,
				fmt.Errorf(
					"[element: %s] [path: %s]: %w (got: %v) (valid values: %+v)",
					spec.Name,
					spec.Path,
					ErrElementHasInvalidValue,
					element,
					spec.ValidCodes,
				),
			)
		}

	}

	elementLength := len(element)
	if elementLength == 0 {
		return nil
	}

	if spec.MaxLength != 0 && elementLength > spec.MaxLength {
		validationErrors = append(
			validationErrors,
			fmt.Errorf(
				"[element: %s] [path: %s]: %w (too long - max length is %d)",
				spec.Name,
				spec.Path,
				ErrElementHasBadLength,
				spec.MaxLength,
			),
		)
	} else if spec.MinLength != 0 && elementLength < spec.MinLength {
		validationErrors = append(
			validationErrors,
			fmt.Errorf(
				"[element: %s] [path: %s]: %w (too short - min length is %d)",
				spec.Name,
				spec.Path,
				ErrElementHasBadLength,
				spec.MinLength,
			),
		)
	}
	return errors.Join(validationErrors...)
}

// validateSegmentNode validates a segment node's child values against
// its SegmentSpec
func validateSegmentNode(x *X12Node) error {
	var validationErrors []error
	if x.Type != SegmentNode {
		return fmt.Errorf("expected SegmentNode, got '%s'", x.Type)
	}
	segmentSpec := x.Spec

	if len(x.Children) == 0 {
		return fmt.Errorf("segment %s has no elements", segmentSpec.Name)
	}
	segIdElement := x.Children[0]

	if segIdElement.Length() == 0 {
		validationErrors = append(
			validationErrors,
			fmt.Errorf(
				"[segment: %s] first element value not set",
				segmentSpec.Name,
			),
		)
	} else if segmentSpec == nil && segIdElement.Value[0] != x.Name {
		validationErrors = append(
			validationErrors,
			fmt.Errorf(
				"[segment: %s] expected first element value '%s' to match segment id '%s'",
				x.Name,
				segIdElement.Value[0],
				x.Name,
			),
		)
	} else if segIdElement.Value[0] != segmentSpec.Name {
		validationErrors = append(
			validationErrors,
			fmt.Errorf(
				"[segment: %s] expected first element value '%s' to match segment id '%s'",
				segmentSpec.Name,
				segIdElement.Value[0],
				segmentSpec.Name,
			),
		)
	}

	if segmentSpec == nil {
		return errors.Join(validationErrors...)
	}

	numElementSpecs := len(segmentSpec.Structure)
	elements := x.Children[1:]
	numElements := len(elements)
	if numElements > numElementSpecs {
		validationErrors = append(
			validationErrors,
			fmt.Errorf(
				"[segment: %s] [path: %s]: spec defines %d elements, segment has %d",
				x.Name, x.Path, numElementSpecs, numElements,
			),
		)
		elements = elements[:numElementSpecs]
		numElements = len(elements)
	}

	for i := 0; i < len(segmentSpec.Structure); i++ {
		validationErrors = append(validationErrors, elements[i].Validate())

	}
	for _, condition := range segmentSpec.Syntax {
		err := validateSegmentSyntax(x, &condition)
		if err != nil {
			validationErrors = append(validationErrors, err)
		}
	}

	return errors.Join(validationErrors...)
}

type elementValue struct {
	Type  NodeType
	Value []string
}

// getSegmentNodeElements returns a slice of elementValue structs created
// from X12Node.Children. Trailing empty elements are removed.
func getSegmentNodeElements(segmentNode *X12Node) []elementValue {
	segElements := make([]elementValue, len(segmentNode.Children))

	for i := 0; i < len(segmentNode.Children); i++ {
		child := segmentNode.Children[i]
		childVal := removeTrailingEmptyElements(child.Value)
		segElements[i] = elementValue{Type: child.Type, Value: childVal}
	}
	// return a slice reflecting all elements up to the last non-empty element
	for i := len(segElements) - 1; i >= 0; i-- {
		if len(segElements[i].Value) > 0 {
			return segElements[:i+1]
		}
	}
	return []elementValue{}
}

// validateElementValue validates a segment's element values against
// the segment's syntax rules
func validateSegmentSyntax(
	segment *X12Node,
	condition *SegmentCondition,
) error {
	segElements := getSegmentNodeElements(segment)
	elementCount := len(segElements)
	elements := []bool{}

	for _, syntaxIndex := range condition.Indexes {
		if syntaxIndex < elementCount {
			e := segElements[syntaxIndex]
			switch e.Type {
			case ElementNode:
				estr := ""
				eVal := segElements[syntaxIndex].Value
				if len(eVal) > 0 {
					estr = eVal[0]
				}
				elements = append(
					elements,
					len(estr) > 0,
				)
			default:
				elements = append(
					elements,
					len(segElements[syntaxIndex].Value) > 0,
				)
			}

		} else {
			elements = append(elements, false)
		}
	}

	switch condition.ConditionCode {
	case PairedOrMultiple:
		expectedValueCt := len(condition.Indexes)
		var foundValueCt int
		for _, element := range elements {
			if element != false {
				foundValueCt++
			}
		}
		if foundValueCt > 0 && foundValueCt != expectedValueCt {
			return fmt.Errorf(
				"[segment: %s] [path: %s]: %w: when at least one value in %+v is present, all must be present",
				segment.Name,
				segment.Path,
				SegmentConditionErr,
				condition.Indexes,
			)

		}
	case RequiredCondition:
		for _, element := range elements {
			if element == true {
				return nil
			}
		}
		return fmt.Errorf(
			"[segment: %s] [path: %s]: %w: at least one of %+v is required",
			segment.Name,
			segment.Path,
			SegmentConditionErr,
			condition.Indexes,
		)
	case Exclusion:
		var foundValueCt int
		for _, element := range elements {
			if element == true {
				foundValueCt++
			}
		}
		if foundValueCt > 1 {
			return fmt.Errorf(
				"[segment: %s] [path: %s]: %w: only one of %+v is allowed",
				segment.Name,
				segment.Path,
				SegmentConditionErr,
				condition.Indexes,
			)
		}
	case Conditional:
		if len(elements) > 0 {
			if elements[0] != false {
				for _, element := range elements[1:] {
					if element == false {
						return fmt.Errorf(
							"\"[segment: %s] [path: %s]: %w: when element %d is present, all must be present",
							segment.Name,
							segment.Path,
							SegmentConditionErr,
							condition.Indexes[0],
						)
					}
				}
			}
		}
	case ListConditional:
		var foundValueCt int
		initialVal := elements[0]
		if initialVal != false {
			for _, element := range elements[1:] {
				if element != false {
					foundValueCt++
				}
			}
			if foundValueCt == 0 {
				return fmt.Errorf(
					"[segment: %s] [path: %s]: %w: if a value is present in element %d, at least one "+
						"value must be present in element %d",
					segment.Name,
					segment.Path,
					SegmentConditionErr,
					condition.Indexes[0],
					condition.Indexes[1],
				)
			}
		}
	}
	return nil
}

func validateLoop(loop *X12Node) error {
	var validationErrors []error
	if loop.Spec == nil {
		return fmt.Errorf("loop has no spec")
	}
	loopSpec := loop.Spec

	loopSpecCount := make(map[*X12Spec]int)
	segmentSpecCount := make(map[*X12Spec]int)

	for _, x12obj := range loop.Children {
		if x12obj.Spec == nil {
			validationErrors = append(
				validationErrors,
				newError(x12obj, ErrMissingSpec),
			)
			continue
		}
		switch x12obj.Type {
		case SegmentNode:
			sSpec := x12obj.Spec
			segmentSpecCount[sSpec]++
		case LoopNode:
			lSpec := x12obj.Spec
			loopSpecCount[lSpec]++
		default:
			return newError(
				x12obj,
				fmt.Errorf("expected segment or node, got %v", x12obj.Type),
			)
		}
	}

	for loopSpecPtr, ct := range loopSpecCount {
		if loopSpecPtr.RepeatMin != 0 && ct < loopSpecPtr.RepeatMin {
			validationErrors = append(
				validationErrors,
				fmt.Errorf(
					"[loop: %s] [path: %s]: %w (min: %d found: %d)",
					loopSpecPtr.Name,
					loopSpecPtr.Path,
					ErrTooFewLoops,
					loopSpecPtr.RepeatMin,
					ct,
				),
			)
		} else if loopSpecPtr.RepeatMax != 0 && ct > loopSpecPtr.RepeatMax {
			validationErrors = append(
				validationErrors,
				fmt.Errorf(
					"[loop: %s] [path: %s]: %w (max: %d found: %d)",
					loopSpecPtr.Name,
					loopSpecPtr.Path,
					ErrTooManyLoops,
					loopSpecPtr.RepeatMax,
					ct,
				),
			)
		}
	}

	for segmentSpecPtr, ct := range segmentSpecCount {
		if segmentSpecPtr.RepeatMin != 0 && ct < segmentSpecPtr.RepeatMin {
			validationErrors = append(
				validationErrors,
				fmt.Errorf(
					"[segment: %s] [path: %s]: %w (min: %d found: %d)",
					segmentSpecPtr.Name,
					segmentSpecPtr.Path,
					ErrTooFewSegments,
					segmentSpecPtr.RepeatMin,
					ct,
				),
			)
		} else if segmentSpecPtr.RepeatMax != 0 && ct > segmentSpecPtr.RepeatMax {
			validationErrors = append(
				validationErrors,
				fmt.Errorf(
					"[segment: %s] [path: %s]: %w (max: %d found: %d)",
					segmentSpecPtr.Name,
					segmentSpecPtr.Path,
					ErrTooManySegments,
					segmentSpecPtr.RepeatMax,
					ct,
				),
			)
		}
	}

	for _, x12def := range loopSpec.Structure {
		switch x12def.Type {
		case SegmentSpec:
			if x12def.Required() {
				_, hasKey := segmentSpecCount[x12def]
				if !hasKey {
					validationErrors = append(
						validationErrors,
						fmt.Errorf(
							"[segment: %s] [path: %s]: %w",
							x12def.Name,
							x12def.Path,
							ErrRequiredSegmentMissing,
						),
					)
					continue
				}
			}
		case LoopSpec:
			if x12def.Required() {
				_, hasKey := loopSpecCount[x12def]
				if !hasKey {
					validationErrors = append(
						validationErrors,
						fmt.Errorf(
							"[loop: %s] [path: %s]: %w",
							x12def.Name,
							x12def.Path,
							ErrRequiredLoopMissing,
						),
					)
					continue
				}
			}
		default:
			return newError(
				loop, fmt.Errorf(
					"expected segment or loop spec, got %v", x12def.Type,
				),
			)
		}
	}

	for _, loopChild := range loop.Children {
		validationErrors = append(validationErrors, loopChild.Validate())
	}
	return errors.Join(validationErrors...)
}

// validateComposite validates a composite element against the composite spec
// and returns any errors
func validateComposite(
	composite *X12Node,
) error {
	var validationErrors []error
	if composite.Type != CompositeNode {
		return fmt.Errorf(
			"expected composite node, got %v",
			composite.Type,
		)
	}

	componentElements := []string{}
	for _, e := range composite.Children {
		ev := e.Value
		if len(ev) == 0 {
			componentElements = append(componentElements, "")
		} else {
			componentElements = append(componentElements, ev[0])
		}
	}
	componentElements = removeTrailingEmptyElements(componentElements)

	if composite.Spec == nil {
		validationErrors = append(
			validationErrors, newError(composite, fmt.Errorf("nil spec")),
		)
		return errors.Join(validationErrors...)
	}
	compositeSpec := composite.Spec
	if compositeSpec.NotUsed() && len(componentElements) > 0 {
		validationErrors = append(
			validationErrors,
			fmt.Errorf("composite %s is not used", compositeSpec.Name),
		)
		return errors.Join(validationErrors...)
	}

	if compositeSpec.Situational() && len(componentElements) == 0 {
		return nil
	}

	numElementSpecs := len(compositeSpec.Structure)
	numElements := len(componentElements)
	if numElements > numElementSpecs {
		validationErrors = append(
			validationErrors,
			fmt.Errorf(
				"composite %s has %d elements, but only %d are defined in the schema",
				compositeSpec.Name,
				numElements,
				numElementSpecs,
			),
		)
		componentElements = componentElements[:numElementSpecs]
		numElements = len(componentElements)
	}

	for i, x12def := range compositeSpec.Structure {
		if i >= numElements {
			if x12def.Required() {
				validationErrors = append(
					validationErrors,
					fmt.Errorf(
						"composite %s is missing required element %s (path: %s) (description: %s) (label: %s)",
						compositeSpec.Name,
						x12def.Name,
						x12def.Path,
						x12def.Description,
						x12def.labelWithParentLabels(),
					),
				)
			}
			continue
		} else {
			elementErrors := validateElement(composite.Children[i])
			if elementErrors != nil {
				validationErrors = append(
					validationErrors, fmt.Errorf(
						"element %s: %s", x12def.Name, elementErrors,
					),
				)
			}
		}
	}

	return errors.Join(validationErrors...)
}

func (x *X12Spec) parentLabels() []string {
	labels := []string{}
	var currentNode *X12Spec
	currentNode = x.Parent
	for currentNode != nil {
		labels = append(labels, currentNode.Label)
		currentNode = currentNode.Parent
	}
	return labels
}

func reverseArray(arr []string) []string {
	newArray := []string{}

	for i := len(arr) - 1; i >= 0; i-- {
		newArray = append(newArray, arr[i])
	}
	return newArray
}

func (x *X12Spec) labelWithParentLabels() string {
	parentLabels := reverseArray(x.parentLabels())
	parentLabels = append(parentLabels, x.Label)
	return strings.Join(parentLabels, ".")
}

func (txn *TransactionSetNode) Validate() error {
	segmentCt := len(txn.Segments())
	validationErrors := []error{}
	trailer := txn.Trailer
	if trailer.Name != seSegmentId {
		validationErrors = append(
			validationErrors,
			newError(
				trailer, fmt.Errorf(
					"%w: last segment in transaction must be %s, got %s",
					ErrInvalidTransactionEnvelope,
					seSegmentId,
					trailer.Name,
				),
			),
		)
	} else {
		seSegmentCt := trailer.Children[seIndexNumberOfIncludedSegments]
		seSegmentCtValue, e := strconv.Atoi(seSegmentCt.Value[0])
		if e != nil {
			validationErrors = append(
				validationErrors,
				newError(
					trailer.Children[seIndexNumberOfIncludedSegments],
					e,
				),
			)
		}

		if e == nil && seSegmentCtValue != segmentCt {
			validationErrors = append(
				validationErrors,
				newError(
					trailer, fmt.Errorf(
						"%w: expected %d segments from %s trailer count, got %d",
						ErrInvalidTransactionEnvelope,
						seSegmentCtValue,
						seSegmentId,
						segmentCt,
					),
				),
			)
		}
		header := txn.Header
		stControlNum := header.Children[stIndexControlNumber].Value[0]
		seControlNum := trailer.Children[seIndexControlNumber].Value[0]
		if stControlNum != seControlNum {
			validationErrors = append(
				validationErrors,
				newError(
					txn.X12Node, fmt.Errorf(
						"%w: ST control number %s does not match SE control number %s",
						ErrInvalidTransactionEnvelope,
						stControlNum,
						seControlNum,
					),
				),
			)
		}
	}
	for i := 0; i < len(txn.Children); i++ {
		validationErrors = append(
			validationErrors,
			txn.Children[i].Validate(),
		)
	}
	if txn.TransactionSpec == nil {
		validationErrors = append(
			validationErrors,
			newError(
				txn.X12Node,
				fmt.Errorf("no spec set for transaction, cannot validate. try Message.Transform()"),
			),
		)
	}
	return errors.Join(validationErrors...)
}

func (x *FunctionalGroupNode) Validate() error {
	transactionSetCt := 0
	validationErrors := []error{}
	gsHeader := x.Header
	idCode := gsHeader.Children[gsIndexFunctionalIdentifierCode].Value[0]
	allowedTxnCodes := []string{}
	for t, i := range functionalIdentifierCodes {
		if i == idCode {
			allowedTxnCodes = append(allowedTxnCodes, t)
		}
	}

	for _, c := range x.Transactions {
		transactionSetCt++
		stHeader := c.Header
		txnCode := stHeader.Children[stIndexTransactionSetCode].Value[0]
		if !sliceContains(allowedTxnCodes, txnCode) {
			validationErrors = append(
				validationErrors,
				newError(
					stHeader.Children[stIndexTransactionSetCode],
					fmt.Errorf(
						"%w: expected transaction set code %s, got %s",
						ErrInvalidTransactionEnvelope,
						allowedTxnCodes,
						txnCode,
					),
				),
			)
		}
	}

	trailer := x.Trailer
	if trailer.Name != geSegmentId {
		validationErrors = append(
			validationErrors,
			fmt.Errorf(
				"%w: last segment in transaction must be GE, got %s",
				ErrInvalidGroupEnvelope,
				trailer.Name,
			),
		)
	} else {
		geSegmentCt := trailer.Children[geIndexNumberOfIncludedTransactionSets]
		geSegmentCtValue, e := strconv.Atoi(geSegmentCt.Value[0])
		validationErrors = append(validationErrors, e)
		if e == nil && geSegmentCtValue != transactionSetCt {
			validationErrors = append(
				validationErrors,
				newError(
					trailer, fmt.Errorf(
						"%w: expected %d transaction sets from GE trailer count, got %d",
						ErrInvalidGroupEnvelope,
						geSegmentCtValue,
						transactionSetCt,
					),
				),
			)
		}

		gsControlNum := gsHeader.Children[gsIndexControlNumber].Value[0]
		geControlNum := trailer.Children[geIndexControlNumber].Value[0]
		if gsControlNum != geControlNum {
			validationErrors = append(
				validationErrors,
				newError(
					trailer, fmt.Errorf(
						"%w: GS control number %s does not match GE control number %s",
						ErrInvalidGroupEnvelope,
						gsControlNum,
						geControlNum,
					),
				),
			)
		}
	}

	validationErrors = append(validationErrors, x.Header.Validate())
	validationErrors = append(validationErrors, x.Trailer.Validate())

	for i := 0; i < len(x.Transactions); i++ {
		validationErrors = append(
			validationErrors,
			x.Transactions[i].Validate(),
		)
	}
	return errors.Join(validationErrors...)
}

//func validateMessageNode(x *X12Node) error {
//	validationErrors := []error{}
//	for i := 0; i < len(x.Children); i++ {
//		validationErrors = append(
//			validationErrors,
//			x.Children[i].Validate(),
//		)
//	}
//	return errors.Join(validationErrors...)
//}

func findTransactionSpec(
	transactionSetCode string,
	version string,
) (*X12TransactionSetSpec, error) {
	availableTransactions := []TransactionIdentifier{}
	for _, v := range transactionSpecs {
		if v.TransactionSetCode == transactionSetCode && (version == "" || v.Version == version) {
			return v, nil
		}
		availableTransactions = append(
			availableTransactions, TransactionIdentifier{
				TransactionSetCode: v.TransactionSetCode,
				Version:            v.Version,
			},
		)
	}
	return nil, fmt.Errorf(
		"transaction set spec not found: %s, %s (available: %#v)",
		transactionSetCode, version, availableTransactions,
	)
}

// segmentDeque mimics Python's `collections.deque`, for Segment instances.
// It's used to store the remaining segments to be parsed/mapped, across
// loopTransformer instances.
type segmentDeque struct {
	segments *list.List
}

// Append adds a Segment to the end of the deque.
func (d *segmentDeque) Append(value *X12Node) {
	d.segments.PushBack(value)
}

// AppendLeft adds a Segment to the beginning of the deque.
func (d *segmentDeque) AppendLeft(value *X12Node) {
	d.segments.PushFront(value)
}

// PopLeft removes and returns the first Segment in the deque.
// If the deque is empty, it returns nil.
func (d *segmentDeque) PopLeft() *X12Node {
	if d.segments.Len() > 0 {
		first := d.segments.Front()
		d.segments.Remove(first)
		return first.Value.(*X12Node)
	} else {
		return nil
	}
}

// Length returns the number of Segments in the deque.
func (d *segmentDeque) Length() int {
	return d.segments.Len()
}

// loopTransformer is a visitor that maps Segment instances to
// the given LoopSpec structure's SegmentSpec. Matched segments are popped
// off the beginning of SegmentQueue and appended to the Children slice.
// If unmatched segments remain and another LoopSpec is encountered,
// a new loopTransformer is created for that LoopSpec and the same
// SegmentQueue is passed to it.
type loopTransformer struct {
	// LoopSpec is the x12 definition we're visiting/mapping
	LoopSpec *X12Spec
	// SegmentQueue contains the remaining Segment instances to map
	SegmentQueue *segmentDeque
	// Children contains the mapped Segment and loop instances, which
	// will be assigned as child instances of the loop instances created
	// by updateLoop()
	Children []*X12Node
	// ErrorLog contains any errors encountered during mapping
	ErrorLog []error
	// message is the top-level Message instance for the current loop
	message *Message
	// parentLoopBuilder is the loopTransformer that created this instance
	parentLoopBuilder *loopTransformer
	// createdLoop is the X12Node instance (nodeType Loop) created by
	// this loopTransformer
	createdLoop *X12Node

	abort       bool
	abortReason error
	SpecVisitor
}

func (v *loopTransformer) updateLoop() *X12Node {
	if len(v.Children) > 0 {
		for i := 0; i < len(v.Children); i++ {
			x12obj := v.Children[i]
			v.createdLoop.Children = append(v.createdLoop.Children, x12obj)
		}
		v.Children = []*X12Node{}
		//if e := v.createdLoop.SetSpec(v.LoopSpec); e != nil {
		//	v.ErrorLog = append(v.ErrorLog, e)
		//}
		return v.createdLoop
	}
	return nil
}

func newLoopTransformer(
	loopSpec *X12Spec,
	segmentQueue *segmentDeque,
	message *Message,
	parentLoopBuilder *loopTransformer,
) (*loopTransformer, error) {
	builder := &loopTransformer{
		LoopSpec:          loopSpec,
		SegmentQueue:      segmentQueue,
		message:           message,
		parentLoopBuilder: parentLoopBuilder,
		abort:             false,
	}
	parentLoop, err := NewNode(LoopNode, loopSpec.Name)
	if err != nil {
		return nil, err
	}
	err = parentLoop.SetSpec(loopSpec)
	if err != nil {
		log.Printf("unable to set spec: %s", err.Error())
		return nil, err
	}
	builder.createdLoop = parentLoop
	return builder, nil
}

// VisitLoopSpec is called when a LoopSpec is encountered in the x12 definition.
// It creates a new loopTransformer for the LoopSpec, and tries to identify
// matching segments in the SegmentQueue. If a match is found, they're
// added to loopTransformer.Children. If another LoopSpec is encountered,
// a new loopTransformer is created for that LoopSpec and the same process
// is repeated.
func (v *loopTransformer) VisitLoopSpec(loopSpec *X12Spec) {
	segmentQueue := v.SegmentQueue

	if segmentQueue.Length() == 0 {
		return
	}
	if v.parentLoopBuilder != nil {
		pb := v.parentLoopBuilder
		if pb.abort {
			return
		}
	}

	rmax := loopSpec.RepeatMax
	occurrence := 0
	loopCount := 0

	for segmentQueue.Length() > 0 {
		if rmax != 0 && loopCount == rmax {
			break
		}
		if v.abort || (v.parentLoopBuilder != nil && v.parentLoopBuilder.abort) {
			break
		}

		var loopParser *loopTransformer
		if len(v.Children) == 0 {
			return
		}
		loopParser, _ = newLoopTransformer(
			loopSpec, segmentQueue, v.message, v,
		)
		if e := loopParser.cascade(); e != nil {
			v.Abort(e)
			break
		}
		v.ErrorLog = append(v.ErrorLog, loopParser.ErrorLog...)

		if len(loopParser.Children) == 0 {
			break
		} else {
			updated := loopParser.updateLoop()
			if updated == nil {
				break
			} else {
				loopCount += 1
				loopParser.createdLoop.Occurrence = occurrence
				occurrence += 1
				v.Children = append(v.Children, loopParser.createdLoop)
			}
		}
	}
}

func (v *loopTransformer) Abort(e error) {
	v.abort = true
	v.abortReason = e
	if v.parentLoopBuilder != nil {
		v.parentLoopBuilder.Abort(e)
	}
}

func (v *loopTransformer) VisitSpec(x *X12Spec) error {
	switch x.Type {
	case SegmentSpec:
		v.VisitSegmentSpec(x)
	case LoopSpec:
		v.VisitLoopSpec(x)
	}
	return nil
}

// VisitSegmentSpec is called when a SegmentSpec is encountered in the x12
// definition. It will pop segments off of loopTransformer.SegmentQueue as long
// as it can consecutively find matches, without exceeding the maximum
// repeat. If a match isn't found, the segment is pushed back to the front
// of the segment queue, and a new Segment is created for each match
// identified, appending them to the end of loopTransformer.Children
func (v *loopTransformer) VisitSegmentSpec(segmentSpec *X12Spec) {
	segmentQueue := v.SegmentQueue

	// No segments remain to match, exit
	if segmentQueue.Length() == 0 {
		return
	}

	isRequired := segmentSpec.Required()

	// The maximum number of repetitions for the given SegmentSpec,
	// which caps the number of matches. If this is nil, then `repeatMin`
	// will be the minimum number of matches. The minimum is not enforced
	// at this point - that point of validation is left for overall transaction
	// validation.
	// The maximum IS enforced here, for cases where you may have
	// multiple consecutive segment IDs, where the first SegmentSpec may match
	// multiple segments, but have a different meaning than subsequent
	// consecutive segments with the same ID.
	// For example, in an 837 (professional) 2300 loop, there are 16 DTP
	// segments in a row (onset of current illness, initial treatment date,
	// last seen date, ...), where most only support a single occurrence,
	// but not always ('assumed and relinquished care dates' repeats twice)
	rmax := segmentSpec.RepeatMax

	// Tracks consecutive segments which match SegmentSpec
	var matches []*X12Node

	if rmax != 0 {
		matches = make([]*X12Node, 0, rmax)
	}

	// Looping until we either run out of segments to match,
	// or we've reached the maximum number of repetitions for the
	// given SegmentSpec
	for segmentQueue.Length() > 0 && (rmax == 0 || len(matches) < rmax) && !v.abort {
		targetSegmentId := segmentSpec.Name
		// Tracks the segment ID first popped off the queue before entering
		// this nested `for` loop, so we can
		var prevSegmentId string
		for (rmax == 0 || len(matches) < rmax) && !v.abort {
			segmentToken := segmentQueue.PopLeft()
			if segmentToken == nil {
				break
			}

			prevSegmentId = segmentToken.Name

			// The HL (hierarchical level) segment can only be used to
			// start a loop, so if we've already identified segment matches
			// on this loop, we need to stop
			if segmentToken.Name == hlSegmentId && len(v.Children) > 0 {
				segmentQueue.AppendLeft(segmentToken)
				break
			}

			// If the current segment doesn't match, break early
			segMatches, e := segmentMatchesSpec(segmentToken, segmentSpec)
			if e != nil {
				v.ErrorLog = append(v.ErrorLog, newSpecErr(e, segmentSpec))
			}
			if !segMatches {
				segmentQueue.AppendLeft(segmentToken)
				break
			}
			matches = append(matches, segmentToken)
		}

		// If this is a required segment and:
		// - we haven't yet found a match
		// - we've already identified other child segments
		//
		// That means we're missing a required segment.
		//
		// If there were no other children, it means we may be in a
		// situational loop and could just move past the segment, which
		// may be matched in a later loop
		//
		// Missing a required segment will set loopTransformer.abort to true,
		// and propagate it to parent loopTransformers, which (should)
		// circuit-break the parsing process. If we continued parsing,
		// in some cases we may end up mis-identifying a subsequent loop
		// and cause segments that should've been in a single loop to be
		// split over repeats of the same loop, or the wrong loop altogether
		if isRequired && prevSegmentId != targetSegmentId && len(matches) == 0 && len(v.Children) > 0 {
			e := newSpecErr(
				errors.Join(
					ErrTransactionAborted,
					ErrRequiredSegmentMissing,
				),
				segmentSpec,
			)
			v.ErrorLog = append(
				v.ErrorLog,
				e,
			)
			v.Abort(e)
		}

		for occurrence := 0; occurrence < len(matches); occurrence++ {
			segment := matches[occurrence]
			segment.Occurrence = occurrence
			if err := segment.SetSpec(segmentSpec); err != nil {
				log.Printf("unable to set spec: %s", err.Error())
				v.ErrorLog = append(v.ErrorLog, err)
			}
			v.Children = append(v.Children, segment)
		}
		break
	}
}

// cascade visits all immediate child LoopSpec and SegmentSpec instances of
// the current loopTransformer.LoopSpec, and calls their Accept methods,
// which should then call the associated VisitSegmentSpec or VisitLoopSpec
// function for their type
func (v *loopTransformer) cascade() error {
	v.Children = []*X12Node{}
	if v.SegmentQueue.Length() == 0 {
		return nil
	}
	loopStructure := v.LoopSpec.Structure
	frontSeg := v.SegmentQueue.segments.Front().Value.(*X12Node)
	if len(loopStructure) > 0 && loopStructure[0].Name != frontSeg.Name {
		return nil
	}
	for i := 0; i < len(loopStructure); i++ {
		if v.abort || (v.parentLoopBuilder != nil && v.parentLoopBuilder.abort) || v.SegmentQueue.Length() == 0 {
			break
		}

		childSpec := v.LoopSpec.Structure[i]
		if i == 0 && childSpec.Name != v.SegmentQueue.segments.Front().Value.(*X12Node).Name {
			continue
		}
		childSpec.Accept(v)
	}
	return nil
}

// segmentMatchesSpec returns true if the given SegmentNode X12Node
// child values (ElementNode, CompositeNode, RepeatElementNode) match
// the given SegmentSpec.
// The first child ElementNode value must match the SegmentSpec X12Spec.Name.
// Any required ElementSpec must have a corresponding populated (non-empty)
// value.
// If ElementSpec.validCodes is not empty, the corresponding value must
// be in the list (for RepeatElementNode, all values must be in the list).
// CompositeNode nodes are not evaluated.
func segmentMatchesSpec(
	segment *X12Node,
	segmentSpec *X12Spec,
) (isMatch bool, err error) {
	if segment.Name != segmentSpec.Name {
		return false, err
	}

	elems := getSegmentNodeElements(segment)
	if len(elems) == 0 {
		return false, err
	}

	// Segment def does not include the segment ID in `structure`
	comparedElements := elems[1:]
	structureLen := len(segmentSpec.Structure)
	elementsLen := len(comparedElements)

	maxLen := int(math.Max(float64(structureLen), float64(elementsLen)))

	for i := 0; i < maxLen; i++ {
		// got an extra element, no match
		if i >= structureLen {
			return false, err
		}

		elemSpec := segmentSpec.Structure[i]
		if elemSpec.NotUsed() {
			continue
		}
		if elemSpec.Type == CompositeSpec {
			continue
		}

		if i >= elementsLen {
			// if we have a required element past the end of the provided
			// values, no match
			if elemSpec.Required() {
				return false, err
			}
			continue
		}

		elementVal := comparedElements[i]
		elemValLength := len(elementVal.Value)
		if elemValLength == 0 && elemSpec.Required() {
			return false, err
		}

		if elemValLength == 0 && elemSpec.Situational() {
			continue
		}

		validCodes := elemSpec.ValidCodes
		if len(validCodes) == 0 {
			continue
		}

		var inValidCodes bool
		switch elementVal.Type {
		case ElementNode:
			inValidCodes = sliceContains(validCodes, elementVal.Value[0])
		case RepeatElementNode:
			for _, repElement := range elementVal.Value {
				if sliceContains(validCodes, repElement) {
					inValidCodes = true
					break
				}
			}
		}
		if !inValidCodes {
			return false, err
		}
	}
	return true, err
}

var ErrTransactionAborted = errors.New("aborted transaction")

func payloadFromX12(n *X12Node) (map[string]any, error) {
	payload := map[string]any{}
	var payloadErrs []error

	for i := 0; i < len(n.Children); i++ {
		childNode := n.Children[i]
		if childNode.Spec == nil {
			payloadErrs = append(
				payloadErrs,
				fmt.Errorf(
					"[name: %s] [path: %s] childNode.Spec is nil",
					childNode.Name,
					childNode.Path,
				),
			)
			continue
		}
		if childNode.Spec.NotUsed() {
			continue
		}

		switch childNode.Type {
		case ElementNode:
			childVal := childNode.Value
			if len(childVal) == 0 {
				payload[childNode.Spec.Label] = ""
			} else if len(childVal) == 1 {
				payload[childNode.Spec.Label] = childNode.Value
			}
		case RepeatElementNode:
			childVal := childNode.Value
			if len(childVal) > 0 {
				payload[childNode.Spec.Label] = childVal
			}
		case CompositeNode:
			childVal, _ := childNode.Payload()
			if allZero(childVal) {
				payload[childNode.Spec.Label] = map[string]any{}
			} else {
				payload[childNode.Spec.Label] = childVal
			}
		case SegmentNode:
			childVal, segErr := childNode.Payload()
			payloadErrs = append(payloadErrs, segErr)
			if allZero(childVal) {
				continue
			}
			childLabel := childNode.Spec.Label

			if childNode.Spec.IsRepeatable() {
				_, ok := payload[childLabel]
				if !ok {
					var tmpval []map[string]any
					payload[childLabel] = tmpval
				}
				val := payload[childLabel]
				payload[childLabel] = append(val.([]map[string]any), childVal)
			} else {
				payload[childNode.Spec.Label] = childVal
			}

		case LoopNode:
			childVal, loopErr := childNode.Payload()
			payloadErrs = append(payloadErrs, loopErr)
			if allZero(childVal) {
				continue
			}
			childLabel := childNode.Spec.Label
			if childNode.Spec.IsRepeatable() {
				_, ok := payload[childLabel]
				if !ok {
					var tmpval []map[string]any
					payload[childLabel] = tmpval
				}
				val := payload[childLabel]
				svv, ok := val.([]map[string]any)
				if !ok {
					return payload, newError(
						childNode,
						fmt.Errorf(
							"expected []map[string]any, got %T",
							payload[childLabel],
						),
					)
				}
				payload[childLabel] = append(svv, childVal)
			} else {
				payload[childNode.Spec.Label] = childVal
			}
		default:
			return payload, newError(
				childNode,
				fmt.Errorf("unknown node type: %v", childNode.Type),
			)
		}
	}
	return payload, errors.Join(payloadErrs...)
}

func removeEmptyOrZero(m map[string]interface{}) {
	for k, v := range m {
		switch v := v.(type) {
		case string:
			if v == "" {
				delete(m, k)
			}
		case map[string]interface{}:
			if len(v) == 0 {
				delete(m, k)
			} else {
				removeEmptyOrZero(v)
			}
		case []map[string]interface{}:
			if len(v) == 0 {
				delete(m, k)
			} else {
				for i := range v {
					removeEmptyOrZero(v[i])
				}
			}
		default:
			// if the value is not one of the above types, check if it's zero
			if reflect.DeepEqual(
				v,
				reflect.Zero(reflect.TypeOf(v)).Interface(),
			) {
				delete(m, k)
			}
		}
	}
}

func (n *X12Node) JSON(excludeEmpty bool) ([]byte, error) {
	var jErr []error
	p, err := payloadFromX12(n)
	jErr = append(jErr, err)
	if excludeEmpty {
		removeEmptyOrZero(p)
	}
	pp, _ := json.MarshalIndent(p, "", "  ")
	jErr = append(jErr, err)
	return pp, errors.Join(jErr...)
}

func (n *X12Node) Payload() (
	payload map[string]any,
	err error,
) {
	switch n.Type {
	case SegmentNode:
		payload, err = payloadFromSegment(n)
	case LoopNode:
		payload, err = payloadFromLoop(n)
	case TransactionNode:
		payload, err = payloadFromLoop(n)
	case CompositeNode:
		payload, err = payloadFromComposite(n)
	default:
		return payload, newError(
			n,
			fmt.Errorf("unexpected node type %v", n.Type),
		)
	}
	return payload, err
}

// payloadFromSegment converts a segment into a map[string]interface{],
// with keys populated based on X12Spec.label and values based on
// element values. Single values vs lists are determined by
// X12Spec.IsRepeatable().
func payloadFromSegment(
	segment *X12Node,
) (payload map[string]any, err error) {
	var payloadErrs []error
	payload = map[string]any{}
	if segment.Spec == nil {
		return payload, fmt.Errorf("segment %s has no spec", segment.Path)
	}

	for i, childSpec := range segment.Spec.Structure {
		if childSpec.NotUsed() {
			continue
		}
		elemInd := i + 1
		if elemInd > len(segment.Children) {
			switch childSpec.Type {
			case ElementSpec:
				if childSpec.IsRepeatable() {
					payload[childSpec.Label] = []string{}
				} else {
					payload[childSpec.Label] = ""
				}
			case CompositeSpec:
				payload[childSpec.Label] = map[string]any{}
			}
			continue
		}

		segChild := segment.Children[elemInd]
		switch segChild.Type {
		case ElementNode:
			childVal := segChild.Value
			if len(childVal) == 0 {
				payload[childSpec.Label] = ""
			} else {
				payload[childSpec.Label] = childVal[0]
			}
		case RepeatElementNode:
			repVal := segChild.Value
			repVal = removeTrailingEmptyElements(repVal)
			payload[childSpec.Label] = repVal
		case CompositeNode:
			cmpPayload, cmpErr := payloadFromComposite(segChild)
			payloadErrs = append(payloadErrs, cmpErr)
			if allZero(cmpPayload) {
				payload[childSpec.Label] = map[string]any{}
			} else {
				payload[childSpec.Label] = cmpPayload
			}
		default:
			return payload, newError(
				segment,
				fmt.Errorf(
					"expected element or composite, got %s",
					segChild.Type,
				),
			)
		}
	}
	err = errors.Join(payloadErrs...)

	return payload, err
}

func (n *X12Node) MarshalJSON() (data []byte, err error) {
	payload, err := n.Payload()
	if err != nil {
		return data, err
	}
	return json.Marshal(payload)
}

func payloadFromLoop(
	loop *X12Node,
) (payload map[string]any, err error) {
	if loop.Spec == nil {
		if loop.Type == TransactionNode {

		}
		return payload, fmt.Errorf("loop %s has no spec", loop.Path)
	}
	payload = map[string]any{}
	var payloadErrs []error

	if loop.Type == LoopNode {
		for _, childSpec := range loop.Spec.Structure {
			if childSpec.NotUsed() {
				continue
			}
			label := childSpec.Label
			if childSpec.IsRepeatable() {
				payload[label] = []map[string]any{}
			} else {
				payload[label] = map[string]any{}
			}
		}
	} else if loop.Type == TransactionNode {
		for _, childSpec := range loop.Spec.Structure {
			if childSpec.NotUsed() {
				continue
			}
			label := childSpec.Label
			if childSpec.IsRepeatable() {
				payload[label] = []map[string]any{}
			} else {
				payload[label] = map[string]any{}
			}
		}
	}

	for _, child := range loop.Children {
		switch child.Type {
		case LoopNode:
			k := child.Spec.Label
			if child.Spec.IsRepeatable() {
				_, ok := payload[k]
				if !ok {
					var tmpval []map[string]any
					payload[k] = tmpval
				}
				v, ok := payload[k].([]map[string]any)
				if !ok {
					return payload, newError(
						child,
						fmt.Errorf(
							"expected []map[string]any, got %T",
							payload[k],
						),
					)
				}
				pv, err := payloadFromLoop(child)
				payloadErrs = append(payloadErrs, err)
				if !allZero(pv) {
					payload[k] = append(v, pv)
				}
			} else {
				pv, err := payloadFromLoop(child)
				if allZero(pv) {
					payload[k] = map[string]any{}
				} else {
					payload[k] = pv
				}
				payloadErrs = append(payloadErrs, err)
			}
		case SegmentNode:
			k := child.Spec.Label

			if child.Spec.IsRepeatable() {
				_, ok := payload[k]
				if !ok {
					var tmpval []map[string]any
					payload[k] = tmpval
				}
				val := payload[k]
				svv, ok := val.([]map[string]any)
				if !ok {
					return payload, newError(
						child,
						fmt.Errorf(
							"expected []map[string]any, got %T",
							payload[k],
						),
					)
				}
				pv, err := payloadFromSegment(child)
				payloadErrs = append(payloadErrs, err)
				if !allZero(pv) {
					payload[k] = append(svv, pv)
				}
			} else {
				pv, err := payloadFromSegment(child)
				if allZero(pv) {
					payload[k] = map[string]any{}
				} else {
					payload[k] = pv
				}
				payloadErrs = append(payloadErrs, err)
			}
		}
	}

	return payload, errors.Join(payloadErrs...)
}

func payloadFromComposite(composite *X12Node) (
	payload map[string]any,
	err error,
) {
	payload = map[string]any{}
	if composite.Spec == nil {
		return payload, fmt.Errorf(
			"[composite: %s] %w: CompositeSpec is nil",
			composite.Name,
			ErrInvalidElementSpecType,
		)
	}

	childCt := len(composite.Children)
	if childCt == 0 {
		return payload, nil
	}

	for i, elemSpec := range composite.Spec.Structure {
		if elemSpec.NotUsed() {
			continue
		}
		elemLabel := elemSpec.Label
		if i > len(composite.Children) {
			payload[elemLabel] = ""
			continue
		}
		elem := composite.Children[i]
		elemVal := elem.Value
		if len(elemVal) == 0 {
			payload[elemLabel] = ""
		} else {
			payload[elemLabel] = elemVal[0]
		}
	}
	return payload, err
}

// setPath updates the path of the current node. If the current node
// has a parent, the path will be the parent's path + the current node's
// name. Setting the path of the current node will also update the paths
// of sibling nodes
func (n *X12Node) setPath() error {
	var pathErrors []error
	nodeName := n.Name
	if n.Spec != nil {
		nodeName = n.Spec.Name
		n.Name = nodeName
	}
	if n.Parent == nil {
		// Avoid re-setting the path if the current root node appears
		// to be a TransactionNode or GroupNode, because the parents
		// of these nodes are detached on initial parsing, and on
		// transformation
		if n.Type != TransactionNode && n.Type != GroupNode {
			n.Path = x12PathSeparator + n.Name
		}
		pathErrors = append(pathErrors, n.setChildPaths())
		return errors.Join(pathErrors...)
	}
	pathErrors = append(pathErrors, n.Parent.setPath())
	basePath := strings.Split(n.Parent.Path, x12PathSeparator)
	basePath = append(basePath, n.Name)
	n.Path = strings.Join(basePath, x12PathSeparator)
	return errors.Join(pathErrors...)
}

// setChildPaths updates the paths of the current node's children,
// and recursively updates the paths of the children's children.
// If there are multiple immediate children with the same X12Node.Name,
// the child path will include the index of the child, relative to
// its siblings with the same name (ex: REF[0], REF[1], REF[2], ...)
func (n *X12Node) setChildPaths() error {
	var pathErrors []error
	basePath := strings.Split(n.Path, x12PathSeparator)
	childNameCount := make(map[string]int)
	childNameCurrentCt := make(map[string]int)

	for i := 0; i < len(n.Children); i++ {
		child := n.Children[i]
		childName := child.Name
		if childName == "" && child.Spec == nil {
			switch n.Type {
			case SegmentNode:
				if len(n.Children) == 1 {
					child.Name = n.Name
				} else {
					child.Name = fmt.Sprintf(
						"%s%02d",
						n.Name,
						len(n.Children)-1,
					)
				}
			case CompositeNode:
				child.Name = fmt.Sprintf("%s%02d", n.Name, len(n.Children)-1)
			}
			childName = child.Name
		}
		if child.Spec != nil {
			childName = child.Spec.Name
			child.Name = childName
		}
		childNameCount[childName]++
		childNameCurrentCt[childName] = 0
	}

	basePathSize := len(basePath)
	childPath := make([]string, basePathSize+1)
	copy(childPath, basePath)
	for i := 0; i < len(n.Children); i++ {
		child := n.Children[i]
		childName := child.Name
		if child.Type == TransactionNode {
			childName += fmt.Sprintf(
				"[%d]",
				childNameCurrentCt[childName],
			)
		} else if child.Type == GroupNode {
			childName += fmt.Sprintf(
				"[%d]",
				childNameCurrentCt[childName],
			)
		} else if childNameCount[childName] > 1 {
			childName += fmt.Sprintf(
				"[%d]",
				childNameCurrentCt[childName],
			)
		}
		childNameCurrentCt[child.Name]++
		childPath[basePathSize] = childName
		child.Path = strings.Join(childPath, x12PathSeparator)
		pathErrors = append(pathErrors, child.setChildPaths())
		child.Parent = n
	}
	return errors.Join(pathErrors...)
}

// AuthInfo returns `ISA02`
func (n *Message) AuthInfo() string {
	return n.headerElementValue(isaIndexAuthInfo)
}

// SetAuthInfo sets `ISA02`
func (n *Message) SetAuthInfo(value string) error {
	targetLength := isaLenAuthInfo
	if len(value) > targetLength {
		return fmt.Errorf(
			"string too long (length: %d, max: %d)",
			len(value),
			targetLength,
		)
	}
	paddedVal := fmt.Sprintf("%*s", targetLength, value)
	return n.header().Children[isaIndexAuthInfo].SetValue(paddedVal)
}

// SecurityInfoQualifier returns `ISA03`
func (n *Message) SecurityInfoQualifier() string {
	return n.headerElementValue(isaIndexSecurityInfoQualifier)
}

// SetSecurityInfoQualifier sets `ISA03`
func (n *Message) SetSecurityInfoQualifier(value string) error {
	targetLength := isaLenSecurityInfoQualifier
	if len(value) > targetLength {
		return fmt.Errorf(
			"string too long (length: %d, max: %d)",
			len(value),
			targetLength,
		)
	}
	paddedVal := fmt.Sprintf("%*s", targetLength, value)
	return n.header().Children[isaIndexSecurityInfoQualifier].SetValue(paddedVal)
}

// SecurityInfo returns `ISA04`
func (n *Message) SecurityInfo() string {
	return n.headerElementValue(isaIndexSecurityInfo)
}

// SetSecurityInfo sets `ISA04`
func (n *Message) SetSecurityInfo(value string) error {
	targetLength := isaLenSecurityInfo
	if len(value) > targetLength {
		return fmt.Errorf(
			"string too long (length: %d, max: %d)",
			len(value),
			targetLength,
		)
	}
	paddedVal := fmt.Sprintf("%*s", targetLength, value)
	return n.header().Children[isaIndexSecurityInfo].SetValue(paddedVal)
}

// SenderIdQualifier returns `ISA05`
func (n *Message) SenderIdQualifier() string {
	return n.headerElementValue(isaIndexSenderIdQualifier)
}

// SetSenderIdQualifier sets `ISA05`
func (n *Message) SetSenderIdQualifier(value string) error {
	targetLength := isaLenSenderIdQualifier
	if len(value) > targetLength {
		return fmt.Errorf(
			"string too long (length: %d, max: %d)",
			len(value),
			targetLength,
		)
	}
	paddedVal := fmt.Sprintf("%*s", targetLength, value)
	return n.header().Children[isaIndexSenderIdQualifier].SetValue(paddedVal)
}

// SenderId returns `ISA06`
func (n *Message) SenderId() string {
	return n.headerElementValue(isaIndexSenderId)
}

// SetSenderId sets `ISA06`
func (n *Message) SetSenderId(value string) error {
	targetLength := isaLenSenderId
	if len(value) > targetLength {
		return fmt.Errorf(
			"string too long (length: %d, max: %d)",
			len(value),
			targetLength,
		)
	}
	paddedVal := fmt.Sprintf("%*s", targetLength, value)
	return n.header().Children[isaIndexSenderId].SetValue(paddedVal)
}

// ReceiverIdQualifier returns `ISA07`
func (n *Message) ReceiverIdQualifier() string {
	return n.headerElementValue(isaIndexReceiverIdQualifier)
}

// SetReceiverIdQualifier sets `ISA07`
func (n *Message) SetReceiverIdQualifier(value string) error {
	targetLength := isaLenReceiverIdQualifier
	if len(value) > targetLength {
		return fmt.Errorf(
			"string too long (length: %d, max: %d)",
			len(value),
			targetLength,
		)
	}
	paddedVal := fmt.Sprintf("%*s", targetLength, value)
	return n.header().Children[isaIndexReceiverIdQualifier].SetValue(paddedVal)
}

// ReceiverId returns `ISA08`
func (n *Message) ReceiverId() string {
	return n.headerElementValue(isaIndexReceiverId)
}

// SetReceiverId sets `ISA08`
func (n *Message) SetReceiverId(value string) error {
	targetLength := isaLenReceiverId
	if len(value) > targetLength {
		return fmt.Errorf(
			"string too long (length: %d, max: %d)",
			len(value),
			targetLength,
		)
	}
	paddedVal := fmt.Sprintf("%*s", targetLength, value)
	return n.header().Children[isaIndexReceiverId].SetValue(paddedVal)
}

// Date returns `ISA09`
func (n *Message) Date() string {
	return n.headerElementValue(isaIndexDate)
}

// SetDate sets `ISA09`
func (n *Message) SetDate(value string) error {
	targetLength := isaLenDate
	if len(value) > targetLength {
		return fmt.Errorf(
			"string too long (length: %d, max: %d)",
			len(value),
			targetLength,
		)
	}
	paddedVal := fmt.Sprintf("%*s", targetLength, value)
	return n.header().Children[isaIndexDate].SetValue(paddedVal)
}

// Time returns `ISA10`
func (n *Message) Time() string {
	return n.headerElementValue(isaIndexTime)
}

// SetTime sets `ISA10`
func (n *Message) SetTime(value string) error {
	targetLength := isaLenTime
	if len(value) > targetLength {
		return fmt.Errorf(
			"string too long (length: %d, max: %d)",
			len(value),
			targetLength,
		)
	}
	paddedVal := fmt.Sprintf("%*s", targetLength, value)
	return n.header().Children[isaIndexTime].SetValue(paddedVal)
}

// SetElementSeparator sets the given rune as the element delimiter. When
// set, this value is stripped from all element values in the message.
func (n *Message) SetElementSeparator(value rune) error {
	n.ElementSeparator = value
	e := n.removeChar(value)
	return e
}

// Version returns `ISA12`
func (n *Message) Version() string {
	return n.headerElementValue(isaIndexVersion)
}

// SetVersion sets `ISA12`
func (n *Message) SetVersion(value string) error {
	targetLength := isaLenVersion
	if len(value) > targetLength {
		return fmt.Errorf(
			"string too long (length: %d, max: %d)",
			len(value),
			targetLength,
		)
	}
	paddedVal := fmt.Sprintf("%*s", targetLength, value)
	return n.header().Children[isaIndexVersion].SetValue(paddedVal)
}

// AckRequested returns `ISA14`
func (n *Message) AckRequested() string {
	return n.headerElementValue(isaIndexAckRequested)
}

// SetAckRequested sets `ISA14`
func (n *Message) SetAckRequested(value string) error {
	targetLength := isaLenAckRequested
	if len(value) != targetLength {
		return fmt.Errorf(
			"string does not meet target length (length: %d, target: %d)",
			len(value),
			targetLength,
		)
	}
	return n.header().Children[isaIndexAckRequested].SetValue(value)
}

// UsageIndicator returns `ISA15`
func (n *Message) UsageIndicator() string {
	return n.headerElementValue(isaIndexUsageIndicator)
}

// SetUsageIndicator sets `ISA15`
func (n *Message) SetUsageIndicator(value string) error {
	targetLength := isaLenUsageIndicator
	if len(value) != targetLength {
		return fmt.Errorf(
			"string does not meet target length (length: %d, target: %d)",
			len(value),
			targetLength,
		)
	}
	return n.header().Children[isaIndexUsageIndicator].SetValue(value)
}
