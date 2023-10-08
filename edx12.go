package edx12

import (
	"bytes"
	"container/list"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// NodeType indicates the type of X12Node
type NodeType uint

const (
	UnknownNode     NodeType = iota
	MessageNode              // Bounded by ISA/IEA
	GroupNode                // Bounded by GS/GE
	TransactionNode          // Bounded by ST/SE
	LoopNode
	SegmentNode
	CompositeNode     // A component/composite element
	RepeatElementNode // An ElementNode that is repeatable
	ElementNode
	RepeatCompositeNode
)

func (n NodeType) String() string {
	return [...]string{
		"",
		"Message",
		"Group",
		"TransactionSet",
		"Loop",
		"Segment",
		"Composite",
		"RepeatElement",
		"Element",
		"RepeatComposite",
	}[n]
}

func (n NodeType) MarshalText() ([]byte, error) {
	return []byte(n.String()), nil
}

var nodeTypeNames = map[NodeType]string{
	UnknownNode:         "",
	MessageNode:         "Message",
	GroupNode:           "Group",
	TransactionNode:     "TransactionSet",
	LoopNode:            "Loop",
	SegmentNode:         "Segment",
	CompositeNode:       "Composite",
	RepeatElementNode:   "RepeatElement",
	ElementNode:         "Element",
	RepeatCompositeNode: "RepeatComposite",
}

var nodeTypeValues = map[string]NodeType{
	"":                UnknownNode,
	"Message":         MessageNode,
	"Group":           GroupNode,
	"TransactionSet":  TransactionNode,
	"Loop":            LoopNode,
	"Segment":         SegmentNode,
	"Composite":       CompositeNode,
	"RepeatElement":   RepeatElementNode,
	"Element":         ElementNode,
	"RepeatComposite": RepeatCompositeNode,
}

func (n NodeType) MarshalJSON() ([]byte, error) {
	return json.Marshal(nodeTypeNames[n])
}

func (n *NodeType) UnmarshalJSON(b []byte) error {
	var nodeName string
	if err := json.Unmarshal(b, &nodeName); err != nil {
		return err
	}
	*n = nodeTypeValues[nodeName]
	return nil
}

// nodeChildTypes indicates which other NodeType(s) can be
// set as children of a given NodeType
var nodeChildTypes = map[NodeType][]NodeType{
	SegmentNode: {
		ElementNode,
		CompositeNode,
		RepeatElementNode,
	},
	LoopNode:            {SegmentNode, LoopNode},
	MessageNode:         {SegmentNode, GroupNode},
	GroupNode:           {SegmentNode, TransactionNode},
	TransactionNode:     {SegmentNode, LoopNode},
	CompositeNode:       {ElementNode},
	RepeatElementNode:   {ElementNode},
	RepeatCompositeNode: {CompositeNode},
	ElementNode:         {},
}

// NodeError is an error which provides context about an error by referencing
// the X12Node related to the error
type NodeError struct {
	Node *X12Node
	Err  error
	data nodeErrorData
}

func (e *NodeError) MarshalJSON() (data []byte, err error) {
	e.data = nodeErrorData{
		NodeType: e.Node.Type,
		NodeName: e.Node.Name,
		NodePath: e.Node.Path,
		Error:    e.Err.Error(),
	}
	if e.Node.spec != nil {
		e.data.NodeLabel = e.Node.spec.Label
	}
	return json.Marshal(e.data)
}

type nodeErrorData struct {
	NodeType  NodeType `json:"node_type,omitempty"`
	NodeName  string   `json:"node_name,omitempty"`
	NodePath  string   `json:"node_path,omitempty"`
	NodeLabel string   `json:"node_label,omitempty"`
	Error     string   `json:"error"`
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
	if e.Node.Type == SegmentNode && e.Node.index > 0 {
		_, _ = fmt.Fprintf(&b, "position: %d ", e.Node.index)
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

// newNodeError creates a new NodeError referencing the given X12Node
func newNodeError(node *X12Node, err error) error {
	return &NodeError{
		Node: node,
		Err:  err,
	}
}

// X12Node is a node in an X12 message tree
type X12Node struct {
	// Type indicates the type of X12 structure (transaction, segment,
	// element, etc)
	Type NodeType `json:"type"`
	// Name indicates the segment/loop/element/composite/etc x12 ID
	Name string `json:"name"`
	// Path is the full path of the node, including the node's name, relative
	// to the rest of the tree.
	Path string `json:"-"`
	// Occurrence is the number of times the node has occurred at this level,
	// based on spec, if set.
	Occurrence int `json:"occurrence,omitempty"`
	// Parent is the parent node of the current node
	Parent *X12Node `json:"-"`
	// Children consists of child X12 nodes
	Children []*X12Node `json:"children,omitempty"`
	spec     *X12Spec
	// Value is the value of an X12Node when its type is
	// either ElementNode or RepeatElementNode
	Value []string `json:"value,omitempty"`
	// index is the position of the node within the message, one-indexed.
	// Applies only to SegmentNode.
	index int
	// originalText is the string value of the node as it was
	// parsed when using UnmarshalText. Applies only to SegmentNode.
	originalText string
}

// Spec returns the X12Spec associated with the node
func (n *X12Node) Spec() *X12Spec {
	return n.spec
}

// unsetSpec nullifies the spec field for the node and all of its children.
// For TransactionNode instances, immediate children are replaced with
// all SegmentNode types in the underlying tree.
func (n *X12Node) unsetSpec() {
	n.spec = nil

	if n.Type == TransactionNode {
		n.Children = n.Segments()
		for _, c := range n.Children {
			c.unsetSpec()
			c.Parent = n
		}
		return
	}

	for ind, c := range n.Children {
		if ind == 0 && n.Type == SegmentNode {
			continue
		}
		c.unsetSpec()
	}
}

// elements returns a slice of all child nodes that are of type ElementNode
func (n *X12Node) elements() []*X12Node {
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
					child.Children[j].elements()...,
				)
			}
		}
	}
	return elementNodes
}

// convertElement converts X12Node.Value to the appropriate type for
// both ElementNode and RepeatElementNode, based on X12Spec.DataType.
//
// For ElementNode, a single value will be returned for:
//   - Numeric: int, or nil for an empty/blank value
//   - Decimal: float64, or nil for an empty/blank value
//   - Date: Converted to time.Time, then to a string using MarshalText(). If
//     the time.Time is a zero value, an empty string will be returned.
//   - Time: Converted to time.Time, then to a string using MarshalText(). If
//     the time.Time is a zero value, an empty string will be returned.
//   - String, Identifier, Binary, UnknownDataType: string
func (n *X12Node) convertElement() (value any, err error) {
	if n.spec == nil {
		return nil, newNodeError(n, ErrMissingSpec)
	}

	switch n.Type {
	case ElementNode:
		switch len(n.Value) {
		case 0:
			value, err = convertElement("", n.spec.DataType)
		case 1:
			value, err = convertElement(n.Value[0], n.spec.DataType)
		default:
			err = newNodeError(
				n,
				errors.New("non-repeating element has multiple values"),
			)
		}
	case RepeatElementNode:
		errs := make([]error, 0)
		rv := removeTrailingEmptyElements(n.Value)
		values := make([]any, len(rv))
		for ind, v := range rv {
			elementVal, e := convertElement(v, n.spec.DataType)
			errs = append(errs, e)
			values[ind] = elementVal
		}
		value = values
		err = errors.Join(errs...)
	default:
		err = fmt.Errorf("cannot convert non-element type '%s'", n.Type)
	}
	return value, err
}

// Validate validates the X12Node against its X12Spec
func (n *X12Node) Validate() error {
	v := &Validator{}
	return v.ValidateNode(n)
}

func (n *X12Node) childTypes() []NodeType {
	return nodeChildTypes[n.Type]
}

// Length returns the length of the node's value, for
// nodes which are either ElementNode or RepeatElementNode
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

// SetSpec applies the given X12Spec to this node. This may change the
// current NodeType. If a RepeatElementNode or ElementNode are set with
// a CompositeSpec, its type will change to CompositeNode, and the current
// values (if any) will be converted to child ElementNode nodes.
// This will be called for any child spec for any child node.
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
			if s.IsRepeatable() {
				return fmt.Errorf("repeat composites not currently supported")
			}
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

	if !sliceContains(specNodeTypes[s.Type], n.Type) {
		specErrors = append(
			specErrors,
			newNodeError(
				n, fmt.Errorf(
					"spec '%s' not allowed for node type '%s'",
					s.Type,
					n.Type,
				),
			),
		)
		return errors.Join(specErrors...)
	}

	switch s.Type {
	case ElementSpec:
		specErrors = append(specErrors, setElementSpec(n, s))
	case CompositeSpec:
		specErrors = append(specErrors, setCompositeSpec(n, s))
	case SegmentSpec:
		specErrors = append(specErrors, setSegmentSpec(n, s))
	case LoopSpec, TransactionSetSpec:
		specStructure := s.Structure
		n.spec = s
		for i := 1; i < len(specStructure); i++ {
			if i < len(n.Children) {
				childNode := n.Children[i]
				err := childNode.SetSpec(specStructure[i-1])
				specErrors = append(specErrors, err)
				n.Children[i] = childNode
			}
		}
		return errors.Join(specErrors...)
	default:
		return fmt.Errorf("unknown spec type '%s'", s.Type)
	}

	n.spec = s
	return errors.Join(specErrors...)
}

// SetValue sets the given value for the node, for ElementNode,
// RepeatElementNode or CompositeNode, overwriting any existing value(s) set.
// For CompositeNode, if an X12Spec is already set, an error will be returned
// if the list of values exceeds the length of the defined structure.
func (n *X12Node) SetValue(value ...string) error {
	var errs []error
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
		if n.spec != nil {
			maxLen = len(n.spec.Structure)
			if elementCt > maxLen {
				errs = append(
					errs, newNodeError(
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
					errs = append(errs, newNodeError(n, err))
					return errors.Join(errs...)
				}
				if err = n.append(newChildNode); err != nil {
					errs = append(errs, newNodeError(n, err))
					return errors.Join(errs...)
				}
				continue
			}
			if e := n.Children[i].SetValue(values[i]); e != nil {
				errs = append(errs, newNodeError(n.Children[i], e))
			}
		}
	default:
		errs = append(
			errs,
			fmt.Errorf("cannot set value for node type '%s'", n.Type),
		)
	}
	return errors.Join(errs...)
}

// Walk walks the X12Node tree, calling walkFunc for each node
func (n *X12Node) Walk(walkFunc walkNodeFunc) error {
	return walkNodes(n, walkFunc)
}

func (n *X12Node) SegmentsWithName(name string) []*X12Node {
	var segments []*X12Node
	_ = n.Walk(
		func(nx *X12Node) error {
			if nx.Type == SegmentNode && nx.Name == name {
				segments = append(segments, nx)
			}
			return nil
		},
	)
	return segments
}

// defaultObjValue creates a new X12Node with default values
// based on the spec, creating a more comprehensive 'skeleton'
// tree/nodes
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
		return defaultElementObj(x)
	}
	return n, err
}

// replace replaces the node at the given index with the given node.
// The paths of the current node's children will be updated, including
// the replacement node. The node being replaced will be detached
// from the current node, and its paths (including children) will
// be updated.
// It will return an error if the given node's NodeType is not present
// in childTypes()
func (n *X12Node) replace(index int, node *X12Node) error {
	if index < 0 || index >= len(n.Children) {
		return errors.New("index out of range")
	}

	if !sliceContains(n.childTypes(), node.Type) {
		return newNodeError(
			n,
			fmt.Errorf(
				"cannot append node of type %s (must be one of: %v)",
				node.Type, n.childTypes(),
			),
		)
	}

	currentNode := n.Children[index]
	currentNode.Parent = nil
	n.Children[index] = node
	node.Parent = n

	if node.Name == "" && node.spec == nil {
		if n.Type == SegmentNode {
			if node.Type == ElementNode && index == 0 {
				node.Name = n.Name
			} else {
				node.Name = fmt.Sprintf("%s%02d", n.Name, len(n.Children)-1)
			}
		} else if n.Type == CompositeNode {
			node.Name = fmt.Sprintf("%s%02d", n.Name, len(n.Children)-1)
		}
	} else if node.Name == "" && node.spec != nil {
		node.Name = node.spec.Name
	}

	if e := currentNode.setPath(); e != nil {
		return e
	}
	return n.setChildPaths()
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
		payload, err = payloadFromTransactionSet(n)
	case CompositeNode:
		payload, err = payloadFromComposite(n)
	case MessageNode, GroupNode:
		payload, err = payloadFromX12(n)
	case RepeatCompositeNode:
		err = fmt.Errorf("repeat composites not currently supported")
	default:
		return payload, newNodeError(
			n,
			fmt.Errorf("unexpected node type %v", n.Type),
		)
	}
	return payload, err
}

func (n *X12Node) removeChar(char rune, replaceChar rune) error {
	return n.Walk(charRemover(char, replaceChar))
}

func payloadFromTransactionSet(n *X12Node) (payload map[string]any, err error) {
	if n.spec != nil {
		return payloadFromLoop(n)
	}

	err = n.setChildPaths()
	if err != nil {
		return payload, err
	}
	payload = map[string]any{}
	payloadErrs := []error{}
	for _, c := range n.Children {
		p, e := x12PathPayload(c)
		payloadErrs = append(payloadErrs, e)
		childPath := strings.TrimPrefix(c.Path, n.Path)
		childPath = strings.TrimPrefix(childPath, x12PathSeparator)
		payload[childPath] = p
	}
	return payload, errors.Join(payloadErrs...)
}

// x12PathPayload is intended for use where an X12Spec may not be set for
// the given X12Node. Instead of using X12Spec.Label for keys,
// X12Node.Path will be used. If a child (or grandchild, etc) X12Node
// has an X12Spec assigned, this will revert back to using
// X12Spec.Label in those cases.
func x12PathPayload(n *X12Node) (payload map[string]any, err error) {
	payload = map[string]any{}
	parentPath := n.Path
	if n.Type == SegmentNode {
		for ind, c := range n.Children {
			if ind == 0 {
				continue
			}
			childPath := c.Path
			childPath = strings.TrimPrefix(childPath, parentPath)
			childPath = strings.TrimPrefix(childPath, x12PathSeparator)
			if c.Type == CompositeNode {
				values := []string{}
				for _, cc := range c.Children {
					values = append(values, cc.Value...)
				}
				payload[childPath] = values
			} else {
				if len(c.Value) == 0 {
					payload[childPath] = ""
				} else if len(c.Value) == 1 {
					payload[childPath] = c.Value[0]
				} else {
					payload[childPath] = c.Value
				}
			}
		}
		return payload, err
	}

	for _, c := range n.Children {
		if c.spec != nil {
			p, e := payloadFromX12(c)
			if e != nil {
				return payload, e
			}
			_, ok := payload[c.spec.Label]
			if ok {
				return payload, fmt.Errorf("duplicate path: %s", c.spec.Label)
			}
			payload[c.spec.Label] = p
			continue
		}
		childPath := c.Path
		childPath = strings.TrimPrefix(childPath, parentPath)
		_, ok := payload[childPath]
		if ok {
			return payload, fmt.Errorf("duplicate path: %s", childPath)
		}

		p, e := x12PathPayload(c)
		if e != nil {
			return payload, e
		}
		payload[childPath] = p
	}
	return payload, err
}

// setPath updates the path of the current node. If the current node
// has a parent, the path will be the parent's path + the current node's
// name. Setting the path of the current node will also update the paths
// of sibling nodes
func (n *X12Node) setPath() error {
	var pathErrors []error
	if n.spec != nil {
		n.Name = n.spec.Name
	}
	if n.Parent == nil {
		// Avoid re-setting the path if the current root node appears
		// to be a TransactionNode or GroupNode, because the parents
		// of these nodes are detached on initial parsing, and on
		// transformation
		switch n.Type {
		case TransactionNode, GroupNode:
			//
		default:
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

	childNameCount := make(map[string]int)
	childNameCurrentCt := make(map[string]int)

	for _, child := range n.Children {
		childName := child.Name
		if child.spec == nil {
			if child.Name == "" {
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
					child.Name = fmt.Sprintf(
						"%s%02d",
						n.Name,
						len(n.Children)-1,
					)
				}
			}
		} else {
			childName = child.spec.Name
			child.Name = childName
		}

		childNameCount[childName]++
		childNameCurrentCt[childName] = 0
	}
	basePath := strings.Split(n.Path, x12PathSeparator)
	basePathSize := len(basePath)
	childPath := make([]string, basePathSize+1)
	copy(childPath, basePath)
	for _, child := range n.Children {
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

func (n *X12Node) getRootNode() *X12Node {
	var previousNode *X12Node
	visited := make(map[*X12Node]bool)
	previousNode = n
	for previousNode != nil {
		if visited[previousNode] {
			log.Fatalf("cycle at %#v", previousNode)
			return nil
		}
		visited[previousNode] = true
		if previousNode.Parent == nil {
			return previousNode
		}
		previousNode = previousNode.Parent
	}
	return previousNode
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
			return node, newNodeError(
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
		if n.spec == nil {
			return nil, newNodeError(n, fmt.Errorf("no spec set"))
		}
		node = n.Children[index]
		if e := node.SetValue(value); e != nil {
			nodeErrors = append(nodeErrors, e)
		}
		return node, errors.Join(nodeErrors...)
	case SegmentNode:
		if n.spec == nil {
			return nil, newNodeError(
				n,
				fmt.Errorf("no spec set"),
			)
		}
		if e := n.Children[index].SetValue(value); e != nil {
			nodeErrors = append(nodeErrors, e)
		}
		return n.Children[index], errors.Join(nodeErrors...)
	case RepeatCompositeNode:
		return nil, fmt.Errorf("repeat composites not currently supported")
	default:
		return nil, newNodeError(n, fmt.Errorf("invalid node type %s", n.Type))

	}
}

// append adds the given node to X12Node.Children for the current node.
func (n *X12Node) append(node *X12Node) error {
	var nodeErrors []error

	if !sliceContains(n.childTypes(), node.Type) {
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
	if node.Name == "" && node.spec == nil {
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

	if node.Name == "" && node.spec != nil {
		node.Name = node.spec.Name
	}
	node.Parent = n
	n.Children = append(n.Children, node)
	return nil

}

// Segments returns a slice of all SegmentNode types in the tree.
func (n *X12Node) Segments() []*X12Node {
	segs := []*X12Node{}

	for _, child := range n.Children {
		if child.Type == SegmentNode {
			segs = append(segs, child)
		} else {
			segs = append(segs, child.Segments()...)
		}
	}
	return segs
}

// Format will return a string representation of the X12Node tree as an X12
// text message, using the given delimiters/separators
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
	case RepeatCompositeNode:
		var composites []string
		for _, c := range n.Children {
			composites = append(
				composites, c.Format(
					segmentTerminator,
					elementSeparator,
					repetitionSeparator,
					componentElementSeparator,
				),
			)
		}
		composites = removeTrailingEmptyElements(composites)
		return strings.Join(composites, repetitionSeparator)
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

// NewNode creates a new X12Node of the given type, name, and values.
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

// Message is the root node of an X12 message
type Message struct {
	Header           *X12Node `json:"header"`
	Trailer          *X12Node `json:"trailer"`
	functionalGroups []*FunctionalGroupNode
	segments         []*X12Node
	rawMessage       *RawMessage
	*X12Node
}

// RawMessage returns the RawMessage used to create the Message, if any
func (n *Message) RawMessage() *RawMessage {
	return n.rawMessage
}

// TransactionSets returned the nested TransactionSetNode objects under
// this message
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

func (n *X12Node) detectCycles() error {
	tv := &treeValidator{VisitedNodes: map[*X12Node]bool{}}
	return tv.Visit(n)
}

func (n *Message) Validate() error {
	v := NewValidator(context.Background(), n)
	return v.Validate()
}

func (n *Message) ValidateWithContext(ctx context.Context) error {
	v := NewValidator(ctx, n)
	return v.Validate()
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

// RepetitionSeparator returns `ISA11`
func (n *Message) RepetitionSeparator() rune {
	var r rune
	for _, c := range n.headerElementValue(isaIndexRepetitionSeparator) {
		r = c
		break
	}
	return r
}

func (n *Message) AuthInfoQualifier() string {
	return n.headerElementValue(isaIndexAuthInfoQualifier)
}

func (n *Message) SetAuthInfoQualifier(value string) error {
	targetLength := isaLenAuthInfoQualifier
	if len(value) > targetLength {
		return fmt.Errorf(
			"string too long (length: %d, max: %d)",
			len(value),
			targetLength,
		)
	}
	paddedVal := fmt.Sprintf("%*s", targetLength, value)
	return n.header().Children[isaIndexAuthInfoQualifier].SetValue(paddedVal)
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
func (n *Message) SetDate(value time.Time) error {
	return n.header().Children[isaIndexDate].SetValue(value.Format("060102"))
}

// Time returns `ISA10`
func (n *Message) Time() string {
	return n.headerElementValue(isaIndexTime)
}

// SetTime sets `ISA10`
func (n *Message) SetTime(value time.Time) error {
	return n.header().Children[isaIndexTime].SetValue(value.Format("1504"))
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

// SetRepetitionSeparator sets the given rune as the repetition delimiter.
// When set, this value is stripped from all element values in the message.
func (n *Message) SetRepetitionSeparator(value rune) error {
	currentSeparator := n.RepetitionSeparator()
	if e := n.removeChar(value, currentSeparator); e != nil {
		return e
	}
	return n.Header.Children[isaIndexRepetitionSeparator].SetValue(string(value))
}

// SetComponentElementSeparator sets `ISA16`. When set, this value is
// stripped from all element values in the message.
func (n *Message) SetComponentElementSeparator(value rune) error {
	currentSeparator := n.ComponentElementSeparator()
	if e := n.removeChar(value, currentSeparator); e != nil {
		return e
	}
	return n.Header.Children[isaIndexComponentElementSeparator].SetValue(string(value))
}

type FunctionalGroupNode struct {
	Header       *X12Node              `json:"-"`
	Trailer      *X12Node              `json:"-"`
	Transactions []*TransactionSetNode `json:"transactions"`
	Message      *Message              `json:"-"`
	*X12Node
}

//func (x *FunctionalGroupNode) MarshalJSON() (data []byte, err error) {
//	payload, err := x.Payload()
//}

func (x *FunctionalGroupNode) IdentifierCode() string {
	return x.Header.Children[gsIndexFunctionalIdentifierCode].Value[0]
}

func (x *FunctionalGroupNode) SenderCode() string {
	return x.Header.Children[gsIndexSenderCode].Value[0]
}

func (x *FunctionalGroupNode) ReceiverCode() string {
	return x.Header.Children[gsIndexReceiverCode].Value[0]
}

func (x *FunctionalGroupNode) Date() string {
	return x.Header.Children[gsIndexDate].Value[0]
}

func (x *FunctionalGroupNode) Time() string {
	return x.Header.Children[gsIndexTime].Value[0]
}

func (x *FunctionalGroupNode) ControlNumber() string {
	return x.Header.Children[gsIndexControlNumber].Value[0]
}

func (x *FunctionalGroupNode) ResponsibleAgencyCode() string {
	return x.Header.Children[gsIndexResponsibleAgencyCode].Value[0]
}

func (x *FunctionalGroupNode) Version() string {
	return x.Header.Children[gsIndexVersion].Value[0]
}

func (x *FunctionalGroupNode) NumTransactionSets() string {
	return x.Trailer.Children[geIndexNumberOfIncludedTransactionSets].Value[0]
}

func (x *FunctionalGroupNode) TrailerControlNumber() string {
	return x.Trailer.Children[geIndexControlNumber].Value[0]
}

func (x *FunctionalGroupNode) Validate() error {
	v := &Validator{}
	_ = v.validateFunctionalGroup(x)
	return v.Err()
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
) (segmentGroups [][]*X12Node, err error) {
	//var segmentGroups [][]*X12Node
	var lastSeen bool
	var currentGroup []*X12Node
	segmentGroups = [][]*X12Node{}

	for _, segment := range segments {
		if segment.Name == segmentHeaderId {
			if len(currentGroup) > 0 {
				return segmentGroups, newNodeError(
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
				return segmentGroups, newNodeError(
					segment, fmt.Errorf(
						"unexpected state- found segment Trailer %v, but not currently in a segment group (already have: %v)",
						segment.Name,
						segmentGroups,
					),
				)
			}
		} else {
			if lastSeen == true && segment.Name != segmentHeaderId && len(currentGroup) == 0 {
				return segmentGroups, err
			} else if len(currentGroup) > 0 {
				currentGroup = append(currentGroup, segment)
			}
		}
	}
	return segmentGroups, err
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
	segments           []*X12Node
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

func (txn *TransactionSetNode) Validate() error {
	v := &Validator{}
	_ = v.validateTransactionSet(txn)
	return v.Err()
}

// TransformWithContext takes a list of X12TransactionSetSpec objects
// (which will automatically include the default specs at the end), and
// chooses the first spec where the header X12Spec (the first X12Spec in
// X12TransactionSetSpec, which must be a SegmentSpec named ST).
// That spec is then used to transform and validate the transaction set.
// If it's already been transformed, it will be re-set to a flat list
// of child segments.
// If no spec is found, an error is returned.
// Finally, each segment is matched against each SegmentSpec and LoopSpec
// in the spec, and is used to transform the transaction set into a hierarchy
// of matching loops, segments, composites and elements.
// This is what enables the JSON output of each transaction set, with
// meaningful naming/labeling.
func (txn *TransactionSetNode) TransformWithContext(
	ctx context.Context, specs ...*X12TransactionSetSpec,
) error {
	var txnSpec *X12TransactionSetSpec

	if len(specs) == 0 {
		if txn.TransactionSpec != nil {
			txnSpec = txn.TransactionSpec
		} else {
			return fmt.Errorf("no specs provided")
		}
	}
	for _, s := range specs {
		headerSpec := s.headerSpec()
		match, _ := segmentMatchesSpec(txn.Header, headerSpec)
		if match {
			txnSpec = s
			break
		}
	}

	if txnSpec == nil {
		return fmt.Errorf("no matching spec found for transaction")
	}

	txn.unsetSpec()
	txn.TransactionSpec = txnSpec
	var validationErrors []error
	segmentQueue := &segmentDeque{segments: list.New()}

	for _, segment := range txn.segments {
		segmentQueue.Append(segment)
	}
	segmentCount := segmentQueue.segments.Len()

	loopVisitor := &loopTransformer{
		message:         txn.Group.Message,
		createdLoop:     txn.X12Node,
		LoopSpec:        txn.TransactionSpec.X12Spec,
		SegmentQueue:    segmentQueue,
		ctx:             ctx,
		matchValidCodes: true,
	}

	for _, childNode := range txn.Children {
		childNode.Parent = nil
	}
	txn.Children = []*X12Node{}

	// We set the parent to nil here and re-set it after transforming
	// the loop, to avoid re-setting the paths of the entire tree very
	// time we append a new loop or node. X12Node.append() will not
	// update the path of a TransactionNode or GroupNode if the parent
	// is nil, so each TransactionSet child hierarchy will still have
	// the correct base path, and the nil parent will prevent the
	// update from moving up the tree and down into all the other
	// transactions
	txnParent := txn.Parent
	txn.Parent = nil
	loopVisitor.Children = []*X12Node{}
	for _, childSpec := range loopVisitor.LoopSpec.Structure {
		if ctx.Err() != nil {
			break
		}
		if loopVisitor.SegmentQueue.Length() == 0 {
			break
		}
		childSpec.Accept(loopVisitor)
	}

	_, err := loopVisitor.createLoop()
	if err != nil {
		validationErrors = append(validationErrors, err)
	}

	validationErrors = append(validationErrors, loopVisitor.ErrorLog...)
	matchedSegments := txn.Segments()
	unmatchedSegments := []*X12Node{}

	lastSegment := segmentQueue.Pop()
	if lastSegment != nil {
		trailerSpec := txnSpec.trailerSpec()
		log.Printf(
			"setting trailer spec on last orphan segment %s",
			lastSegment.Name,
		)
		var addedSE bool
		if lastSegment.Name == seSegmentId && trailerSpec != nil {
			e := lastSegment.SetSpec(trailerSpec)
			if e == nil {
				txn.Children = append(
					txn.Children,
					lastSegment,
				)
				lastSegment.Parent = txn.X12Node
				addedSE = true
			}
		}
		if addedSE {
			matchedSegments = txn.Segments()
		} else {
			segmentQueue.Append(lastSegment)
		}
	}
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

		segErr := newNodeError(
			breakingSegment,
			fmt.Errorf("unable to match to a SegmentSpec"),
		)
		var sArr []interface{}
		for _, s := range remainingSegments {
			sArr = append(sArr, s.Children)
		}
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
				fmt.Errorf("remaining segments: %q", sArr),
			),
		)
	}
	rootNode := txn.X12Node.getRootNode()

	if rootNode == txn.X12Node {
		err = rootNode.setPath()
		validationErrors = append(validationErrors, err)
	}
	txn.Parent = txnParent
	return errors.Join(validationErrors...)
}

// Transform takes a flat list of Segment instances for the current
// TransactionSetNode, and maps them to
// the TransactionSetSpec structure, identifying which segments
// map to which SegmentSpec instances, and which LoopSpec instances those
// segments belong to. A TransactionSet is created, with the hierarchy
// of loop and Segment instances matched to the transaction set.

// Transform takes a list of X12TransactionSetSpec instances, and chooses the
// first one where
func (txn *TransactionSetNode) Transform(txnSpecs ...*X12TransactionSetSpec) error {
	return txn.TransformWithContext(context.Background(), txnSpecs...)
}

func (txn *TransactionSetNode) unsetSpec() {
	txn.X12Node.unsetSpec()
	txn.TransactionSpec = nil
	_ = txn.setChildPaths()
	txn.segments = make([]*X12Node, len(txn.Children))
	copy(txn.segments, txn.Children)
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

	for i, element := range node.Children {
		switch i {
		case hlIndexHierarchicalId:
			hlNode.id = element.Value[0]
		case hlIndexParentId:
			hlNode.parentId = element.Value[0]
		case hlIndexLevelCode:
			hlNode.levelCode = element.Value[0]
		case hlIndexChildCode:
			hlNode.childCode = element.Value[0]
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
		return parentLevels, newNodeError(
			transactionSetNode,
			fmt.Errorf(
				"expected node type to be %s, got %s",
				TransactionNode,
				transactionSetNode.Type,
			),
		)
	}

	hlSegments := transactionSetNode.SegmentsWithName(hlSegmentId)
	if len(hlSegments) == 0 {
		return parentLevels, nil
	}
	levels := map[string]*HierarchicalLevel{}

	for _, hlSegment := range hlSegments {
		hlNode, e := newHierarchicalLevel(hlSegment)
		if e != nil {
			return parentLevels, e
		}
		_, exists := levels[hlNode.id]
		if exists {
			return parentLevels, newNodeError(
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
				//return parentLevels, nil
				// TODO: uncomment
				return parentLevels, newNodeError(
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
				return parentLevels, newNodeError(
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

type NodeVisitor interface {
	Visit(node *X12Node) error
}

type treeValidator struct {
	VisitedNodes map[*X12Node]bool
}

func (v *treeValidator) Visit(node *X12Node) error {
	if v.VisitedNodes[node] {
		return fmt.Errorf("duplicate node %s already seen", node.Path)
	}
	v.VisitedNodes[node] = true
	return nil
}

type walkNodeFunc func(x *X12Node) error

func walkNodes(node *X12Node, walkFunc walkNodeFunc) (err error) {
	errs := []error{}

	errs = append(errs, walkFunc(node))

	for _, child := range node.Children {
		e := walkNodes(child, walkFunc)
		if e != nil {
			errs = append(errs, newNodeError(child, e))
		}
	}
	err = errors.Join(errs...)
	return err
}

// validateSegmentNode validates a segment node's child values against
// its SegmentSpec
func validateSegmentNode(x *X12Node) error {
	v := &Validator{}
	_ = v.validateSegment(x)
	return v.Err()
}

type elementValue struct {
	Type  NodeType
	Value []string
}

// getSegmentNodeElements returns a slice of elementValue structs created
// from X12Node.Children. Trailing empty elements are removed.
func getSegmentNodeElements(segmentNode *X12Node) []elementValue {
	segElements := make([]elementValue, len(segmentNode.Children))

	for i, child := range segmentNode.Children {
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

func findTransactionSpec(txn *TransactionSetNode) (
	*X12TransactionSetSpec,
	error,
) {
	_defaultReader.transactionSpecsMu.RLock()
	defer _defaultReader.transactionSpecsMu.RUnlock()
	for _, txnSpec := range _defaultReader.transactionSpecs {
		headerSpec := txnSpec.headerSpec()
		if headerSpec == nil {
			continue
		}
		match, _ := segmentMatchesSpec(txn.Header, headerSpec)
		if match {
			return txnSpec, nil
		}
	}
	return nil, nil
}

var ErrTransactionAborted = errors.New("aborted transaction")

func payloadFromX12(n *X12Node) (map[string]any, error) {
	payload := map[string]any{}
	var payloadErrs []error

	for _, childNode := range n.Children {
		switch childNode.Type {
		case MessageNode, GroupNode, TransactionNode:
			//
		default:
			if childNode.spec == nil {
				payloadErrs = append(
					payloadErrs,
					fmt.Errorf(
						"[name: %s] [path: %s] childNode.spec is nil",
						childNode.Name,
						childNode.Path,
					),
				)
				continue
			}
		}

		if childNode.spec != nil && childNode.spec.NotUsed() {
			continue
		}

		switch childNode.Type {
		case ElementNode, RepeatElementNode:
			v, e := childNode.convertElement()
			payloadErrs = append(payloadErrs, e)
			payload[childNode.spec.Label] = v
		case CompositeNode:
			childVal, _ := childNode.Payload()
			if allZero(childVal) {
				payload[childNode.spec.Label] = map[string]any{}
			} else {
				payload[childNode.spec.Label] = childVal
			}
		case SegmentNode:
			childVal, segErr := childNode.Payload()
			payloadErrs = append(payloadErrs, segErr)
			if allZero(childVal) {
				continue
			}
			childLabel := childNode.spec.Label

			if childNode.spec.IsRepeatable() {
				_, ok := payload[childLabel]
				if !ok {
					var tmpval []map[string]any
					payload[childLabel] = tmpval
				}
				val := payload[childLabel]
				payload[childLabel] = append(val.([]map[string]any), childVal)
			} else {
				payload[childNode.spec.Label] = childVal
			}
		case LoopNode:
			childVal, loopErr := childNode.Payload()
			payloadErrs = append(payloadErrs, loopErr)
			if allZero(childVal) {
				continue
			}
			childLabel := childNode.spec.Label
			if childNode.spec.IsRepeatable() {
				_, ok := payload[childLabel]
				if !ok {
					var tmpval []map[string]any
					payload[childLabel] = tmpval
				}
				val := payload[childLabel]
				svv := val.([]map[string]any)
				payload[childLabel] = append(svv, childVal)
			} else {
				payload[childNode.spec.Label] = childVal
			}
		case GroupNode, TransactionNode:
			var label string
			if childNode.Type == GroupNode {
				label = "functionalGroups"
			} else {
				label = "transactionSets"
			}
			_, ok := payload[label]
			if !ok {
				var tmpval []map[string]any
				payload[label] = tmpval
			}
			val := payload[label]
			gvv := val.([]map[string]any)
			groupVal, groupErr := childNode.Payload()
			payloadErrs = append(payloadErrs, groupErr)
			payload[label] = append(gvv, groupVal)
		default:
			return payload, newNodeError(
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

// payloadFromSegment converts a segment into a map[string]interface{],
// with keys populated based on X12Spec.label and values based on
// element values. Single values vs lists are determined by
// X12Spec.IsRepeatable().
func payloadFromSegment(
	segment *X12Node,
) (payload map[string]any, err error) {
	var payloadErrs []error
	payload = map[string]any{}
	if segment.spec == nil {
		return payload, fmt.Errorf("segment %s has no spec", segment.Path)
	}

	for i, childSpec := range segment.spec.Structure {
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
				if childSpec.IsRepeatable() {
					payloadErrs = append(
						payloadErrs,
						errors.New("composite is repeatable but not present"),
					)
					continue
				}
				payload[childSpec.Label] = map[string]any{}
			}
			continue
		}

		segChild := segment.Children[elemInd]
		switch segChild.Type {
		case ElementNode, RepeatElementNode:
			v, e := segChild.convertElement()
			if e != nil {
				payloadErrs = append(payloadErrs, newNodeError(segChild, e))
			}
			payload[childSpec.Label] = v
		case CompositeNode:
			cmpPayload, cmpErr := payloadFromComposite(segChild)
			payloadErrs = append(payloadErrs, cmpErr)
			if allZero(cmpPayload) {
				payload[childSpec.Label] = map[string]any{}
			} else {
				payload[childSpec.Label] = cmpPayload
			}
		default:
			return payload, newNodeError(
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
	// using this instead of a straight `json.Marshal` on the payload, because
	// repetition separators like > will be escaped, even if you use your
	// own encoder with HTML escaping disabled
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	err = enc.Encode(payload)
	return b.Bytes(), err
}

// convertElement converts a string value to a value of the given X12
// data type. Empty values return an empty string.
func convertElement(value string, dataType DataType) (v any, err error) {
	valueLen := len(value)
	if valueLen == 0 {
		switch dataType {
		case Numeric, Decimal:
			return nil, err
		default:
			return "", err
		}
	}

	switch dataType {
	case Numeric:
		v, err = strconv.Atoi(value)
	case Decimal:
		v, err = strconv.ParseFloat(value, 32)
	case Date:
		var dt time.Time
		dt, err = parseDate(value)
		if dt.IsZero() {
			v = ""
		} else {
			v, err = dt.MarshalText()
		}
		v = fmt.Sprintf("%s", v)
	case Time:
		var ts time.Time
		ts, err = parseTime(value)
		if ts.IsZero() {
			v = ""
		} else {
			v, err = ts.MarshalText()
		}
		v = fmt.Sprintf("%s", v)
	default:
		v = value
	}
	return v, err
}

func payloadFromLoop(
	loop *X12Node,
) (payload map[string]any, err error) {
	if loop.spec == nil {
		return payload, fmt.Errorf("loop %s has no spec", loop.Path)
	}
	payload = map[string]any{}
	var payloadErrs []error

	if loop.Type == LoopNode {
		for _, childSpec := range loop.spec.Structure {
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
		for _, childSpec := range loop.spec.Structure {
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
			k := child.spec.Label
			if child.spec.IsRepeatable() {
				var val []map[string]any
				_, ok := payload[k]
				if !ok {
					payload[k] = []map[string]any{}
				}
				val = payload[k].([]map[string]any)

				pv, err := payloadFromLoop(child)
				payloadErrs = append(payloadErrs, err)
				if !allZero(pv) {
					payload[k] = append(val, pv)
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
			k := child.spec.Label

			if child.spec.IsRepeatable() {
				var val []map[string]any
				_, ok := payload[k]
				if !ok {
					payload[k] = []map[string]any{}
				}
				val = payload[k].([]map[string]any)
				pv, err := payloadFromSegment(child)
				payloadErrs = append(payloadErrs, err)
				if !allZero(pv) {
					payload[k] = append(val, pv)
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
	payloadErrs := []error{}
	payload = map[string]any{}
	if composite.spec == nil {
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

	for i, elemSpec := range composite.spec.Structure {
		if elemSpec.NotUsed() {
			continue
		}
		elemLabel := elemSpec.Label
		if i > len(composite.Children) {
			payload[elemLabel] = ""
			continue
		}
		elem := composite.Children[i]
		elemVal, e := elem.convertElement()
		if e != nil {
			payloadErrs = append(payloadErrs, newNodeError(elem, e))
		}
		payload[elemLabel] = elemVal

	}
	err = errors.Join(payloadErrs...)
	return payload, err
}

func charRemover(char rune, replaceChar rune) walkNodeFunc {
	return func(n *X12Node) error {
		var b strings.Builder
		for i := 0; i < len(n.Value); i++ {
			val := n.Value[i]
			if !strings.ContainsRune(val, char) {
				continue
			}
			for _, r := range val {
				if r == char {
					b.WriteRune(replaceChar)
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

// parseDate parses a string into a time.Time value, using corresponding
// X12 date/time formats based on the length of the string.
func parseDate(value string) (v time.Time, err error) {
	switch len(value) {
	case 6:
		v, err = time.Parse("060102", value)
	case 8:
		v, err = time.Parse("20060102", value)
	case 0:
	default:
		err = fmt.Errorf("date value '%s' should be length 0, 6 or 8", value)
	}
	return v, err
}

// parseTime parses a string into a time.Time value, using corresponding
// X12 date/time formats based on the length of the string.
func parseTime(value string) (v time.Time, err error) {
	switch len(value) {
	case 4:
		v, err = time.Parse("1504", value)
	case 6:
		v, err = time.Parse("150405", value)
	case 7, 8:
		newVal := []rune(value)
		value = fmt.Sprintf("%s.%s", string(newVal[:6]), string(newVal[6:]))
		v, err = time.Parse("150405.99", value)
		return v, err
	case 0:
	default:
		err = fmt.Errorf(
			"time value '%s' should be length 0, 4 or 6, 7 or 8",
			value,
		)
	}
	return v, err
}

// Validate reads, transforms and validates the given X12 message
func Validate(data []byte) error {
	validationErrors := []error{}

	rawMessage, err := Read(data)
	validationErrors = append(validationErrors, err)

	msg, err := rawMessage.Message(context.Background())
	validationErrors = append(validationErrors, err)
	validationErrors = append(validationErrors, msg.Validate())
	return errors.Join(validationErrors...)
}

func setSegmentSpec(n *X12Node, s *X12Spec) error {
	specErrors := []error{}

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
			switch childSpec.Type {
			case ElementSpec, CompositeSpec:
				childNode := &X12Node{}
				specErrors = append(
					specErrors,
					childNode.SetSpec(childSpec),
				)
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
	return errors.Join(specErrors...)
}

func setCompositeSpec(n *X12Node, s *X12Spec) error {
	specErrors := []error{}
	if s.IsRepeatable() {
		return fmt.Errorf("repeat composites not currently supported")
	}
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
	return errors.Join(specErrors...)
}

func setElementSpec(n *X12Node, s *X12Spec) error {
	specErrors := []error{}
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
		return errors.Join(specErrors...)
	}

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
	}

	return errors.Join(specErrors...)
}

type MessageData struct {
	Payload map[string]any  `json:"payload"`
	Errors  []nodeErrorData `json:"errors"`
}

// defaultCompositeObj creates a new X12Node populated with default values
// where the X12Spec indicates Required
func defaultCompositeObj(x *X12Spec) (cmpNode *X12Node, err error) {
	if x.IsRepeatable() {
		return nil, errors.New("repeatable composites not currently supported")
	}
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
			elemDefault, e := defaultElementObj(childStruct)
			errs = append(errs, e)
			e = cmpNode.replace(i, elemDefault)
			errs = append(errs, e)
		}

	}
	return cmpNode, errors.Join(errs...)
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
	n, err = NewNode(SegmentNode, x.Name)
	if err != nil {
		return n, err
	}
	if err = n.SetSpec(x); err != nil {
		return n, err
	}
	errs := []error{}
	for i := 0; i < len(x.Structure); i++ {
		childSpec := x.Structure[i]
		if !childSpec.Required() {
			continue
		}
		defaultVal, e := childSpec.defaultObjValue()
		errs = append(errs, e)
		// offset by 1, as the spec structure doesn't include segment ID
		n.Children[i+1] = defaultVal

	}
	return n, errors.Join(errs...)
}

// defaultElementObj creates a new X12Node with type ElementNode from
// the given ElementSpec. X12Node.value is set to the default value for the
// element, if there is one. If there is no default value, an empty string
// is set as the first value - for RepeatElementNode, an empty array will
// be set.
func defaultElementObj(e *X12Spec) (
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
