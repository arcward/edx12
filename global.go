package edx12

import (
	"encoding/json"
)

// NodeType is an enum for the type of node in the X12 tree
type NodeType uint

const (
	UnknownNode NodeType = iota
	MessageNode
	GroupNode
	TransactionNode
	LoopNode
	SegmentNode
	CompositeNode
	RepeatElementNode
	ElementNode
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
	}[n]
}

func (n NodeType) MarshalJSON() ([]byte, error) {
	nodeNames := map[NodeType]string{
		UnknownNode:       "",
		MessageNode:       "Message",
		GroupNode:         "Group",
		TransactionNode:   "TransactionSet",
		LoopNode:          "Loop",
		SegmentNode:       "Segment",
		CompositeNode:     "Composite",
		RepeatElementNode: "RepeatElement",
		ElementNode:       "Element",
	}
	return json.Marshal(nodeNames[n])
}

func (n *NodeType) UnmarshalJSON(b []byte) error {
	nodeNames := map[string]NodeType{
		"":               UnknownNode,
		"Message":        MessageNode,
		"Group":          GroupNode,
		"TransactionSet": TransactionNode,
		"Loop":           LoopNode,
		"Segment":        SegmentNode,
		"Composite":      CompositeNode,
		"RepeatElement":  RepeatElementNode,
		"Element":        ElementNode,
	}
	var nodeName string
	if err := json.Unmarshal(b, &nodeName); err != nil {
		return err
	}
	*n = nodeNames[nodeName]
	return nil
}

var nodeChildTypes = map[NodeType][]NodeType{
	SegmentNode: {
		ElementNode,
		CompositeNode,
		RepeatElementNode,
	},
	LoopNode:          {SegmentNode, LoopNode},
	MessageNode:       {SegmentNode, GroupNode},
	GroupNode:         {SegmentNode, TransactionNode},
	TransactionNode:   {SegmentNode, LoopNode},
	CompositeNode:     {ElementNode},
	RepeatElementNode: {ElementNode},
	ElementNode:       {},
}

const (
	isaSegmentId = "ISA"
	ieaSegmentId = "IEA"
	gsSegmentId  = "GS"
	geSegmentId  = "GE"
	stSegmentId  = "ST"
	seSegmentId  = "SE"
	//lsSegmentId                   = "LS"
	//lsSegmentLoopIdIndex          = 1
	//leSegmentLoopIdIndex          = 1
	//leSegmentId                   = "LE"
	hlSegmentId                 = "HL"
	functionalGroupName         = "GS_LOOP"
	transactionSetName          = "ST_LOOP"
	transactionSpecDescription  = "Transaction set spec"
	transactionSetCodeSpecLabel = "transactionSetCode"
	transactionVersionSpecLabel = "versionCode"
	transactionSpecLabel        = "transactionSets"
	//numericSymbol                 = "N0"
	//decimalSymbol                 = "R"
	identifierSymbol = "ID"
	stringSymbol     = "AN"
	//dateSymbol                    = "DT"
	//timeSymbol                    = "TM"
	//binarySymbol                  = "B"
	basicCharacterSet        = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 !\"&'()+*,-./:;?="
	extendedCharacterSet     = basicCharacterSet + "abcdefghijklmnopqrstuvwxyz%~@[]_{}\\|<>^`#$"
	isaElementCount          = 17
	isaByteCount             = 106
	isaElementSeparatorIndex = 3
	x12PathSeparator         = "/"
	compositeTypeName        = "Composite"
	elementTypeName          = "Element"
)

var functionalIdentifierCodes = map[string]string{
	"270": "HS",
	"271": "HB",
	"276": "HR",
	"277": "HN",
	"278": "HI",
	"820": "RA",
	"834": "HP",
	"835": "HP",
	"837": "HC",
	"997": "FA",
	"999": "FA",
}

const (
	isaIndexSegmentId = iota
	isaIndexAuthInfoQualifier
	isaIndexAuthInfo
	isaIndexSecurityInfoQualifier
	isaIndexSecurityInfo
	isaIndexSenderIdQualifier
	isaIndexSenderId
	isaIndexReceiverIdQualifier
	isaIndexReceiverId
	isaIndexDate
	isaIndexTime
	isaIndexRepetitionSeparator
	isaIndexVersion
	isaIndexControlNumber
	isaIndexAckRequested
	isaIndexUsageIndicator
	isaIndexComponentElementSeparator
)

const (
	ieaIndexFunctionalGroupCount = iota + 1
	ieaIndexControlNumber
)

const (
	gsIndexFunctionalIdentifierCode = iota + 1
	gsIndexReceiverCode
	gsIndexSenderCode
	gsIndexDate
	gsIndexTime
	gsIndexControlNumber
	gsIndexResponsibleAgencyCode
	gsIndexVersion
)

const (
	geIndexNumberOfIncludedTransactionSets = iota + 1
	geIndexControlNumber
)

const (
	stIndexTransactionSetCode = iota + 1
	stIndexControlNumber
	stIndexVersionCode
)

const (
	seIndexNumberOfIncludedSegments = iota + 1
	seIndexControlNumber
)

// isaLen* consts indicate the length of elements in the ISA
// header (no more, no less, whitespace padded on the left)
const (
	isaLenSegmentId                 = 3
	isaLenAuthInfoQualifier         = 2
	isaLenAuthInfo                  = 10
	isaLenSecurityInfoQualifier     = 2
	isaLenSecurityInfo              = 10
	isaLenSenderIdQualifier         = 2
	isaLenSenderId                  = 15
	isaLenReceiverIdQualifier       = 2
	isaLenReceiverId                = 15
	isaLenDate                      = 6
	isaLenTime                      = 4
	isaLenRepetitionSeparator       = 1
	isaLenVersion                   = 5
	isaLenControlNumber             = 9
	isaLenAckRequested              = 1
	isaLenUsageIndicator            = 1
	isaLenComponentElementSeparator = 1
)

const (
	hlIndexHierarchicalId = iota + 1
	hlIndexParentId
	hlIndexLevelCode
	hlIndexChildCode
)
