package edx12

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
	basicCharacterSet           = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 !\"&'()+*,-./:;?="
	extendedCharacterSet        = basicCharacterSet + "abcdefghijklmnopqrstuvwxyz%~@[]_{}\\|<>^`#$"
	isaElementCount             = 17
	isaByteCount                = 106
	isaElementSeparatorIndex    = 3
	x12PathSeparator            = "/"
	compositeTypeName           = "Composite"
	elementTypeName             = "Element"
)

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
	gsIndexSenderCode
	gsIndexReceiverCode
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
	//isaLenSegmentId                 = 3
	isaLenAuthInfoQualifier     = 2
	isaLenAuthInfo              = 10
	isaLenSecurityInfoQualifier = 2
	isaLenSecurityInfo          = 10
	isaLenSenderIdQualifier     = 2
	isaLenSenderId              = 15
	isaLenReceiverIdQualifier   = 2
	isaLenReceiverId            = 15
	isaLenDate                  = 6
	isaLenTime                  = 4
	//isaLenRepetitionSeparator       = 1
	isaLenVersion        = 5
	isaLenControlNumber  = 9
	isaLenAckRequested   = 1
	isaLenUsageIndicator = 1
	//isaLenComponentElementSeparator = 1
)

const (
	hlIndexHierarchicalId = iota + 1
	hlIndexParentId
	hlIndexLevelCode
	hlIndexChildCode
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
