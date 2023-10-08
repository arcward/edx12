package edx12

import (
	"container/list"
	"context"
	"math"
)

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
	// by createLoop()
	Children []*X12Node
	// ErrorLog contains any errors encountered during mapping
	ErrorLog []error
	// message is the top-level Message instance for the current loop
	message *Message
	// parentLoopBuilder is the loopTransformer that created this instance
	parentLoopBuilder *loopTransformer
	// createdLoop is the X12Node instance (nodeType Loop) created by
	// this loopTransformer
	createdLoop     *X12Node
	ctx             context.Context
	matchValidCodes bool
}

func (v *loopTransformer) createLoop() (*X12Node, error) {
	if len(v.Children) == 0 {
		return nil, nil
	}
	if v.createdLoop == nil {
		v.createdLoop = &X12Node{}
	}
	if err := v.createdLoop.SetSpec(v.LoopSpec); err != nil {
		return nil, err
	}
	v.createdLoop.Children = v.Children
	return v.createdLoop, nil
}

// VisitLoopSpec is called when a LoopSpec is encountered in the x12 definition.
// It creates a new loopTransformer for the LoopSpec, and tries to identify
// matching segments in the SegmentQueue. If a match is found, they're
// added to loopTransformer.Children. If another LoopSpec is encountered,
// a new loopTransformer is created for that LoopSpec and the same process
// is repeated.
func (v *loopTransformer) VisitLoopSpec(loopSpec *X12Spec) {
	segmentQueue := v.SegmentQueue
	if v.ctx.Err() != nil || segmentQueue.Length() == 0 {
		return
	}

	rmax := loopSpec.RepeatMax
	occurrence := 0
	loopCount := 0
	for segmentQueue.Length() > 0 {
		if v.ctx.Err() != nil || (rmax != 0 && loopCount == rmax) {
			break
		} else if len(v.Children) == 0 {
			return
		}
		leadSegment := segmentQueue.PopLeft()
		leadSegmentName := leadSegment.Name
		segmentQueue.AppendLeft(leadSegment)
		if leadSegmentName != loopSpec.Structure[0].Name {
			break
		}

		loopParser := &loopTransformer{
			LoopSpec:          loopSpec,
			SegmentQueue:      segmentQueue,
			message:           v.message,
			parentLoopBuilder: v,
			ctx:               v.ctx,
			matchValidCodes:   v.matchValidCodes,
			Children:          []*X12Node{},
		}
		for _, childSpec := range loopSpec.Structure {
			if v.ctx.Err() != nil {
				break
			}
			if segmentQueue.Length() == 0 {
				break
			}
			childSpec.Accept(loopParser)
		}

		v.ErrorLog = append(v.ErrorLog, loopParser.ErrorLog...)

		if len(loopParser.Children) == 0 {
			break
		} else {
			updated, err := loopParser.createLoop()
			if err != nil {
				v.ErrorLog = append(v.ErrorLog, err)
			}
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
	matches := v.matches(segmentSpec)
	for _, match := range matches {
		v.Children = append(v.Children, match)
	}
}

// matches pops segments off of loopTransformer.SegmentQueue as long as
// they match the given SegmentSpec. If a segment is matched to the
// maximum number of repeats, but still matches the current spec, it will
// continue to match against the current spec (past the max repeat) as long
// as the next spec in the parent loop isn't a SegmentSpec which also matches
// (or there is no subsequent spec).
func (v *loopTransformer) matches(
	spec *X12Spec,
) (matches []*X12Node) {
	segmentQueue := v.SegmentQueue
	rmax := spec.RepeatMax

	// Unbounded repeat = -1
	repeatable := spec.IsRepeatable()
	if repeatable {
		if rmax < 1 {
			rmax = -1
		}
	} else {
		rmax = 1
	}

	for segmentQueue.Length() > 0 && v.ctx.Err() == nil {
		if !repeatable && len(matches) == 1 {
			return matches
		}

		segment := segmentQueue.PopLeft()
		if segment == nil {
			return matches
		}

		// An HL segment begins a new loop, so if we already have
		// children identified for this loop, we need to stop
		if segment.Name == hlSegmentId && len(v.Children) > 0 {
			segmentQueue.AppendLeft(segment)
			return matches
		}

		// Handles repeatable segments which have a maximum repeat
		// and we've matched up to that maximum. This allows us to
		// accept excess repeats of the same segment, as long as
		// the spec still matches, and either the parent spec has
		// no remaining specs, or the next spec in the parent
		// doesn't match the segment
		if repeatable && rmax > 1 && len(matches) == rmax {
			nextSpec, err := spec.nextSpec()
			if err != nil {
				segmentQueue.AppendLeft(segment)
				return matches
			}

			currentSpecMatches, _ := segmentMatchesSpec(segment, spec)

			if !currentSpecMatches {
				if nextSpec == nil {
					segmentQueue.AppendLeft(segment)
					return matches
				}
				if nextSpec.Type != SegmentSpec {
					segmentQueue.AppendLeft(segment)
					return matches
				}
			}

			if nextSpec != nil {
				nextMatch, e := segmentMatchesSpec(segment, nextSpec)
				if !currentSpecMatches && e != nil {
					segmentQueue.AppendLeft(segment)
					return matches
				}
				if nextMatch {
					segmentQueue.AppendLeft(segment)
					return matches
				}
			}

			if !currentSpecMatches {
				segmentQueue.AppendLeft(segment)
				return matches
			}
		}

		var segMatches bool
		if v.matchValidCodes {
			var e error
			segMatches, e = segmentMatchesSpec(segment, spec)
			if e != nil {
				v.ErrorLog = append(v.ErrorLog, newSpecErr(e, spec))
			}
		} else if segment.Name == segment.Name {
			segMatches = true
		}

		if !segMatches {
			segmentQueue.AppendLeft(segment)
			return matches
		}
		matches = append(matches, segment)

		if err := segment.SetSpec(spec); err != nil {
			v.ErrorLog = append(v.ErrorLog, err)
		}
		segment.Occurrence = len(matches)
	}
	return matches
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

func (d *segmentDeque) Pop() *X12Node {
	if d.segments.Len() > 0 {
		last := d.segments.Back()
		d.segments.Remove(last)
		return last.Value.(*X12Node)
	}
	return nil
}

// Length returns the number of Segments in the deque.
func (d *segmentDeque) Length() int {
	return d.segments.Len()
}
