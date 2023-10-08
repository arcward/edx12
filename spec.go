package edx12

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

var (
	pathSeparator  = "/"
	specChildTypes = map[SpecType][]SpecType{
		TransactionSetSpec: {LoopSpec, SegmentSpec},
		LoopSpec:           {LoopSpec, SegmentSpec},
		SegmentSpec:        {ElementSpec, CompositeSpec},
		CompositeSpec:      {ElementSpec},
		ElementSpec:        {},
	}
	specNodeTypes = map[SpecType][]NodeType{
		TransactionSetSpec: {TransactionNode},
		LoopSpec:           {LoopNode, TransactionNode},
		SegmentSpec:        {SegmentNode},
		ElementSpec:        {ElementNode, RepeatElementNode},
		CompositeSpec:      {CompositeNode, RepeatElementNode, ElementNode},
	}
)

type SpecError struct {
	Spec *X12Spec
	Err  error
}

func (e *SpecError) Error() string {
	var b strings.Builder

	if specName := e.Spec.Name; specName != "" {
		_, _ = fmt.Fprintf(&b, "name: %s ", specName)
	}
	if specPath := e.Spec.path; specPath != "" {
		_, _ = fmt.Fprintf(&b, "path: %s ", specPath)
	}
	if specLabel := e.Spec.Label; specLabel != "" {
		_, _ = fmt.Fprintf(&b, "label: %s ", specLabel)
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

func (e *SpecError) Unwrap() error {
	return e.Err
}

// newSpecErr creates a new SpecError referencing the given X12Spec
func newSpecErr(e error, spec *X12Spec) error {
	return &SpecError{
		Spec: spec,
		Err:  e,
	}
}

type DataType uint

const (
	UnknownDataType DataType = iota
	String
	Numeric
	Identifier
	Decimal
	Date
	Time
	Binary
)

func (d DataType) String() string {
	names := map[DataType]string{
		UnknownDataType: "",
		String:          "AN",
		Numeric:         "N",
		Identifier:      "ID",
		Decimal:         "R",
		Date:            "DT",
		Time:            "TM",
		Binary:          "B",
	}
	return names[d]
}

func (d DataType) GoString() string {
	s := map[DataType]string{
		UnknownDataType: "UnknownDataType",
		String:          "String",
		Numeric:         "Numeric",
		Identifier:      "Identifier",
		Decimal:         "Decimal",
		Date:            "Date",
		Time:            "Time",
		Binary:          "Binary",
	}
	return s[d]
}

func (d DataType) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *DataType) UnmarshalJSON(b []byte) error {
	var typeName string

	if err := json.Unmarshal(b, &typeName); err != nil {
		return err
	}
	switch strings.ToLower(typeName) {
	case "":
		*d = UnknownDataType
	case "an":
		*d = String
	case "n":
		*d = Numeric
	case "id":
		*d = Identifier
	case "r":
		*d = Decimal
	case "dt":
		*d = Date
	case "tm":
		*d = Time
	case "b":
		*d = Binary
	default:
		return fmt.Errorf("unknown DataType: '%#v' / '%#v'", d, typeName)
	}
	return nil
}

type ConditionCode uint

const (
	UnknownConditionCode ConditionCode = iota
	PairedOrMultiple
	RequiredCondition
	Exclusion
	Conditional
	ListConditional
)

func (c ConditionCode) String() string {
	names := map[ConditionCode]string{
		UnknownConditionCode: "",
		PairedOrMultiple:     "P",
		RequiredCondition:    "R",
		Exclusion:            "E",
		Conditional:          "C",
		ListConditional:      "L",
	}
	return names[c]
}

func (c ConditionCode) GoString() string {
	s := map[ConditionCode]string{
		UnknownConditionCode: "UnknownConditionCode",
		PairedOrMultiple:     "PairedOrMultiple",
		RequiredCondition:    "RequiredCondition",
		Exclusion:            "Exclusion",
		Conditional:          "Conditional",
		ListConditional:      "ListConditional",
	}
	return s[c]
}

func (c ConditionCode) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.String())
}

func (c *ConditionCode) UnmarshalJSON(b []byte) error {
	var conditionName string

	if err := json.Unmarshal(b, &conditionName); err != nil {
		return err
	}
	switch conditionName {
	case "":
		*c = UnknownConditionCode
	case "P":
		*c = PairedOrMultiple
	case "R":
		*c = RequiredCondition
	case "E":
		*c = Exclusion
	case "C":
		*c = Conditional
	case "L":
		*c = ListConditional
	default:
		return fmt.Errorf("unknown ConditionCode: %s", conditionName)
	}
	return nil
}

type SpecType uint

const (
	UnknownSpec SpecType = iota
	TransactionSetSpec
	LoopSpec
	SegmentSpec
	CompositeSpec
	ElementSpec
)

func (s SpecType) String() string {
	names := map[SpecType]string{
		TransactionSetSpec: "TransactionSet",
		LoopSpec:           "Loop",
		SegmentSpec:        "Segment",
		CompositeSpec:      "Composite",
		ElementSpec:        "Element",
	}
	return names[s]
}

func (s SpecType) GoString() string {
	names := map[SpecType]string{
		TransactionSetSpec: "TransactionSetSpec",
		LoopSpec:           "LoopSpec",
		SegmentSpec:        "SegmentSpec",
		CompositeSpec:      "CompositeSpec",
		ElementSpec:        "ElementSpec",
	}
	return names[s]
}

func (s SpecType) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s *SpecType) UnmarshalJSON(b []byte) error {
	var nodeName string

	if err := json.Unmarshal(b, &nodeName); err != nil {
		return err
	}
	switch strings.ToLower(nodeName) {
	case "transactionset":
		*s = TransactionSetSpec
	case "loop":
		*s = LoopSpec
	case "segment":
		*s = SegmentSpec
	case "composite":
		*s = CompositeSpec
	case "element":
		*s = ElementSpec
	default:
		return fmt.Errorf("unknown SpecType: %s", nodeName)
	}
	return nil
}

type Usage int

const (
	UnknownUsage Usage = iota
	// Required indicates that the field is required
	Required
	// Situational indicates that the field is situational
	Situational
	// NotUsed indicates that the field is not used
	NotUsed
)

func (u Usage) GoString() string {
	names := map[Usage]string{
		UnknownUsage: "UnknownUsage",
		Required:     "Required",
		Situational:  "Situational",
		NotUsed:      "NotUsed",
	}
	return names[u]
}

func (u Usage) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.String())
}

func (u Usage) String() string {
	usageNames := map[Usage]string{
		Required:    "REQUIRED",
		Situational: "SITUATIONAL",
		NotUsed:     "NOT USED",
	}
	return usageNames[u]
}

func (u *Usage) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	switch s {
	case "REQUIRED":
		*u = Required
	case "SITUATIONAL":
		*u = Situational
	case "NOT USED":
		*u = NotUsed
	}
	return nil
}

type X12Spec struct {
	// Type indicates the type of the X12 structure this spec defines
	Type SpecType `json:"type"`
	// Name is the name of the field as it should appear in the x12 spec.
	// Ex: `2000A`, `NM1`, ...
	Name string `validate:"required,gte=1" json:"name"`
	// Usage indicates whether the field is required, situational, or not used.
	// If Type is ElementSpec, this may be situationally overridden based on
	// a parent SegmentSpec's SegmentCondition list.
	Usage       Usage  `json:"usage"`
	Description string `json:"description,omitempty"`
	// RepeatMin specifies the minimum number of times this structure should
	// repeat (when used).
	// If Usage is NotUsed, this has no effect.
	// If Usage is Situational, the number of repetitions must be zero, or
	// between this number and RepeatMax, inclusive (if RepeatMax > 0)
	// If Usage is Required and this is not set, it will be set/overriden as 1.
	// The number of repetitions must be between this
	// number and RepeatMax, inclusive (if RepeatMax > 0)
	RepeatMin int `json:"repeatMin,omitempty"`
	// RepeatMax is the maximum number of times this can repeat.
	// If Usage is NotUsed, this has no effect.
	// If Usage is Situational, the number of repetitions must be at or
	// below this number, and between either zero or RepeatMin, inclusive.
	// If Usage is Required, the number of repetitions must be between
	// RepeatMin and this number, inclusive. If RepeatMin isn't set, it must
	// be between one and this number.
	RepeatMax int `json:"repeatMax,omitempty"`
	// Label sets the key for X12Node objects when marshalling to JSON.
	Label string `json:"label,omitempty"`
	// Structure consists of the child X12Specs of this X12Spec
	Structure []*X12Spec `json:"structure,omitempty"`
	// ValidCodes (for ElementSpec), is a list of valid values for this
	// element
	ValidCodes []string `json:"validCodes,omitempty"`
	// MinLength (for ElementSpec) specifies the minimum length of the element
	// when a value is provided.
	// If Usage is NotUsed, this has no effect.
	// If Usage is Situational, the length of the element must either be zero,
	// or between this number and MaxLength, inclusive (if MaxLength > 0)
	// If Usage is Required and MinLength is not set, it will be interpreted
	// or overridden as 1. The length of the element must be between this
	// and MaxLength, inclusive (if MaxLength > 0).
	// For elements, this sets minLength in the generated JSON schema.
	MinLength int `json:"minLength,omitempty"`
	// MaxLength (for ElementSpec) specifies the maximum length of the element
	// when a value is provided.
	// If Usage is NotUsed, this has no effect.
	// For elements, this sets maxLength in the generated JSON schema.
	MaxLength int `json:"maxLength,omitempty"`
	// DefaultVal specifies the default ElementNode value when Usage
	// is Required, when generating messages with this spec.
	// If Usage is NotUsed or Situational, this has no effect. The value must
	// be present in ValidCodes, if not empty.
	// For Required elements, this sets 'default' in the generated JSON schema.
	DefaultVal string `json:"default,omitempty"`
	// DataType indicates the ASC X12 data type of this element.
	// When marshalling to JSON, this is used to convert element values to
	// the appropriate type.
	// For the generated JSON schema:
	// - Decimal sets type=number and format=float
	// - Numeric sets type=number
	// - All others set type=string
	DataType DataType `validate:"json:dataType,omitempty"`
	// Syntax describes relational conditions across elements in a segment.
	// Conditions defined here may situationally override child ElementSpec
	// settings.
	Syntax []SegmentCondition `json:"syntax,omitempty"`
	// loopDepth is the depth of this X12Spec in the overall spec tree,
	// or the depth of its parent loop
	loopDepth int
	// path is a materialized path of this X12Spec's position in
	// the overall spec
	path string
	// parent is the parent of this X12Spec, or nil if this is the root
	parent     *X12Spec
	jsonSchema *jsonSchema
}

// index returns the position of this X12Spec in its parent's Structure.
// If this X12Spec has no parent, -1 is returned. If this X12Spec is not
// found in its parent's Structure, -1 is returned along with an error.
func (x *X12Spec) index() (int, error) {
	if x.parent == nil {
		return -1, nil
	}
	ind := x.parent.childIndex(x)
	if ind == -1 {
		return -1, fmt.Errorf("X12Spec %s not found in parent", x.Name)
	}
	return ind, nil
}

// childIndex returns the position of the given child X12Spec in this
// X12Spec's Structure. If the child is not found, -1 is returned.
func (x *X12Spec) childIndex(childSpec *X12Spec) int {
	for i, child := range x.Structure {
		if child == childSpec {
			return i
		}
	}
	return -1
}

// nextSpec returns the next X12Spec in this X12Spec's parent's Structure,
// or nil if this is the last X12Spec in the parent's Structure.
// If this X12Spec isn't found in its parent's Structure, an error is returned.
func (x *X12Spec) nextSpec() (*X12Spec, error) {
	ind, err := x.index()
	if err != nil {
		return nil, err
	}
	if ind == -1 {
		return nil, nil
	}
	if ind+1 >= len(x.parent.Structure) {
		return nil, nil
	}
	return x.parent.Structure[ind+1], nil
}

func (x *X12Spec) findName(name string) (matchingSpecs []*X12Spec) {
	if x.Name == name {
		matchingSpecs = append(matchingSpecs, x)
	}
	for _, child := range x.Structure {
		matchingSpecs = append(matchingSpecs, child.findName(name)...)
	}
	return matchingSpecs
}

func (x *X12Spec) findType(specType SpecType) (matchingSpecs []*X12Spec) {
	if x.Type == specType {
		matchingSpecs = append(matchingSpecs, x)
	}
	for _, child := range x.Structure {
		matchingSpecs = append(matchingSpecs, child.findType(specType)...)
	}
	return matchingSpecs
}

// newSchema creates a new jsonSchema for this X12Spec and any child
// specs, and assigns it to X12Spec.schema
func (x *X12Spec) newSchema() *jsonSchema {
	if x.Usage == NotUsed {
		return nil
	}

	j := &jsonSchema{
		Description: x.Description,
	}

	if x.Type == ElementSpec {
		j.Title = x.Name
		switch x.DataType {
		case Decimal:
			j.Type = "number"
			j.Format = "float"
		case Numeric:
			j.Type = "number"
		case Date:
			j.Type = "string"
			switch x.MaxLength {
			case 8:
				j.Example = "20060102"
			case 6:
				j.Example = "200601"
			}
		case Time:
			j.Type = "string"
			switch x.MinLength {
			case 4:
				j.Example = "1504"
			case 6:
				j.Example = "150405"
			case 8:
				j.Example = "15040599"
			}
		default:
			j.Type = "string"
			if len(x.ValidCodes) > 0 {
				j.Enum = x.ValidCodes
			}
		}
		j.MinLength = x.MinLength
		j.MaxLength = x.MaxLength

		if x.Usage == Required {
			if x.DefaultVal != "" {
				j.Default = x.DefaultVal
			}
		} else {
			if x.DefaultVal != "" {
				j.Example = x.DefaultVal
			} else if len(x.ValidCodes) > 0 {
				j.Example = x.ValidCodes[0]
			}
		}
	} else {
		j.Type = "object"
		j.Title = fmt.Sprintf("%s_%s", x.Type, x.Name)
		j.Properties = make(map[string]*jsonSchema)
		for _, child := range x.Structure {
			childProperty := child.newSchema()
			if childProperty != nil {
				j.Properties[child.Label] = childProperty
				if child.Usage == Required {
					j.Required = append(j.Required, child.Label)
				}
			}
		}
	}

	if x.IsRepeatable() {
		aj := &jsonSchema{
			Type:     "array",
			MinItems: x.RepeatMin,
			MaxItems: x.RepeatMax,
			Items:    j,
		}
		x.jsonSchema = aj
		return aj
	}
	x.jsonSchema = j
	return j
}

// Path returns the path of this X12Spec, which is a materialized path of
// this X12Spec's position in the overall spec
func (x *X12Spec) Path() string {
	return x.path
}

// Parent returns the parent of this X12Spec, or nil if this is the root
func (x *X12Spec) Parent() *X12Spec {
	return x.parent
}

func (x *X12Spec) treeString(indent string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s- %s (%#v)\n", indent, x.Name, x.Type))
	for _, child := range x.Structure {
		switch child.Type {
		case TransactionSetSpec, LoopSpec, SegmentSpec:
			b.WriteString(child.treeString(indent + "  "))
		}
	}
	return b.String()
}

func (x *X12Spec) GoString() string {
	var b strings.Builder
	b.WriteString("&X12Spec{")
	b.WriteString(fmt.Sprintf("Type: %#v, ", x.Type))
	b.WriteString(fmt.Sprintf("Name: %q, ", x.Name))
	b.WriteString(fmt.Sprintf("Usage: %#v, ", x.Usage))
	if x.Description != "" {
		b.WriteString(fmt.Sprintf("Description: %q, ", x.Description))
	}

	if x.Usage != NotUsed {
		if x.RepeatMin != 0 {
			b.WriteString(fmt.Sprintf("RepeatMin: %d, ", x.RepeatMin))
		}
		if x.RepeatMax != 0 {
			b.WriteString(fmt.Sprintf("RepeatMax: %d, ", x.RepeatMax))
		}

		b.WriteString(fmt.Sprintf("Label: %q, ", x.Label))

		switch x.Type {
		case ElementSpec:
			if len(x.ValidCodes) > 0 {
				b.WriteString(fmt.Sprintf("ValidCodes: %#v, ", x.ValidCodes))
			}
			if x.MinLength != 0 {
				b.WriteString(fmt.Sprintf("MinLength: %d, ", x.MinLength))
			}
			if x.MaxLength != 0 {
				b.WriteString(fmt.Sprintf("MaxLength: %d, ", x.MaxLength))
			}
			if x.DefaultVal != "" {
				b.WriteString(fmt.Sprintf("DefaultVal: %q, ", x.DefaultVal))
			}
			b.WriteString(fmt.Sprintf("DataType: %#v, ", x.DataType))
		case SegmentSpec:
			if len(x.Syntax) > 0 {
				b.WriteString(fmt.Sprintf("Syntax: %#v, ", x.Syntax))
			}
			if len(x.Structure) > 0 {
				b.WriteString(fmt.Sprintf("Structure: %#v, ", x.Structure))
			}
		default:
			if len(x.Structure) > 0 {
				b.WriteString(fmt.Sprintf("Structure: %#v, ", x.Structure))
			}
		}
	}
	b.WriteString("}")
	return b.String()
}

// setPaths sets the path of each X12Spec, recursively, using the given
// parent path and suffix. The path map is used to check for duplicate paths,
// and associate each path to its X12Spec. An error is returned if any
// duplicate paths are detected.
func (x *X12Spec) setPaths(
	parentPath string,
	pathMap map[string]*X12Spec,
	suffix string,
) error {
	parentPath = strings.TrimRight(parentPath, pathSeparator)
	x.path = parentPath + pathSeparator + x.Name + suffix

	dupeVal, isDupe := pathMap[x.path]
	if isDupe && dupeVal != x {
		return fmt.Errorf("duplicate path %s", x.path)
	}
	pathMap[x.path] = x
	childNameCount := make(map[string]int)
	childNameCurrentCt := make(map[string]int)
	for _, child := range x.Structure {
		if child.parent == nil {
			child.parent = x
		}
		childNameCount[child.Name]++
		childNameCurrentCt[child.Name] = 0
	}
	for _, child := range x.Structure {
		childSuffix := ""
		if childNameCount[child.Name] > 1 {
			childSuffix = fmt.Sprintf(
				"[%d]",
				childNameCurrentCt[child.Name],
			)
		}
		childNameCurrentCt[child.Name]++
		if e := child.setPaths(
			x.path,
			pathMap,
			childSuffix,
		); e != nil {
			return e
		}
	}
	return nil
}

func (x *X12Spec) Accept(v SpecVisitor) {
	v.VisitSpec(x)
}

// validateRepeat validates the RepeatMin and RepeatMax fields of the X12Spec
func (x *X12Spec) validateRepeat() error {
	var errs []error
	if x.RepeatMin < 0 {
		errs = append(
			errs,
			errors.New("repeatMin must be greater than or equal to 0"),
		)
	}
	if x.RepeatMax < 0 {
		errs = append(
			errs,
			errors.New("repeatMax must be greater than or equal to 0"),
		)
	}
	if x.RepeatMin > x.RepeatMax && x.RepeatMax > 0 {
		errs = append(
			errs,
			errors.New("repeatMin must be less than or equal to repeatMax"),
		)
	}
	return errors.Join(errs...)
}

// finalize sets the default values for certain fields if the X12Spec is
func (x *X12Spec) finalize() {
	switch x.Usage {
	case NotUsed:
		x.RepeatMin = 0
		x.RepeatMax = 0
		x.MaxLength = 0
		x.MinLength = 0
		x.ValidCodes = nil
		x.DefaultVal = ""
		x.Syntax = nil
		x.Label = ""
	default:
		if x.Type != ElementSpec {
			x.DataType = UnknownDataType
			x.ValidCodes = nil
			x.MaxLength = 0
			x.MinLength = 0
			x.ValidCodes = nil
			x.DefaultVal = ""
		}
		if x.Type != SegmentSpec {
			x.Syntax = nil
		}
	}

	if x.Usage == Required {
		switch x.Type {
		case ElementSpec:
			if x.MinLength == 0 {
				x.MinLength = 1
			}
		default:
			if x.RepeatMin == 0 {
				x.RepeatMin = 1
				if x.RepeatMax == 0 {
					x.RepeatMax = 1
				}
			}
		}
	}

	if x.parent != nil {
		if x.Type == LoopSpec {
			x.loopDepth = x.parent.loopDepth + 1
		} else {
			x.loopDepth = x.parent.loopDepth
		}
	}
	for _, s := range x.Structure {
		s.parent = x
		s.finalize()
	}
}

func (x *X12Spec) validateUsage() error {
	var errs []error

	switch x.Usage {
	case Required, Situational:
		if x.Type == LoopSpec && len(x.Structure) > 0 {
			firstSpec := x.Structure[0]
			if firstSpec.Type == SegmentSpec {
				if firstSpec.Usage != x.Usage {
					errs = append(
						errs,
						fmt.Errorf(
							"first segment in loop must match loop usage (%s)",
							x.Usage,
						),
					)
				}
			} else {
				errs = append(
					errs,
					errors.New("first child of loop must be a segment"),
				)
			}
		}
	case NotUsed:
		if x.Type != ElementSpec && x.Type != CompositeSpec {
			errs = append(
				errs,
				fmt.Errorf(
					"only elements and composites can have usage %s",
					NotUsed,
				),
			)
		}
	default:
		errs = append(
			errs,
			fmt.Errorf(
				"usage must be one of: %s, %s, %s",
				Required,
				Situational,
				NotUsed,
			),
		)
	}
	return errors.Join(errs...)
}

func (x *X12Spec) validateMinMaxLength() error {
	var errs []error
	if x.MinLength < 0 {
		errs = append(
			errs,
			errors.New("minLength must be greater than or equal to 0"),
		)
	}
	if x.MaxLength < 0 {
		errs = append(
			errs,
			errors.New("maxLength must be greater than or equal to 0"),
		)
	}
	// If this element is required, then MinLength defaults to 1 if not
	// otherwise set, in which case it's OK if it's greater than MaxLength
	// as long as MaxLength is greater than 0.
	// Situational elements can either have MinLength 0 or something
	// less than MaxLength.
	if x.MinLength > x.MaxLength && (x.Usage != Required || x.MaxLength > 0) {
		errs = append(
			errs,
			errors.New("minLength must be less than or equal to maxLength"),
		)
	}
	return errors.Join(errs...)
}

func (x *X12Spec) validateValidCodes() error {
	var errs []error
	// If DefaultVal and ValidCodes are set, then ValidCodes needs to
	// contain DefaultVal, or... it'll fail by default
	if x.DefaultVal != "" && len(x.ValidCodes) > 0 && !sliceContains(
		x.ValidCodes, x.DefaultVal,
	) {
		errs = append(
			errs,
			errors.New("defaultVal must be one of validCodes"),
		)
	}

	for i := 0; i < len(x.ValidCodes); i++ {
		c := x.ValidCodes[i]
		if x.MinLength > 0 && len(c) < x.MinLength {
			errs = append(
				errs,
				fmt.Errorf(
					"validCode '%s' is shorter than minLength %d",
					c,
					x.MinLength,
				),
			)
		}
		if x.MaxLength > 0 && len(c) > x.MaxLength {
			errs = append(
				errs,
				fmt.Errorf(
					"validCode %s is longer than maxLength %d",
					c,
					x.MaxLength,
				),
			)
		}
	}
	return errors.Join(errs...)
}

// IsRepeatable returns true if this X12Spec is either Situational
// or Required, and RepeatMin or RepeatMax are greater than 1.
func (x *X12Spec) IsRepeatable() bool {
	if x.NotUsed() {
		return false
	}
	// indicates an 'unbounded' repetition
	if x.RepeatMin == 1 && x.RepeatMax == 0 {
		return true
	}

	// Indicates bounded repetition
	if x.RepeatMin == 1 && x.RepeatMax > 1 {
		return true
	}

	if x.RepeatMin > 1 || x.RepeatMax > 1 {
		return true
	}

	return false
}

func (x *X12Spec) validateSpec() error {
	var errs []error

	switch x.Type {
	case TransactionSetSpec, LoopSpec:
		if len(x.Structure) == 0 {
			errs = append(
				errs,
				errors.New("transaction set/loop spec must have children"),
			)
		}
	case ElementSpec:
		if len(x.Structure) > 0 {
			errs = append(errs, errors.New("element spec cannot have children"))
		}
	case CompositeSpec:
		if x.IsRepeatable() {
			errs = append(
				errs,
				errors.New("repeatable composites not currently supported"),
			)
		}
	case SegmentSpec:
		//
	default:
		errs = append(errs, errors.New("type is required"))
	}

	if x.Name == "" {
		errs = append(errs, errors.New("name is required"))
	}

	errs = append(
		errs,
		x.validateValidCodes(),
		x.validateRepeat(),
		x.validateUsage(),
		x.validateMinMaxLength(),
	)

	for i := 0; i < len(x.Structure); i++ {
		errs = append(errs, x.Structure[i].validateSpec())
	}
	err := errors.Join(errs...)
	if err != nil {
		err = newSpecErr(err, x)
	}
	return err
}

func (x *X12Spec) NotUsed() bool {
	return x.Usage == NotUsed
}

func (x *X12Spec) Required() bool {
	return x.Usage == Required
}

func (x *X12Spec) Situational() bool {
	return x.Usage == Situational
}

// SegmentCondition defines a segment syntax rule
type SegmentCondition struct {
	// Indexes is a list of element indexes that this condition applies to
	Indexes []int `validate:"required" json:"indexes,omitempty"`
	// ConditionCode determines the behavior of this rule.
	// It should be one of:
	// - segmentSyntaxPairedOrMultiple
	// - segmentSyntaxRequired
	// - segmentSyntaxConditional
	// - segmentSyntaxListConditional
	// - segmentSyntaxExclusion
	ConditionCode ConditionCode `json:"conditionCode,omitempty"`
	Details       string        `json:"details,omitempty"`
}

// X12TransactionSetSpec defines the structure of an X12 transaction set
// bounded by ST/SE segments, as defined by X12 implementation guides.
// The first child X12Spec must be a SegmentSpec with the name ST.
// The last child X12Spec must be a SegmentSpec with the name SE.
type X12TransactionSetSpec struct {
	// Key is a description of the transaction set. It should be unique.
	Key string `json:"key"`
	//// TransactionSetCode should reflect the value to populate ST01
	//TransactionSetCode string `json:"transactionSetCode"`
	//// Version should reflect the value to populate ST03
	//Version string              `json:"version"`
	PathMap map[string]*X12Spec `json:"-"`
	*X12Spec
}

// Matches returns True if the given TransactionSetNode matches the
// X12TransactionSetSpec header
func (xs *X12TransactionSetSpec) Matches(txn *TransactionSetNode) (
	bool,
	error,
) {
	return segmentMatchesSpec(txn.Header, xs.headerSpec())
}

// headerSpec returns the first X12Spec in the structure named ST
func (xs *X12TransactionSetSpec) headerSpec() *X12Spec {
	for _, s := range xs.Structure {
		if s.Name == stSegmentId {
			return s
		}
	}
	return nil
}

func (xs *X12TransactionSetSpec) trailerSpec() *X12Spec {
	s := xs.Structure[len(xs.Structure)-1]
	if s.Name != seSegmentId {
		return nil
	}
	return s
}

// Validate validates the X12TransactionSetSpec recursively
func (xs *X12TransactionSetSpec) Validate() error {
	xs.X12Spec.finalize()

	if len(xs.Structure) >= 2 {
		header := xs.Structure[0]
		if header.Type != SegmentSpec {
			return fmt.Errorf("first child of transaction set spec must be a segment spec")
		}
		if header.Name != stSegmentId {
			return fmt.Errorf("first child of transaction set spec must be named ST")
		}

		trailer := xs.Structure[len(xs.Structure)-1]
		if trailer.Type != SegmentSpec {
			return fmt.Errorf("last child of transaction set spec must be a segment spec")
		}
		if trailer.Name != seSegmentId {
			return fmt.Errorf("last child of transaction set spec must be named SE")
		}
	} else {
		return fmt.Errorf("transaction set spec must have at least 2 child segment specs")
	}

	if err := xs.validateSpec(); err != nil {
		return fmt.Errorf("invalid transaction set spec: %w", err)
	}
	if xs.PathMap == nil {
		xs.PathMap = make(map[string]*X12Spec)
	}
	if err := xs.setPaths("", xs.PathMap, ""); err != nil {
		return fmt.Errorf("invalid transaction set spec: %w", err)
	}
	return nil
}

func (xs *X12TransactionSetSpec) GoString() string {
	var b strings.Builder
	b.WriteString("&X12TransactionSetSpec{")
	b.WriteString(fmt.Sprintf("Key: %q, ", xs.Key))
	b.WriteString(fmt.Sprintf("X12Spec: %#v, ", xs.X12Spec))
	b.WriteString("}")
	return b.String()
}

func (xs *X12TransactionSetSpec) UnmarshalJSON(data []byte) error {
	type tempSpec X12TransactionSetSpec
	var temp tempSpec

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}
	temp.PathMap = make(map[string]*X12Spec)
	temp.Name = transactionSetName
	temp.Usage = Situational
	temp.Type = TransactionSetSpec
	temp.Description = transactionSpecDescription

	// accumulates errors rather than returning on the first error
	specErrors := []error{}

	// copy the structure into a new slice so we can clear it out
	specStructure := []*X12Spec{}
	for i := 0; i < len(temp.Structure); i++ {
		specStructure = append(specStructure, temp.Structure[i])
	}
	temp.Structure = []*X12Spec{}

	temp.RepeatMin = 1
	temp.Label = transactionSpecLabel

	for _, ss := range specStructure {
		specErrors = append(
			specErrors,
			appendSpec(temp.X12Spec, ss),
		)
	}

	*xs = X12TransactionSetSpec(temp)
	if err := xs.Validate(); err != nil {
		specErrors = append(specErrors, err)
	}
	return errors.Join(specErrors...)
}

// JSONSchema returns the marshaled JSON schema for this X12TransactionSetSpec
func (txn *X12TransactionSetSpec) JSONSchema() ([]byte, error) {
	return json.Marshal(txn.newSchema())
}

// jsonSchema returns a jsonSchema for this X12TransactionSetSpec
func (txn *X12TransactionSetSpec) newSchema() *jsonSchema {
	j := txn.X12Spec.newSchema()
	j.Title = "TransactionSet"
	j.Description = txn.Key
	txn.jsonSchema = j
	return j
}

// SpecVisitor is an interface for implementing the visitor
// pattern to walk X12Spec
type SpecVisitor interface {
	VisitSpec(x *X12Spec) error
}

// VisitableSpec is an interface to define a spec as visitable
type VisitableSpec interface {
	Accept(v *SpecVisitor)
}

func appendSpec(parentSpec *X12Spec, childSpec *X12Spec) error {
	allowed, ok := specChildTypes[parentSpec.Type]
	if !ok {
		return newSpecErr(
			fmt.Errorf("unexpected spec type '%d'", parentSpec.Type),
			parentSpec,
		)
	}
	if !sliceContains(allowed, childSpec.Type) {
		specNames := make([]string, len(allowed))
		for i, specType := range allowed {
			specNames[i] = specType.String()
		}
		return newSpecErr(
			fmt.Errorf(
				"cannot append %s to %s: must be one of: %s",
				childSpec.Type,
				parentSpec.Type,
				strings.Join(specNames, ", "),
			), childSpec,
		)
	}

	if childSpec.Label == "" && !childSpec.NotUsed() {
		return newSpecErr(
			fmt.Errorf("spec must have a label"), childSpec,
		)
	}
	specStructure := parentSpec.Structure
	for i := 0; i < len(specStructure); i++ {
		subSpec := specStructure[i]
		if subSpec.Label == childSpec.Label && !subSpec.NotUsed() {
			return newSpecErr(
				fmt.Errorf("spec label '%s' already exists", childSpec.Label),
				childSpec,
			)
		}
	}
	parentSpec.Structure = append(parentSpec.Structure, childSpec)
	childSpec.parent = parentSpec
	return nil
}

// sliceContains returns true if the given value is present in the given slice
func sliceContains[V comparable](row []V, val V) bool {
	for _, v := range row {
		if v == val {
			return true
		}
	}
	return false
}

// allZero returns true if all values in the given map are zero values
func allZero(m map[string]any) bool {
	for _, v := range m {
		if !isZero(reflect.ValueOf(v)) {
			return false
		}
	}
	return true
}

// isZero returns true if the given value is a zero value
func isZero(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}
	switch v.Kind() {
	case reflect.Func, reflect.Map, reflect.Slice:
		return v.IsNil()
	default:
		return v.Interface() == reflect.Zero(v.Type()).Interface()
	}
}

// removeTrailingEmptyElements removes trailing empty elements from a
// slice of elements. These are truncated in segments. For example,
// a segment which specifies 5 elements, where the latter two are optional,
// wouldn't look like `SEGID*A*B*C**~`, but rather `SEGID*A*B*C~`
func removeTrailingEmptyElements(elements []string) []string {
	for i := len(elements) - 1; i >= 0; i-- {
		if elements[i] != "" {
			newSlice := make([]string, i+1)
			copy(newSlice, elements)
			return newSlice
		}
	}
	return []string{}
}

// uniqueElements returns a slice containing each unique element in
// the given slice, ex: ["a", "b", "a", "c"] -> ["a", "b", "c"]
func uniqueElements[V comparable](elements []V) []V {
	keysSeen := make(map[V]bool)
	result := make([]V, 0, len(elements))

	for _, v := range elements {
		if _, seen := keysSeen[v]; !seen {
			keysSeen[v] = true
			result = append(result, v)
		}
	}
	return result
}

// jsonSchema is a struct for marshaling a JSON schema
type jsonSchema struct {
	Type       string                 `json:"type"`
	Format     string                 `json:"format,omitempty"`
	Title      string                 `json:"title,omitempty"`
	Required   []string               `json:"required,omitempty"`
	Properties map[string]*jsonSchema `json:"properties,omitempty"`
	//AdditionalProperties map[string]jsonSchema `json:"additionalProperties,omitempty"`
	Description string `json:"description,omitempty"`

	Example              string   `json:"example,omitempty"`
	Enum                 []string `json:"enum,omitempty"`
	AdditionalProperties bool     `json:"additionalProperties"`

	Default string `json:"default,omitempty"`

	ReadOnly  *bool `json:"readOnly,omitempty"`
	WriteOnly *bool `json:"writeOnly,omitempty"`
	//Contains *jsonSchema `json:"contains,omitempty"`
	//MinContains int               `json:"minContains,omitempty"`
	//MaxContains int               `json:"maxContains,omitempty"`

	Items    *jsonSchema `json:"items,omitempty"`
	MinItems int         `json:"minItems,omitempty"`
	MaxItems int         `json:"maxItems,omitempty"`

	MinLength int `json:"minLength,omitempty"`
	MaxLength int `json:"maxLength,omitempty"`

	DependentRequired map[string][]string `json:"dependentRequired,omitempty"`
}
