package dig

// Fixtures for Scope.Resource tests.
//
// Fixtures that only exercise constructor parsing live in
// testing_testsdata_1_test.go; the ones here are about registering into a
// scope: duplicate keys and cycles.

// Two fields of the same type provide the same key twice.
type resultDuplicateType struct {
	Provided

	A *typ1
	B *typ1
}

// Two fields of the same type and the same name provide the same named key
// twice.
type resultSanmeName struct {
	Provided

	A *typ1 `name:"rain"`
	B *typ1 `name:"rain"`
}

// cycleA -> cycleC -> cycleB -> cycleA forms a three-node cycle.
type (
	cycleA struct{}
	cycleB struct{}
	cycleC struct{}
)

func newCycleA(c *cycleC) *cycleA { return nil }
func newCycleB(a *cycleA) *cycleB { return nil }
func newCycleC(b *cycleB) *cycleC { return nil }

// newSelfCycle consumes and provides the same type, so a single node closes a
// cycle on itself.
func newSelfCycle(a *cycleA) *cycleA { return nil }

// newReceiptCtor requires one resource and provides one, so its receipt has
// exactly one param and one result.
func newReceiptCtor(*typ2) *typ1 { return nil }
