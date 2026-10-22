package dig

type typ1 struct{}

type typ2 struct{}

type typ3 struct{}

// ---- Good cases: params (simple -> complex) ----

type requiredParamsGoodData1 struct {
	Required

	A *typ1
	B *typ2 `name:"lee"`
}

var _ = requiredParamsGoodData1{}

type optionalParamsGoodData2 struct {
	Required

	A *typ1 `optional:"true"`
	B *typ2
}

var _ = optionalParamsGoodData2{}

type groupParamsGoodData3 struct {
	Required

	A []*typ1 `group:"rain"`
}

var _ = groupParamsGoodData3{}

type passiveGroupParamsGoodData4 struct {
	Required

	A int
	B []*typ1 `group:"rain,passive"`
}

var _ = passiveGroupParamsGoodData4{}

type ignoreUnexportedParamsGoodData5 struct {
	Required `ignore-unexported:"true"`

	A []*typ1
	b typ2
}

var _ = ignoreUnexportedParamsGoodData5{}

// ---- Good cases: results (simple -> complex) ----

type providedResultsGoodData7 struct {
	Provided

	A *typ1
	B *typ2 `name:"lee"`
}

var _ = providedResultsGoodData7{}

type groupResultsGoodData8 struct {
	Provided

	A *typ1 `group:"rain"`
}

var _ = groupResultsGoodData8{}

type flattenResultsGoodData9 struct {
	Provided

	A []*typ1 `group:"rain,flatten"`
}

var _ = flattenResultsGoodData9{}

// Two fields of the same type contribute to the same group: legal, because a
// group accepts multiple values for one key.
type duplicateGroupResultsGoodData10 struct {
	Provided

	A *typ1 `group:"rain"`
	B *typ1 `group:"rain"`
}

var _ = duplicateGroupResultsGoodData10{}

// Two fields of different types share a group name: legal, and they form two
// distinct group keys, because the value type is part of the key.
type mixedGroupResultsGoodData11 struct {
	Provided

	A *typ1 `group:"rain"`
	B *typ2 `group:"rain"`
}

var _ = mixedGroupResultsGoodData11{}

// ---- Bad cases (simple -> complex) ----

// Bad Data 1
// param side: optional value is not a valid bool
type paramsBadData1 struct {
	Required

	A *typ1 `optional:"janus"`
}

var _ = paramsBadData1{}

// Bad Data 2
// result side: optional is not allowed
type resultsBadData2 struct {
	Provided

	A *typ1 `optional:"ture"`
}

var _ = resultsBadData2{}

// Bad Data 3
// param side: group can only be used on slices
type paramsBadData3 struct {
	Required

	A *typ1 `group:"rain"`
}

var _ = paramsBadData3{}

// Bad Data 4
// param side: group conflicts with name
type paramsBadData4 struct {
	Required

	A []*typ1 `group:"rain" name:"lee" optional:"true"`
}

var _ = paramsBadData4{}

// Bad Data 5
// result side: group conflicts with name/optional
type resultsBadData5 struct {
	Provided

	A *typ1 `group:"rain" name:"lee" optional:"true"`
}

var _ = resultsBadData5{}

// Bad Data 6
// result side: error is not a valid result field
type resultsBadData6 struct {
	Provided

	A         *typ1
	ErrorType error
}

var _ = resultsBadData6{}

// Bad Data 7
// param side: unexported field
type paramsBadData7 struct {
	Required

	a *typ1
	c *typ3
}

var _ = paramsBadData7{}

// Bad Data 8
// result side: unexported field (ignore-unexported does not apply to Provided)
type resultsBadData8 struct {
	Provided `ignore-unexported:"true"`

	A *typ1

	c *typ3
}

var _ = resultsBadData8{}

// Bad Data 9
// param side: embeds *Required
type paramsBadData9 struct {
	*Required

	A *typ1
}

var _ = paramsBadData9{}

// Bad Data 10
// result side: embeds *Provided
type resultsBadData10 struct {
	*Provided

	A *typ1
}

var _ = resultsBadData10{}

// Bad Data 11
// a Provided struct used as a param
type paramsBadData11 struct {
	Provided

	A *typ1
}

var _ = paramsBadData11{}

// Bad Data 12
// a parameter-struct used as a result
type resultsBadData12 struct {
	Required

	A *typ1
}

var _ = resultsBadData12{}

// Bad Data 13
// param side: flatten is not allowed in parameter-struct
type paramsBadData13 struct {
	Required

	A []*typ1 `group:"rain,flatten"`
}

var _ = paramsBadData13{}

// Bad Data 14
// param side: error is not a valid dependency
type paramsBadData14 struct {
	Required

	A         *typ1
	ErrorType error
}

var _ = paramsBadData14{}

// Bad Data 15
// param side: error nested in a parameter-struct field
type paramsBadData15 struct {
	Required

	A     *typ1
	Inner paramsBadData15Inner
}

type paramsBadData15Inner struct {
	Required

	ErrorType error
}

var (
	_ = paramsBadData15{}
	_ = paramsBadData15Inner{}
)

// Bad Data 16
// result side: error nested in a return-struct field
type resultsBadData16 struct {
	Provided

	A     *typ1
	Inner resultsBadData16Inner
}

type resultsBadData16Inner struct {
	Provided

	ErrorType error
}

var (
	_ = resultsBadData16{}
	_ = resultsBadData16Inner{}
)
