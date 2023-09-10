package edx12

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

func UnmarshalText(text []byte, msg *Message) (err error) {
	if !utf8.Valid(text) {
		return fmt.Errorf("message text is not valid UTF-8")
	}
	msg.Children = []*X12Node{}
	// You can't safely trim trailing whitespace at this point, because
	// the segment terminator may be a newline, which is whitespace. But
	// since the first three characters of the message are always `ISA`,
	// you can safely trim leading whitespace.
	text = bytes.TrimLeftFunc(text, unicode.IsSpace)
	runes := bytes.Runes(text)
	if len(runes) < isaByteCount {
		return fmt.Errorf(
			"message too short to accommodate ISA segment (expected at least %d bytes, got %d)",
			isaByteCount,
			len(runes),
		)
	}
	isaStr := runes[:isaByteCount]

	msg.ElementSeparator = isaStr[isaElementSeparatorIndex]
	if msg.ElementSeparator == ' ' {
		return fmt.Errorf(
			"%w: element separator cannot be whitespace",
			ErrInvalidISA,
		)
	}

	msg.SegmentTerminator = isaStr[isaByteCount-1]

	if msg.SegmentTerminator == msg.ElementSeparator {
		return fmt.Errorf(
			"%w: segment terminator same as element delimiter",
			ErrInvalidISA,
		)
	}
	var validationErrors []error

	isaSpec, specErr := interchangeHeaderSpec()
	if specErr != nil {
		return fmt.Errorf("error loading ISA spec: %w", specErr)
	}

	isaNode := &X12Node{Type: SegmentNode, Name: isaSegmentId}
	//log.Printf("spec: %#v", isaSpec)
	if err = isaNode.SetSpec(isaSpec); err != nil {
		return err
	}
	msg.Header = isaNode

	msgText := string(runes)
	msgText = textCleanup(msgText, msg.SegmentTerminator)

	splitLines := strings.Split(
		msgText,
		string(msg.SegmentTerminator),
	)
	segmentLines := make([]string, 0, len(splitLines))
	for _, line := range splitLines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		segmentLines = append(segmentLines, line)
	}

	isaSegmentElements := strings.Split(
		segmentLines[0],
		string(msg.ElementSeparator),
	)
	//log.Printf("elems: %#v", isaSegmentElements)
	names := []string{}
	for i := 0; i < len(isaNode.Children); i++ {
		names = append(names, isaNode.Children[i].Name)
	}
	//log.Printf("children: %#v", names)
	for elemInd, elemVal := range isaSegmentElements {
		if len(isaNode.Children) <= elemInd {
			return newError(
				isaNode, fmt.Errorf(
					"%w: expected ISA segment, got '%s' (%d vs %d)",
					ErrInvalidISA,
					isaSegmentElements[0],
					len(isaNode.Children),
					elemInd,
				),
			)
		}
		isaElem := isaNode.Children[elemInd]
		if e := isaElem.SetValue(elemVal); e != nil {
			validationErrors = append(
				validationErrors,
				fmt.Errorf("error setting ISA element %d: %w", elemInd, e),
			)
		}
	}

	if len(isaSegmentElements) != isaElementCount {
		return newError(
			isaNode, fmt.Errorf(
				"%w: expected %d elements, got %d",
				ErrInvalidISA,
				isaElementCount,
				len(isaSegmentElements),
			),
		)
	}
	if isaSegmentElements[isaIndexSegmentId] != isaSegmentId {
		validationErrors = append(
			validationErrors, newError(
				isaNode, fmt.Errorf(
					"%w: expected ISA segment, got '%s'",
					ErrInvalidISA, isaSegmentElements[0],
				),
			),
		)
		// return early here, because without an ISA segment, we can't
		// group the rest of the message by ISA/GS/ST/SE/GE/IEA
		return errors.Join(validationErrors...)
	}

	msg.Children = append(msg.Children, isaNode)
	isaNode.Parent = msg.X12Node

	repetitionSeparator := msg.RepetitionSeparator()
	componentSeparator := msg.ComponentElementSeparator()

	// Disallow empty or duplicate separators
	if len(strings.TrimSpace(string(componentSeparator))) == 0 {
		err = newError(
			isaNode,
			fmt.Errorf("%w: empty component separator", ErrInvalidISA),
		)
	}
	if len(strings.TrimSpace(string(repetitionSeparator))) == 0 {
		err = newError(
			isaNode,
			fmt.Errorf("%w: empty repetition separator", ErrInvalidISA),
		)
	}
	if err != nil {
		return err
	}

	uq := uniqueElements(
		[]string{
			string(repetitionSeparator),
			string(componentSeparator),
			string(msg.SegmentTerminator),
		},
	)
	if len(uq) != 3 {
		err = newError(
			isaNode, fmt.Errorf(
				"%w: separators must be unique (segment terminator: '%s' repetition separator: '%s' component separator: '%s')",
				ErrInvalidISA,
				string(msg.SegmentTerminator),
				string(repetitionSeparator),
				string(componentSeparator),
			),
		)
		return err
	}

	// Yes, the segment terminator can sometimes be a newline - so only
	// clean out newlines when that's not the case
	var segs []*X12Node

	ieaSpec, _ := interchangeTrailerSpec()
	ieaNode := &X12Node{}
	validationErrors = append(validationErrors, ieaNode.SetSpec(ieaSpec))
	msg.Trailer = ieaNode
	errFmt := "segment %d, element %d: %w"
	lineCt := len(segmentLines)
	if lineCt < 2 {
		return newError(
			msg.X12Node,
			fmt.Errorf("expected at least 2 segments, got %d", lineCt),
		)
	}
	for i := 0; i < lineCt; i++ {
		elems := strings.Split(segmentLines[i], string(msg.ElementSeparator))
		elems = removeTrailingEmptyElements(elems)
		segmentID := elems[0]
		segNode, e := NewNode(SegmentNode, segmentID)
		if e != nil {
			validationErrors = append(
				validationErrors,
				fmt.Errorf(errFmt, i, 0, e),
			)
		}
		segs = append(segs, segNode)
		for elemInd := 0; elemInd < len(elems); elemInd++ {
			v := elems[elemInd]
			containsRepetitionSeparator := strings.Contains(
				v,
				string(repetitionSeparator),
			)
			containsComponentSeparator := strings.Contains(
				v,
				string(componentSeparator),
			)

			if elemInd == 0 {
				if containsRepetitionSeparator {
					validationErrors = append(
						validationErrors,
						fmt.Errorf(
							errFmt,
							i,
							elemInd,
							fmt.Errorf(
								"%w: segment ID cannot contain repetition separator",
								ErrElementHasInvalidValue,
							),
						),
					)
				}
				if containsComponentSeparator {
					validationErrors = append(
						validationErrors,
						fmt.Errorf(
							errFmt,
							i,
							elemInd,
							fmt.Errorf(
								"%w: segment ID cannot contain component separator",
								ErrElementHasInvalidValue,
							),
						),
					)
				}
				continue
			}
			// Segment ID is the first element, any element after that
			// gets an index, so the first element value being `REF` means
			// the next element gets named `REF01`, ...
			elemName := fmt.Sprintf("%s%02d", segmentID, elemInd)
			switch {
			case containsRepetitionSeparator:
				repElems := strings.Split(v, string(msg.RepetitionSeparator()))
				repNode, nodeErr := NewNode(
					RepeatElementNode,
					elemName,
					repElems...,
				)
				if nodeErr != nil {
					validationErrors = append(validationErrors, e)
					err = errors.Join(validationErrors...)
					return err
				}
				segNode.Children = append(segNode.Children, repNode)
				repNode.Parent = segNode
			case containsComponentSeparator:
				cmpElems := strings.Split(
					v,
					string(msg.ComponentElementSeparator()),
				)
				cmpNode, cmpErr := NewNode(CompositeNode, elemName)
				if cmpErr != nil {
					validationErrors = append(
						validationErrors,
						fmt.Errorf(errFmt, i, elemInd, e),
					)
					err = errors.Join(validationErrors...)
					return err
				}
				if e = populateCompositeNode(cmpNode, cmpElems...); e != nil {
					validationErrors = append(
						validationErrors,
						fmt.Errorf(errFmt, i, elemInd, e),
					)
					err = errors.Join(validationErrors...)
					return err
				}
				segNode.Children = append(segNode.Children, cmpNode)
				cmpNode.Parent = segNode
			case i == len(segmentLines)-1:
				// IEA node already has a spec set, so we look out for it
				// when assigning elements
				_, e = ieaNode.setIndexValue(elemInd, v)
				if e != nil {
					validationErrors = append(
						validationErrors,
						fmt.Errorf(errFmt, i, elemInd, e),
					)
					return errors.Join(validationErrors...)
				}
			default:
				elemNode, err := NewNode(ElementNode, elemName, v)
				if err != nil {
					validationErrors = append(
						validationErrors,
						fmt.Errorf(errFmt, i, elemInd, e),
					)
				}
				segNode.Children = append(segNode.Children, elemNode)
				elemNode.Parent = segNode
			}
		}
	}

	// Group segments by ISA/IEA, then group the segments by GS/GE
	groupedEnvelope, err := groupNodesByName(isaSegmentId, ieaSegmentId, segs)
	if err != nil {
		validationErrors = append(validationErrors, err)
		err = errors.Join(validationErrors...)
		return err
	}
	if len(groupedEnvelope) == 0 {
		validationErrors = append(
			validationErrors,
			fmt.Errorf("expected at least one functional group, got zero"),
		)
		err = errors.Join(validationErrors...)
		return err
	}

	newMsgChildren := make(
		[]*X12Node,
		len(groupedEnvelope)+2,
		len(groupedEnvelope)+2,
	)
	newMsgChildren[0] = isaNode
	groupedSegments, err := groupNodesByName(
		gsSegmentId, geSegmentId, groupedEnvelope[0],
	)
	if err != nil {
		validationErrors = append(validationErrors, err)
		err = errors.Join(validationErrors...)
		return err
	}

	msg.functionalGroups = make([]*FunctionalGroupNode, 0, len(groupedSegments))
	// Create functional groups from gsSegmentId/geSegmentId-bounded segments
	for fgInd, functionalSegments := range groupedSegments {
		n := &X12Node{}
		groupChildren := make([]*X12Node, 0, len(functionalSegments))
		n.Children = groupChildren
		for find := 0; find < len(functionalSegments); find++ {
			n.Children = append(n.Children, functionalSegments[find])
		}
		fGroup, e := createFunctionalGroupNode(n)
		fGroup.Message = msg
		if e != nil {
			validationErrors = append(validationErrors, e)
			err = errors.Join(validationErrors...)
			return err
		}
		newMsgChildren[fgInd+1] = fGroup.X12Node
		fGroup.Parent = msg.X12Node
		fGroup.Occurrence = fgInd
		msg.functionalGroups = append(msg.functionalGroups, fGroup)

		transactionSegments, grpErr := groupNodesByName(
			stSegmentId,
			seSegmentId,
			functionalSegments,
		)
		fGroup.Transactions = make(
			[]*TransactionSetNode,
			0,
			len(transactionSegments),
		)
		newGroupChildren := make(
			[]*X12Node,
			len(transactionSegments)+2,
			len(transactionSegments)+2,
		)
		newGroupChildren[0] = fGroup.Header
		if grpErr != nil {
			validationErrors = append(validationErrors, grpErr)
			err = errors.Join(validationErrors...)
			return err
		}

		// Create transaction sets from stSegmentId/seSegmentId-bounded segments, and set
		// the parent of each segment to the transaction set
		for txInd, stSegments := range transactionSegments {
			t := &X12Node{}
			txnChildren := make([]*X12Node, 0, len(stSegments))
			t.Children = txnChildren
			for ti := 0; ti < len(stSegments); ti++ {
				seg := stSegments[ti]
				t.Children = append(t.Children, seg)
				seg.Parent = t
			}
			transactionSet, err := createTransactionNode(t)
			if err != nil {
				validationErrors = append(validationErrors, err)
				err = errors.Join(validationErrors...)
				return err
			}
			transactionSet.Group = fGroup
			newGroupChildren[txInd+1] = transactionSet.X12Node
			transactionSet.Parent = fGroup.X12Node
			fGroup.Transactions = append(fGroup.Transactions, transactionSet)
			transactionSet.Parent = fGroup.X12Node
		}
		newGroupChildren[len(newGroupChildren)-1] = fGroup.Trailer
		fGroup.Children = newGroupChildren
	}

	newMsgChildren[len(newMsgChildren)-1] = ieaNode
	ieaNode.Parent = msg.X12Node
	msg.Children = newMsgChildren
	validationErrors = append(validationErrors, msg.X12Node.setPath())
	err = errors.Join(validationErrors...)
	return err

}

func textCleanup(msg string, segmentTerminator rune) string {
	msg = strings.TrimSpace(msg)

	var keepChars []string

	for _, c := range strings.Split(extendedCharacterSet, "") {
		keepChars = append(keepChars, c)
	}

	if segmentTerminator == '\n' {
		keepChars = append(keepChars, "\n")
	}
	if segmentTerminator == '\r' {
		keepChars = append(keepChars, "\r")
	}
	if segmentTerminator == '\t' {
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
