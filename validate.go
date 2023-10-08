package edx12

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var (
	ErrRequiredElementMissing     = errors.New("missing required element")
	ErrRequiredCompositeMissing   = errors.New("missing required composite")
	ErrInvalidElementSpecType     = errors.New("unexpected type for element definition")
	ErrRequiredSegmentMissing     = errors.New("missing required segment")
	ErrElementHasInvalidValue     = errors.New("element has invalid value")
	ErrElementHasBadLength        = errors.New("bad element value length")
	ErrRequiredLoopMissing        = errors.New("missing required loop")
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

// Validator is used to validate an X12 message
type Validator struct {
	message  *Message
	errorLog map[*X12Node][]error
	ctx      context.Context
}

var ValidationError = errors.New("validation error")

func NewValidator(ctx context.Context, message *Message) *Validator {
	return &Validator{
		message:  message,
		errorLog: make(map[*X12Node][]error),
		ctx:      ctx,
	}
}

func (v *Validator) String() string {
	return errorTree(v.message.X12Node, v, "")
}
func (v *Validator) Validate() error {
	if v.ctx == nil {
		v.ctx = context.Background()
	}
	v.errorLog = make(map[*X12Node][]error)
	return v.ValidateWithContext(v.ctx)
}

// ValidateNode runs validation on the given X12Node
func (v *Validator) ValidateNode(n *X12Node) error {
	v.errorLog = make(map[*X12Node][]error)
	_ = v.validateNode(n)
	return v.Err()
}

// Err returns a wrapped list of all errors encountered for all nodes
func (v *Validator) Err() error {
	errs := []error{}
	for _, nv := range v.errorLog {
		errs = append(errs, nv...)
	}
	return errors.Join(errs...)
}

// ErrorList returns a flat list of all validation errors encountered
func (v *Validator) ErrorList() []NodeError {
	errs := []NodeError{}
	for n, nv := range v.errorLog {
		for _, e := range nv {
			ne := &NodeError{}
			if errors.As(e, &ne) {
				errs = append(errs, *ne)
			} else {
				errs = append(errs, NodeError{Node: n, Err: e})
			}
		}
	}
	return errs
}

func (v *Validator) ValidateWithContext(ctx context.Context) error {
	v.ctx = ctx
	_ = v.validateMessage(v.message)
	return v.Err()
}

// addError adds the given error to the error log, if it's not nil- in
// which case, it will be wrapped into a NodeError
func (v *Validator) addError(n *X12Node, err error) {
	if err != nil {
		if v.errorLog == nil {
			v.errorLog = make(map[*X12Node][]error)
		}
		_, ok := v.errorLog[n]
		if !ok {
			v.errorLog[n] = []error{}
		}
		ne := newNodeError(n, err)
		v.errorLog[n] = append(v.errorLog[n], ne)
	}
}

// validateMessage validates the given Message's envelope, and then
// validates its child nodes and groups
func (v *Validator) validateMessage(n *Message) error {
	v.addError(n.X12Node, n.X12Node.detectCycles())

	isaControlNum := n.ControlNumber()
	ieaControlNum := n.IEAControlNumber()
	if isaControlNum != ieaControlNum {
		v.addError(
			n.X12Node, fmt.Errorf(
				"%w: ISA control number %s does not match IEA control number %s",
				ErrInvalidInterchangeEnvelope,
				isaControlNum,
				ieaControlNum,
			),
		)
	}
	groupCount := len(n.functionalGroups)
	trailerCt, e := n.numberOfFunctionalGroups()
	v.addError(n.X12Node, e)
	if e == nil && trailerCt != groupCount {
		v.addError(
			n.X12Node, fmt.Errorf(
				"%w: expected %d functional groups, got %d",
				ErrInvalidInterchangeEnvelope,
				trailerCt,
				groupCount,
			),
		)
	}
	for _, fg := range n.functionalGroups {
		_ = v.validateFunctionalGroup(fg)
	}
	_ = v.validateNode(n.X12Node)
	return errors.Join(v.errorLog[n.X12Node]...)
}

// validateFunctionalGroup validates the given FunctionalGroup's envelope,
// and then validates its child nodes and transactions
func (v *Validator) validateFunctionalGroup(x *FunctionalGroupNode) error {
	idCode := x.IdentifierCode()
	allowedTxnCodes := []string{}
	for t, i := range functionalIdentifierCodes {
		if i == idCode {
			allowedTxnCodes = append(allowedTxnCodes, t)
		}
	}

	transactionSetCt := len(x.Transactions)

	trailer := x.Trailer
	if trailer.Name != geSegmentId {
		v.addError(
			x.X12Node, fmt.Errorf(
				"%w: last segment in transaction must be GE, got %s",
				ErrInvalidGroupEnvelope,
				trailer.Name,
			),
		)
	} else {
		geSegmentCt := x.NumTransactionSets()
		geSegmentCtValue, e := strconv.Atoi(geSegmentCt)
		v.addError(x.X12Node, e)
		if e == nil && geSegmentCtValue != transactionSetCt {
			v.addError(
				x.X12Node, fmt.Errorf(
					"%w: expected %d transaction sets from GE trailer count, got %d",
					ErrInvalidGroupEnvelope,
					geSegmentCtValue,
					transactionSetCt,
				),
			)
		}

		gsControlNum := x.ControlNumber()
		geControlNum := x.TrailerControlNumber()
		if gsControlNum != geControlNum {
			v.addError(
				x.X12Node, fmt.Errorf(
					"%w: GS control number %s does not match GE control number %s",
					ErrInvalidGroupEnvelope,
					gsControlNum,
					geControlNum,
				),
			)
		}
	}
	_ = v.validateNode(x.X12Node)
	for _, txn := range x.Transactions {
		_ = v.validateTransactionSet(txn)
	}
	return errors.Join(v.errorLog[x.X12Node]...)
}

// validateTransactionSet validates the given TransactionSet's envelope,
// and then validates its child nodes
func (v *Validator) validateTransactionSet(txn *TransactionSetNode) error {
	segmentCt := len(txn.segments)
	trailer := txn.Trailer

	seSegmentCt := trailer.Children[seIndexNumberOfIncludedSegments]
	seSegmentCtValue, e := strconv.Atoi(seSegmentCt.Value[0])
	v.addError(trailer.Children[seIndexNumberOfIncludedSegments], e)

	if e == nil && seSegmentCtValue != segmentCt {
		v.addError(
			txn.X12Node, fmt.Errorf(
				"%w: expected %d segments from %s trailer count, got %d",
				ErrInvalidTransactionEnvelope,
				seSegmentCtValue,
				seSegmentId,
				segmentCt,
			),
		)
	}

	header := txn.Header
	stControlNum := header.Children[stIndexControlNumber].Value[0]
	seControlNum := trailer.Children[seIndexControlNumber].Value[0]

	if stControlNum != seControlNum {
		v.addError(
			txn.X12Node, fmt.Errorf(
				"%w: ST control number %s does not match SE control number %s",
				ErrInvalidTransactionEnvelope,
				stControlNum,
				seControlNum,
			),
		)
	}

	_ = v.validateLoop(txn.X12Node)
	return errors.Join(v.errorLog[txn.X12Node]...)
}

// validateNode validates the given X12Node, and then validates its
// child nodes
func (v *Validator) validateNode(n *X12Node) error {
	if n.spec == nil {
		// MessageNode and GroupNode nodes have specs set on their
		// header and trailer, but not their immediate children (with
		// exception for TransactionNode nodes) - so we cut out early
		if n.Type == MessageNode || n.Type == GroupNode {
			for _, c := range n.Children {
				if c.Type == TransactionNode {
					continue
				}
				_ = v.validateNode(c)
			}
			return v.getNodeErrors(n)
		}
		v.addError(n, ErrMissingSpec)
		return v.getNodeErrors(n)
	}

	if n.spec.NotUsed() && n.Length() > 0 {
		v.addError(n, ErrNotUsed)
	} else if n.spec.Required() && n.Length() == 0 {
		var err error
		switch n.spec.Type {
		case ElementSpec:
			err = ErrRequiredElementMissing
		case CompositeSpec:
			err = ErrRequiredCompositeMissing
		case SegmentSpec:
			err = ErrRequiredSegmentMissing
		case LoopSpec:
			err = ErrRequiredLoopMissing
		default:
			err = errors.New("missing required node")
		}
		v.addError(n, err)
	}

	switch n.Type {
	case SegmentNode:
		_ = v.validateSegment(n)
	case ElementNode, RepeatElementNode:
		_ = v.validateElement(n)
	case CompositeNode:
		_ = v.validateComposite(n)
	case LoopNode:
		_ = v.validateLoop(n)
	}
	return v.getNodeErrors(n)
}

// getNodeErrors returns a wrapped error of all errors encountered
// for the given X12Node
func (v *Validator) getNodeErrors(n *X12Node) error {
	return errors.Join(v.errorLog[n]...)
}

func (v *Validator) errorTree() string {
	return errorTree(v.message.X12Node, v, "")
}

// childNodeErrors returns a map of all errors encountered for the given
// node's child nodes (recursively)
func childNodeErrors(n *X12Node, v *Validator) map[*X12Node][]error {
	errs := make(map[*X12Node][]error)
	for _, c := range n.Children {
		e, ok := v.errorLog[n]
		if ok {
			errs[c] = e
		}
		childErrs := childNodeErrors(c, v)
		for k, cv := range childErrs {
			errs[k] = cv
		}
	}
	return errs
}

// hasChildErrors returns true if the given node has any errors in any
// of its child nodes (recursively)
func hasChildErrors(n *X12Node, v *Validator) bool {
	errs := childNodeErrors(n, v)
	return len(errs) > 0
}

func errorTree(n *X12Node, v *Validator, indent string) string {
	var b strings.Builder

	if n.spec == nil {
		b.WriteString(fmt.Sprintf("%s- %s\n", indent, n.Name))
	} else {
		b.WriteString(
			fmt.Sprintf(
				"%s- %s (%s)\n",
				indent,
				n.Name,
				n.spec.Label,
			),
		)
	}

	errs := v.errorLog[n]
	if len(errs) == 0 && !hasChildErrors(n, v) {
		return b.String()
	}
	for _, err := range errs {
		ne := &NodeError{}
		if errors.As(err, &ne) {
			b.WriteString(fmt.Sprintf("%s  !! %s\n", indent, ne.Err.Error()))
		} else {
			b.WriteString(fmt.Sprintf("%s  !! %s\n", indent, err.Error()))
		}
	}

	for _, child := range n.Children {
		switch child.Type {
		case MessageNode, GroupNode, TransactionNode, LoopNode:
			//b.WriteString(child.treeString(indent + "  "))
			bs := errorTree(child, v, indent+"  ")
			if bs != "" {
				b.WriteString(bs)
			}
		case SegmentNode, CompositeNode:
			if len(v.errorLog[child]) > 0 || hasChildErrors(child, v) {
				b.WriteString(errorTree(child, v, indent+"  "))
			}
		default:
			if len(v.errorLog[child]) > 0 {
				b.WriteString(errorTree(child, v, indent+"  "))
			}
		}
	}
	return b.String()
}

// validateSegment validates SegmentNode X12Node objects
func (v *Validator) validateSegment(x *X12Node) error {
	segmentSpec := x.spec

	if len(x.Children) == 0 {
		return fmt.Errorf("segment %s has no elements", segmentSpec.Name)
	}
	segIdElement := x.Children[0]

	if segIdElement.Length() == 0 {
		v.addError(
			x, fmt.Errorf(
				"[segment: %s] first element value not set",
				segmentSpec.Name,
			),
		)
	} else if segmentSpec == nil && segIdElement.Value[0] != x.Name {
		v.addError(
			x, fmt.Errorf(
				"[segment: %s] expected first element value '%s' to match segment id '%s'",
				x.Name,
				segIdElement.Value[0],
				x.Name,
			),
		)
	} else if segIdElement.Value[0] != segmentSpec.Name {
		v.addError(
			x, fmt.Errorf(
				"[segment: %s] expected first element value '%s' to match segment id '%s'",
				segmentSpec.Name,
				segIdElement.Value[0],
				segmentSpec.Name,
			),
		)
	}

	if segmentSpec == nil {
		v.addError(x, ErrMissingSpec)
		return v.getNodeErrors(x)
	}

	numElementSpecs := len(segmentSpec.Structure)
	elements := x.Children[1:]
	numElements := len(elements)
	if numElements > numElementSpecs {
		v.addError(
			x, fmt.Errorf(
				"spec defines %d elements, segment has %d",
				numElementSpecs, numElements,
			),
		)
		elements = elements[:numElementSpecs]
		numElements = len(elements)
	}

	for i := 0; i < len(segmentSpec.Structure); i++ {
		_ = v.validateNode(elements[i])
	}
	for _, condition := range segmentSpec.Syntax {
		err := validateSegmentSyntax(x, &condition)
		v.addError(x, err)
	}

	return v.getNodeErrors(x)
}

func (v *Validator) validateComposite(composite *X12Node) error {
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

	if composite.spec.Situational() && len(componentElements) == 0 {
		return nil
	}

	numElementSpecs := len(composite.spec.Structure)
	numElements := len(componentElements)
	if numElements > numElementSpecs {
		v.addError(
			composite,
			fmt.Errorf(
				"composite %s has %d elements, but only %d are defined in the schema",
				composite.spec.Name,
				numElements,
				numElementSpecs,
			),
		)
		componentElements = componentElements[:numElementSpecs]
		numElements = len(componentElements)
	}

	for i, x12def := range composite.spec.Structure {
		if i >= numElements {
			if x12def.Required() {
				v.addError(
					composite,
					fmt.Errorf(
						"composite %s is missing required element %s (path: %s) (description: %s)",
						composite.spec.Name,
						x12def.Name,
						x12def.path,
						x12def.Description,
					),
				)
			}
			continue
		} else {
			_ = v.validateElement(composite.Children[i])
		}
	}

	return v.getNodeErrors(composite)
}

// validateLoop validates LoopNode X12Node objects, and all child loops/segments
func (v *Validator) validateLoop(loop *X12Node) error {
	if loop.spec == nil {
		if loop.Type == MessageNode || loop.Type == GroupNode {
			for _, c := range loop.Children {
				_ = v.validateNode(c)
			}
			return v.getNodeErrors(loop)
		} else {
			return fmt.Errorf("loop has no spec")
		}
	}
	loopSpec := loop.spec

	loopSpecCount := make(map[*X12Spec]int)
	segmentSpecCount := make(map[*X12Spec]int)

	for ind, x12obj := range loop.Children {
		if x12obj.spec == nil {
			continue
		}
		switch x12obj.Type {
		case SegmentNode:
			sSpec := x12obj.spec
			segmentSpecCount[sSpec]++
		case LoopNode:
			lSpec := x12obj.spec
			loopSpecCount[lSpec]++
		default:
			v.addError(
				loop,
				fmt.Errorf(
					"expected segment or loop at %d, got %v",
					ind,
					x12obj.Type,
				),
			)
			return v.getNodeErrors(loop)
		}
	}

	for loopSpecPtr, ct := range loopSpecCount {
		if loopSpecPtr.RepeatMin != 0 && ct < loopSpecPtr.RepeatMin {
			v.addError(
				loop,
				fmt.Errorf(
					"[loop: %s] [path: %s]: %w (min: %d found: %d)",
					loopSpecPtr.Name,
					loopSpecPtr.path,
					ErrTooFewLoops,
					loopSpecPtr.RepeatMin,
					ct,
				),
			)
		} else if loopSpecPtr.RepeatMax != 0 && ct > loopSpecPtr.RepeatMax {
			v.addError(
				loop,
				fmt.Errorf(
					"[loop: %s] [path: %s]: %w (max: %d found: %d)",
					loopSpecPtr.Name,
					loopSpecPtr.path,
					ErrTooManyLoops,
					loopSpecPtr.RepeatMax,
					ct,
				),
			)

		}
	}

	for segmentSpecPtr, ct := range segmentSpecCount {
		if segmentSpecPtr.RepeatMin != 0 && ct < segmentSpecPtr.RepeatMin {
			v.addError(
				loop,
				fmt.Errorf(
					"[segment: %s] [path: %s]: %w (min: %d found: %d)",
					segmentSpecPtr.Name,
					segmentSpecPtr.path,
					ErrTooFewSegments,
					segmentSpecPtr.RepeatMin,
					ct,
				),
			)
		} else if segmentSpecPtr.RepeatMax != 0 && ct > segmentSpecPtr.RepeatMax {
			v.addError(
				loop,
				fmt.Errorf(
					"[segment: %s] [path: %s]: %w (max: %d found: %d)",
					segmentSpecPtr.Name,
					segmentSpecPtr.path,
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
					v.addError(
						loop,
						fmt.Errorf(
							"[segment: %s] [path: %s]: %w",
							x12def.Name,
							x12def.path,
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
					v.addError(
						loop,
						fmt.Errorf(
							"[loop: %s] [description: %s] [path: %s]: %w",
							x12def.Name,
							x12def.Description,
							x12def.path,
							ErrRequiredLoopMissing,
						),
					)
					continue
				}
			}
		}
	}

	for _, loopChild := range loop.Children {
		_ = v.validateNode(loopChild)
	}
	return v.getNodeErrors(loop)
}

func (v *Validator) validateElement(element *X12Node) error {
	if element.Type != ElementNode && element.Type != RepeatElementNode {
		v.addError(
			element,
			fmt.Errorf("expected element, got %s", element.Type),
		)
		return v.getNodeErrors(element)
	}
	if element.spec == nil {
		v.addError(element, ErrMissingSpec)
		return v.getNodeErrors(element)
	}

	if element.spec.IsRepeatable() && element.Type != RepeatElementNode {
		v.addError(
			element,
			fmt.Errorf("expected repeat element, got %s", element.Type),
		)
	} else if !element.spec.IsRepeatable() && element.Type != ElementNode {
		v.addError(
			element,
			fmt.Errorf("expected element, got %s", element.Type),
		)
	}

	switch element.Type {
	case ElementNode:
		elemVal := ""
		if len(element.Value) == 1 {
			elemVal = element.Value[0]
		}
		elementErrors := validateElementValue(element.spec, elemVal)
		v.addError(element, elementErrors)
	case RepeatElementNode:
		ev := element.Value
		if element.spec.IsRepeatable() {
			if element.spec.RepeatMax != 0 && len(ev) > element.spec.RepeatMax {
				v.addError(
					element, fmt.Errorf(
						"element %s has too many occurrences, max %d",
						element.spec.Name,
						element.spec.RepeatMax,
					),
				)
			}
			if element.spec.RepeatMin != 0 && len(ev) < element.spec.RepeatMin {
				v.addError(
					element, fmt.Errorf(
						"element %s has too few occurrences, min %d",
						element.spec.Name,
						element.spec.RepeatMin,
					),
				)
			}
		}
		for i, repElem := range ev {
			elementErrors := validateElementValue(element.spec, repElem)
			if elementErrors != nil {
				v.addError(
					element, fmt.Errorf(
						"element %s, Occurrence %d: %s",
						element.spec.Name,
						i+1,
						elementErrors,
					),
				)
			}
		}
	}
	return v.getNodeErrors(element)
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
				"%w: when at least one value in %+v is present, all must be present",
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
			"%w: at least one of %+v is required",
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
				"%w: only one of %+v is allowed",
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
							"%w: when element %d is present, all must be present",
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
					"%w: if a value is present in element %d, at least one "+
						"value must be present in element %d",
					SegmentConditionErr,
					condition.Indexes[0],
					condition.Indexes[1],
				)
			}
		}
	}
	return nil
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
	if element.spec == nil {
		validationErrors = append(
			validationErrors,
			newNodeError(element, fmt.Errorf("nil spec")),
		)
		return errors.Join(validationErrors...)
	}

	if element.spec.IsRepeatable() && element.Type != RepeatElementNode {
		validationErrors = append(
			validationErrors,
			fmt.Errorf("expected repeat element, got %s", element.Type),
		)
	} else if !element.spec.IsRepeatable() && element.Type != ElementNode {
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
		elementErrors := validateElementValue(element.spec, elemVal)
		validationErrors = append(validationErrors, elementErrors)
	case RepeatElementNode:
		v := element.Value
		if element.spec.IsRepeatable() {
			if element.spec.RepeatMax != 0 && len(v) > element.spec.RepeatMax {
				validationErrors = append(
					validationErrors,
					fmt.Errorf(
						"element %s has too many occurrences, max %d",
						element.spec.Name,
						element.spec.RepeatMax,
					),
				)
			}
			if element.spec.RepeatMin != 0 && len(v) < element.spec.RepeatMin {
				validationErrors = append(
					validationErrors,
					fmt.Errorf(
						"element %s has too few occurrences, min %d",
						element.spec.Name,
						element.spec.RepeatMin,
					),
				)
			}
		}
		for i, repElem := range v {
			elementErrors := validateElementValue(element.spec, repElem)
			if elementErrors != nil {
				validationErrors = append(
					validationErrors,
					fmt.Errorf(
						"element %s, Occurrence %d: %s",
						element.spec.Name,
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
	if element == "" {
		return nil
	}

	var validationErrors []error
	// If a value has been provided and validCodes is defined, check that
	// it's in the allowed list
	if len(spec.ValidCodes) > 0 && !sliceContains(spec.ValidCodes, element) {
		validationErrors = append(
			validationErrors,
			fmt.Errorf(
				"[element: %s] [path: %s]: %w (got: %v) (valid values: %+v)",
				spec.Name,
				spec.path,
				ErrElementHasInvalidValue,
				element,
				spec.ValidCodes,
			),
		)
	}

	elementLength := len(element)
	if spec.MaxLength != 0 && elementLength > spec.MaxLength {
		validationErrors = append(
			validationErrors,
			fmt.Errorf(
				"[element: %s] [path: %s]: %w (too long - max length is %d)",
				spec.Name,
				spec.path,
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
				spec.path,
				ErrElementHasBadLength,
				spec.MinLength,
			),
		)
	}
	return errors.Join(validationErrors...)
}
