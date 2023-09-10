package edx12

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"reflect"
	"strings"
	"sync"
)

//go:embed assets/transactions/*
var x12Specs embed.FS

//go:embed assets/envelope/isa.json
var isaEnvelopeSpecJSON []byte

//go:embed assets/envelope/iea.json
var ieaEnvelopeSpecJSON []byte

//go:embed assets/envelope/gs.json
var gsEnvelopeSpecJSON []byte

//go:embed assets/envelope/ge.json
var geEnvelopeSpecJSON []byte

//go:embed assets/envelope/st.json
var stEnvelopeSpecJSON []byte

//go:embed assets/envelope/se.json
var seEnvelopeSpecJSON []byte

var transactionSpecs = make(map[string]*X12TransactionSetSpec)
var transactionSpecMu = &sync.Mutex{}
var pathSeparator = "/"
var defaultTransactionSpecPath = fmt.Sprintf("%sST_LOOP", pathSeparator)

// getAllFilenames returns a list of all filenames in the given embed.FS
func getAllFilenames(efs *embed.FS) (files []string, err error) {
	if err := fs.WalkDir(
		efs,
		".",
		func(path string, d fs.DirEntry, err error) error {

			if d.IsDir() {
				return nil
			}

			files = append(files, path)

			return nil
		},
	); err != nil {
		return nil, err
	}

	return files, nil
}

func loadTransactionSpecFiles(efs *embed.FS) (
	specs []*X12TransactionSetSpec,
	err error,
) {
	specs = []*X12TransactionSetSpec{}
	var specErrors []error
	specFiles, err := getAllFilenames(efs)
	if err != nil {
		specErrors = append(specErrors, err)
	}

	for _, specFile := range specFiles {
		content, err := x12Specs.ReadFile(specFile)
		if err != nil {
			specErrors = append(specErrors, err)
			continue
		}
		sp := &X12TransactionSetSpec{}
		if err := json.Unmarshal(content, sp); err != nil {
			specErrors = append(specErrors, err)
			continue
		}
		specs = append(specs, sp)
		if err := Register(sp); err != nil {
			log.Printf("error loading %s: %s", specFile, err.Error())
		}
	}
	err = errors.Join(specErrors...)
	return specs, err
}

type DataType uint

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

func (c ConditionCode) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.String())
}

func (c *ConditionCode) UnmarshalJSON(b []byte) error {
	var conditionName string

	if err := json.Unmarshal(b, &conditionName); err != nil {
		return err
	}
	switch conditionName {
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
	// name is the name of the field as it should appear in the x12 spec.
	// Ex: `2000A`, `NM1`, ...
	Name string `validate:"required,gte=1" json:"name"`
	// usage indicates whether the field is required, situational, or not used.
	// Its value should be one of:
	// - `REQUIRED`
	// - `SITUATIONAL`
	// - `NO _USED`
	Usage       Usage  `json:"usage"`
	Description string `json:"description,omitempty"`
	// repeatMin is the minimum number of times this can repeat. If this
	// is nil, then there is no minimum (except when repeatMax is 1 and this
	// is a required X12Spec, in which case repeatMax, if not nil, is
	// implicitly the minimum)
	RepeatMin int `json:"repeatMin,omitempty"`
	// repeatMax is the maximum number of times this can repeat.
	// If this is nil, then there is no maximum.
	RepeatMax int `json:"repeatMax,omitempty"`
	// label is a human-readable label for this X12Spec. It will be used when
	// creating and ingesting JSON payloads. If not set, it will default to
	// lower-cased name
	Label string `json:"label,omitempty"`
	// path is a materialized path of this X12Spec's position in
	// the overall spec
	Path      string     `json:"-"`
	Type      SpecType   `json:"type"`
	Parent    *X12Spec   `json:"-"`
	Structure []*X12Spec `json:"structure,omitempty"`

	ValidCodes []string `json:"validCodes,omitempty"`
	MinLength  int      `json:"minLength,omitempty"`
	// maxLength specifies the maximum length of the element when
	// a value is provided
	MaxLength int `json:"maxLength,omitempty"`
	// defaultVal specifies the default value for this element to use
	// when generating messages - if an element is required and a value
	// is not provided, this will be the value used.
	DefaultVal string `json:"default,omitempty"`
	// dataType indicates the ASC x12 data type of this element
	DataType string `validate:"omitempty,oneof=AN ID N0 N Nn R DT TM" json:"dataType,omitempty"`

	Syntax []SegmentCondition `json:"syntax,omitempty"`
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

type X12TransactionSetSpec struct {
	Key                string              `json:"key"`
	TransactionSetCode string              `json:"transactionSetCode"`
	Version            string              `json:"version"`
	PathMap            map[string]*X12Spec `json:"-"`
	*X12Spec
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

	// 'structure' doesn't include the ST/SE segments, just what's between
	// them, so we add those in here and set the transaction set code/version
	// in ValidCodes and DefaultVal
	headerSpec, err := transactionHeaderSpec()
	if err != nil {
		specErrors = append(specErrors, err)
	}

	// ST01 = transaction set identifier code (270, 271, etc)
	st01, st01Err := headerSpec.getSpecByLabel(transactionSetCodeSpecLabel)
	if st01Err != nil {
		specErrors = append(specErrors, st01Err)
	}
	st01.DefaultVal = temp.TransactionSetCode
	st01.ValidCodes = append(st01.ValidCodes, temp.TransactionSetCode)

	// ST03 = implementation convention reference (ex: 005010X279A1)
	st03, st03Err := headerSpec.getSpecByLabel(transactionVersionSpecLabel)
	if st03Err != nil {
		specErrors = append(specErrors, st03Err)
	}
	st03.DefaultVal = temp.Version
	st03.ValidCodes = append(st03.ValidCodes, temp.Version)

	// SE trailer
	trailerSpec, err := transactionTrailerSpec()
	if err != nil {
		specErrors = append(specErrors, err)
	}

	// copy the structure into a new slice so we can clear it out, insert
	// the ST header, re-add the existing structure, then append the
	// SE trailer
	specStructure := []*X12Spec{}
	for i := 0; i < len(temp.Structure); i++ {
		specStructure = append(specStructure, temp.Structure[i])
	}

	temp.Structure = []*X12Spec{}

	if err = appendSpec(temp.X12Spec, headerSpec); err != nil {
		specErrors = append(specErrors, err)
	}

	temp.RepeatMin = 1
	temp.Label = transactionSpecLabel

	for _, ss := range specStructure {
		switch ss.Type {
		case LoopSpec, SegmentSpec:
			specErrors = append(
				specErrors,
				appendSpec(temp.X12Spec, ss),
			)
		default:
			specErrors = append(
				specErrors,
				fmt.Errorf("unexpected spec type: %s", ss.Type),
			)
		}
	}

	specErrors = append(
		specErrors,
		appendSpec(temp.X12Spec, trailerSpec),
	)

	if err = temp.validateSpec(); err != nil {
		specErrors = append(specErrors, err)
	}
	if e := temp.SetPaths("", temp.PathMap, ""); e != nil {
		specErrors = append(specErrors, e)
	}

	*xs = X12TransactionSetSpec(temp)
	return errors.Join(specErrors...)
}

func (x *X12Spec) SetPaths(
	parentPath string,
	pathMap map[string]*X12Spec,
	suffix string,
) error {
	parentPath = strings.TrimRight(parentPath, pathSeparator)

	x.Path = parentPath + pathSeparator + x.Name + suffix

	dupeVal, isDupe := pathMap[x.Path]
	if isDupe && dupeVal != x {
		return fmt.Errorf("duplicate path %s", x.Path)
	}
	pathMap[x.Path] = x
	childNameCount := make(map[string]int)
	childNameCurrentCt := make(map[string]int)

	for _, child := range x.Structure {
		if child.Parent == nil {
			child.Parent = x
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
		if e := child.SetPaths(
			string(x.Path),
			pathMap,
			childSuffix,
		); e != nil {
			return e
		}
	}
	return nil
}

type SpecVisitor interface {
	VisitSpec(x *X12Spec) error
}

type VisitableSpec interface {
	Accept(v *SpecVisitor)
}

func (x *X12Spec) Accept(v SpecVisitor) {
	v.VisitSpec(x)
}

type SpecWalkerFunc func(x *X12Spec) error

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

func (x *X12Spec) validateUsage() error {
	var errs []error
	if x.Usage != Required && x.Usage != Situational && x.Usage != NotUsed {
		errs = append(
			errs,
			errors.New("usage must be one of REQUIRED, SITUATIONAL, NOT USED"),
		)
	}
	if x.Usage == NotUsed {
		if x.RepeatMin > 0 {
			errs = append(
				errs,
				errors.New("repeatMin must be 0 if usage is NOT USED"),
			)
		}
		if x.RepeatMax > 0 {
			errs = append(
				errs,
				errors.New("repeatMax must be 0 if usage is NOT USED"),
			)
		}
	}
	return errors.Join(errs...)
}

func (x *X12Spec) validateMinMaxLength() error {
	var errs []error
	if x.Type != ElementSpec && (x.MinLength > 0 || x.MaxLength > 0) {
		errs = append(
			errs,
			errors.New("minLength and maxLength are only valid for elements"),
		)
		return errors.Join(errs...)
	}
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
	if x.MinLength > x.MaxLength {
		errs = append(
			errs,
			errors.New("minLength must be less than or equal to maxLength"),
		)
	}
	return errors.Join(errs...)
}

func (x *X12Spec) validateValidCodes() error {
	var errs []error
	if len(x.ValidCodes) > 0 && x.Type != ElementSpec {
		errs = append(
			errs,
			fmt.Errorf(
				"validCodes is only valid for elements, not: %s",
				x.Type,
			),
		)
		return errors.Join(errs...)
	}
	if x.DefaultVal != "" {
		if !sliceContains(x.ValidCodes, x.DefaultVal) {
			errs = append(
				errs,
				errors.New("defaultVal must be one of validCodes"),
			)
		}
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

func (x *X12Spec) IsRepeatable() bool {
	if x.NotUsed() {
		return false
	}
	if x.RepeatMin > 1 || x.RepeatMax > 1 {
		return true
	}
	return false
}

func (x *X12Spec) validateSpec() error {
	var errs []error

	if x.Type == UnknownSpec {
		errs = append(errs, errors.New("type is required"))
	}
	if x.Name == "" {
		errs = append(errs, errors.New("name is required"))
	}
	if x.Usage == UnknownUsage {
		errs = append(errs, errors.New("usage is required"))
	}

	if len(x.Syntax) > 0 && x.Type != SegmentSpec {
		errs = append(errs, errors.New("syntax is only valid for segments"))
	}

	switch x.Type {
	case TransactionSetSpec:
		if x.Parent != nil {
			errs = append(
				errs,
				errors.New("transaction set spec cannot have a parent"),
			)
		}
	case ElementSpec:
		if len(x.Structure) > 0 {
			errs = append(errs, errors.New("element spec cannot have children"))
		}
	}

	errs = append(errs, x.validateValidCodes())
	errs = append(errs, x.validateRepeat())
	errs = append(errs, x.validateUsage())
	errs = append(errs, x.validateMinMaxLength())

	for i := 0; i < len(errs); i++ {
		if errs[i] != nil {
			errs[i] = newSpecErr(errs[i], x)
		}
	}
	for i := 0; i < len(x.Structure); i++ {
		errs = append(errs, x.Structure[i].validateSpec())
	}

	return errors.Join(errs...)
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

func (x *X12Spec) appendStructure(spec *X12Spec) {
	x.Structure = append(x.Structure, spec)
}

func (x *X12Spec) SetParent(spec *X12Spec) {
	x.Parent = spec
}

func childSpecAllowed(
	specType SpecType,
	childSpecType SpecType,
) (isAllowed bool) {

	allowedTypes, e := allowedChildSpecs(specType)
	if e != nil {
		return false
	}
	return sliceContains(allowedTypes, childSpecType)

}

func allowedChildSpecs(specType SpecType) ([]SpecType, error) {
	switch specType {
	case TransactionSetSpec:
		return []SpecType{SegmentSpec, LoopSpec}, nil
	case LoopSpec:
		return []SpecType{SegmentSpec, LoopSpec}, nil
	case SegmentSpec:
		return []SpecType{ElementSpec, CompositeSpec}, nil
	case CompositeSpec:
		return []SpecType{ElementSpec}, nil
	case ElementSpec:
		return []SpecType{}, nil
	default:
		return []SpecType{}, fmt.Errorf("unknown spec type: %s", specType)
	}
}

func (x *X12Spec) allowedNodeTypes() (allowedNodeTypes []NodeType) {
	switch x.Type {
	case CompositeSpec:
		return []NodeType{CompositeNode, RepeatElementNode, ElementNode}
	case ElementSpec:
		return []NodeType{ElementNode, RepeatElementNode}
	case SegmentSpec:
		return []NodeType{SegmentNode}
	case LoopSpec:
		return []NodeType{LoopNode, TransactionNode}
	case TransactionSetSpec:
		return []NodeType{TransactionNode}
	}
	return allowedNodeTypes
}

func appendSpec(parentSpec *X12Spec, childSpec *X12Spec) error {
	if !childSpecAllowed(parentSpec.Type, childSpec.Type) {
		allowedSpecs, e := allowedChildSpecs(parentSpec.Type)
		if e != nil {
			return newSpecErr(e, parentSpec)
		}
		specNames := make([]string, len(allowedSpecs))
		for i, specType := range allowedSpecs {
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
	parentSpec.appendStructure(childSpec)
	childSpec.SetParent(parentSpec)
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

func allZero(m map[string]any) bool {
	for _, v := range m {
		if !isZero(reflect.ValueOf(v)) {
			return false
		}
	}
	return true
}

func isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Func, reflect.Map, reflect.Slice:
		return v.IsNil()
	case reflect.Array, reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if !isZero(v.Field(i)) {
				return false
			}
		}
		return true
	default:
		return v.Interface() == reflect.Zero(v.Type()).Interface()
	}
}

//func mapKeys[K comparable](m map[K]interface{}) []K {
//	keys := make([]K, 0, len(m))
//	for k := range m {
//		keys = append(keys, k)
//	}
//	return keys
//}

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

func interchangeHeaderSpec() (spec *X12Spec, err error) {
	spec = &X12Spec{}
	err = json.Unmarshal(isaEnvelopeSpecJSON, spec)
	return spec, err
}

func interchangeTrailerSpec() (spec *X12Spec, err error) {
	spec = &X12Spec{}
	err = json.Unmarshal(ieaEnvelopeSpecJSON, spec)
	return spec, err
}

func groupHeaderSpec() (spec *X12Spec, err error) {
	spec = &X12Spec{}
	err = json.Unmarshal(gsEnvelopeSpecJSON, spec)
	return spec, err
}

func groupTrailerSpec() (spec *X12Spec, err error) {
	spec = &X12Spec{}
	err = json.Unmarshal(geEnvelopeSpecJSON, spec)
	return spec, err
}

func transactionHeaderSpec() (spec *X12Spec, err error) {
	spec = &X12Spec{}
	err = json.Unmarshal(stEnvelopeSpecJSON, spec)
	return spec, err
}

func transactionTrailerSpec() (spec *X12Spec, err error) {
	spec = &X12Spec{}
	err = json.Unmarshal(seEnvelopeSpecJSON, spec)
	return spec, err
}

func init() {
	_, err := loadTransactionSpecFiles(&x12Specs)
	if err != nil {
		log.Fatalf("error loading transaction specs: %s", err)
	}
	//for _, s := range specs {
	//	log.Printf("loaded: %s/%s (%s)", s.TransactionSetCode, s.Version, s.Key)
	//}

}

// Register registers a TransactionSetSpec with the library.
// If TransactionSetSpec.Key is already registered, an error will be
// returned. If it already exists on init, an error will be returned.
func Register(spec *X12TransactionSetSpec) error {
	if _, hasKey := transactionSpecs[spec.Key]; hasKey {
		return fmt.Errorf("duplicate transaction set spec: %s", spec.Key)
	}
	//log.Printf("registered: %s", spec.Key)
	transactionSpecs[spec.Key] = spec
	return nil
}

// clearRegisteredSpecs removes all loaded TransactionSetSpec from the library.
func clearRegisteredSpecs() {
	for k := range transactionSpecs {
		delete(transactionSpecs, k)
		//log.Printf("deregsitered: %s", k)
	}
}

// TransactionSpecs returns a map of currently registered TransactionSetSpec
// instances, keyed by TransactionSetSpec.Key
func TransactionSpecs() map[string]*X12TransactionSetSpec {
	specs := make(map[string]*X12TransactionSetSpec)
	for k, v := range transactionSpecs {
		specs[k] = v
	}
	return specs
}

func (x *X12Spec) getSpecByLabel(label string) (*X12Spec, error) {
	for _, childSpec := range x.Structure {
		if childSpec.Label == label {
			return childSpec, nil
		}
	}
	return nil, fmt.Errorf("no element spec with label %s found", label)
}

// assignTransactionSpecX12Paths assigns a path to each X12Spec starting
// from the given node
//func assignTransactionSpecX12Paths(
//	node *X12Spec,
//	parentPath string,
//	pathMap map[string]any,
//	suffix string,
//) (err error) {
//	parentPath = strings.TrimRight(parentPath, x12PathSeparator)
//	switch node.Type {
//	case SegmentSpec:
//		node.Path = parentPath + x12PathSeparator + node.Name + suffix
//		dupeVal, isDupe := pathMap[node.Path]
//		if isDupe && dupeVal != node {
//			return newSpecErr(
//				fmt.Errorf(
//					"duplicate path: %s\nexisting: %#v\ncurrent: %#v",
//					node.Path,
//					dupeVal,
//					node,
//				), node,
//			)
//
//		}
//		pathMap[node.Path] = node
//
//		childNameCount := make(map[string]int)
//		childNameCurrentCt := make(map[string]int)
//		for _, child := range node.Structure {
//			childNameCount[child.Name]++
//			childNameCurrentCt[child.Name] = 0
//		}
//		for _, child := range node.Structure {
//			childSuffix := ""
//			if childNameCount[child.Name] > 1 {
//				childSuffix = fmt.Sprintf(
//					"[%d]",
//					childNameCurrentCt[child.Name],
//				)
//			}
//			childNameCurrentCt[child.Name]++
//			if e := assignTransactionSpecX12Paths(
//				child,
//				string(node.Path),
//				pathMap,
//				childSuffix,
//			); e != nil {
//				return e
//			}
//		}
//	case LoopSpec:
//		node.Path = parentPath + x12PathSeparator + node.Name + suffix
//		dupeVal, isDupe := pathMap[node.Path]
//		if isDupe && dupeVal != node {
//			return newSpecErr(
//				fmt.Errorf(
//					"duplicate path: %s\nexisting: %#v\ncurrent: %#v",
//					node.Path,
//					dupeVal,
//					node,
//				), node,
//			)
//		}
//		pathMap[node.Path] = node
//		childNameCount := make(map[string]int)
//		childNameCurrentCt := make(map[string]int)
//		for _, child := range node.Structure {
//			childNameCount[child.Name]++
//			childNameCurrentCt[child.Name] = 0
//		}
//		for _, child := range node.Structure {
//			childSuffix := ""
//			if childNameCount[child.Name] > 1 {
//				childSuffix = fmt.Sprintf(
//					"[%d]",
//					childNameCurrentCt[child.Name],
//				)
//			}
//			childNameCurrentCt[child.Name]++
//			if e := assignTransactionSpecX12Paths(
//				child,
//				string(node.Path),
//				pathMap,
//				childSuffix,
//			); e != nil {
//				return e
//			}
//		}
//	case ElementSpec:
//		node.Path = parentPath + x12PathSeparator + node.Name + suffix
//		dupeVal, isDupe := pathMap[node.Path]
//		if isDupe && dupeVal != node {
//			return newSpecErr(
//				fmt.Errorf(
//					"duplicate path: %s\nexisting: %#v\ncurrent: %#v",
//					node.Path,
//					dupeVal,
//					node,
//				), node,
//			)
//		}
//		pathMap[node.Path] = node
//	case CompositeSpec:
//		node.Path = parentPath + x12PathSeparator + node.Name + suffix
//		dupeVal, isDupe := pathMap[node.Path]
//		if isDupe && dupeVal != node {
//			return newSpecErr(
//				fmt.Errorf(
//					"duplicate path: %s\nexisting: %#v\ncurrent: %#v",
//					node.Path,
//					dupeVal,
//					node,
//				), node,
//			)
//		}
//		pathMap[node.Path] = node
//		childNameCount := make(map[string]int)
//		childNameCurrentCt := make(map[string]int)
//		for _, child := range node.Structure {
//			childNameCount[child.Name]++
//			childNameCurrentCt[child.Name] = 0
//		}
//		for _, child := range node.Structure {
//			childSuffix := ""
//			if childNameCount[child.Name] > 1 {
//				childSuffix = fmt.Sprintf(
//					"[%d]",
//					childNameCurrentCt[child.Name],
//				)
//			}
//			childNameCurrentCt[child.Name]++
//			if e := assignTransactionSpecX12Paths(
//				child,
//				string(node.Path),
//				pathMap,
//				childSuffix,
//			); e != nil {
//				return e
//			}
//		}
//	case TransactionSetSpec:
//		node.Path = parentPath + x12PathSeparator + transactionSetName
//		dupeVal, isDupe := pathMap[node.Path]
//		if isDupe && dupeVal != node {
//			return newSpecErr(
//				fmt.Errorf(
//					"duplicate path: %s\nexisting: %#v\ncurrent: %#v",
//					node.Path,
//					dupeVal,
//					node,
//				), node,
//			)
//		}
//		pathMap[node.Path] = node
//		childNameCount := make(map[string]int)
//		childNameCurrentCt := make(map[string]int)
//		for _, child := range node.Structure {
//			childNameCount[child.Name]++
//			childNameCurrentCt[child.Name] = 0
//		}
//		for _, child := range node.Structure {
//			childSuffix := ""
//			if childNameCount[child.Name] > 1 {
//				childSuffix = fmt.Sprintf(
//					"[%d]",
//					childNameCurrentCt[child.Name],
//				)
//			}
//			childNameCurrentCt[child.Name]++
//			if e := assignTransactionSpecX12Paths(
//				child,
//				string(node.Path),
//				pathMap,
//				childSuffix,
//			); e != nil {
//				return e
//			}
//		}
//	default:
//		return fmt.Errorf("unknown type %T", node)
//	}
//	return err
//}
