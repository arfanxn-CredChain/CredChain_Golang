package query

type Operator string

const (
	OperatorEqual              Operator = "="
	OperatorNotEqual           Operator = "!="
	OperatorGreaterThan        Operator = ">"
	OperatorLessThan           Operator = "<"
	OperatorGreaterThanOrEqual Operator = ">="
	OperatorLessThanOrEqual    Operator = "<="
	OperatorLike               Operator = "~"
	OperatorILike              Operator = "~*"
	OperatorNotLike            Operator = "!~"
	OperatorNotILike           Operator = "!~*"
	OperatorIn                 Operator = "$"
	OperatorNotIn              Operator = "!$"
	OperatorBetween            Operator = ".."
	OperatorNotBetween         Operator = "!.."
	OperatorNull               Operator = "_"
	OperatorNotNull            Operator = "!_"
)

var AllOperators = []Operator{
	OperatorNotEqual,
	OperatorGreaterThanOrEqual,
	OperatorLessThanOrEqual,
	OperatorNotLike,
	OperatorNotILike,
	OperatorNotIn,
	OperatorNotBetween,
	OperatorNotNull,
	OperatorILike,
	OperatorBetween,
	OperatorEqual,
	OperatorGreaterThan,
	OperatorLessThan,
	OperatorLike,
	OperatorIn,
	OperatorNull,
}

type SortOrder string

const (
	SortAsc  SortOrder = "ASC"
	SortDesc SortOrder = "DESC"
)

var AllSortOrders = []SortOrder{
	SortAsc,
	SortDesc,
}
