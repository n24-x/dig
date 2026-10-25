package dig

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestResourceRejectsDuplicateResources covers the two duplicate checks in
// Scope.checkResource: duplicates within one constructor, and duplicates
// against what the scope already provides.
func TestResourceRejectsDuplicateResources(t *testing.T) {
	t.Parallel()

	// registration is a constructor together with the options to register it
	// with.
	type registration struct {
		ctor any
		opts []Option[*ResourceOptions]
	}

	tests := []struct {
		name string
		// pre are registered before subject; each must succeed.
		pre []registration
		// subject is the registration under test.
		subject registration
		// wantErr is a substring of the expected error message.
		wantErr string
		// wantCount is the expected node count after the failed registration.
		wantCount int
	}{
		{
			name: "same type provided by two constructors",
			pre: []registration{{
				ctor: func() *typ1 { return nil },
			}},
			subject: registration{ctor: func() *typ1 { return nil }},
			// The error names the offending constructor and the conflicting
			// key, but not the constructor that already provides it.
			wantErr:   "result type *dig.typ1 has already been added",
			wantCount: 1,
		},
		{
			name: "named value provided by two constructors",
			pre: []registration{{
				ctor: func() *typ1 { return nil },
				opts: []Option[*ResourceOptions]{Name("ro")},
			}},
			subject: registration{
				ctor: func() *typ1 { return nil },
				opts: []Option[*ResourceOptions]{Name("ro")},
			},
			wantErr:   `result type *dig.typ1[name="ro"] has already been added`,
			wantCount: 1,
		},
		{
			name:    "same type returned twice by one constructor",
			subject: registration{ctor: func() (*typ1, *typ1) { return nil, nil }},
			wantErr: "duplicate result type *dig.typ1",
		},
		{
			name:    "resultDuplicateType: same type in two return-struct fields",
			subject: registration{ctor: func() resultDuplicateType { return resultDuplicateType{} }},
			wantErr: "duplicate result type *dig.typ1",
		},
		{
			name:    "resultSanmeName: same named type in two return-struct fields",
			subject: registration{ctor: func() resultSanmeName { return resultSanmeName{} }},
			wantErr: `duplicate result type *dig.typ1[name="rain"]`,
		},
		{
			// As reports the interface as the provided key, so it conflicts
			// with any constructor that provides that interface directly.
			name: "As collides with an explicitly provided interface",
			pre: []registration{{
				ctor: func() *typ1 { return nil },
				opts: []Option[*ResourceOptions]{As(new(any))},
			}},
			subject:   registration{ctor: func() any { return nil }},
			wantErr:   "result type interface {} has already been added",
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := New()
			for _, pre := range tt.pre {
				require.NoError(t, c.Resource(pre.ctor, pre.opts...))
			}

			err := c.Resource(tt.subject.ctor, tt.subject.opts...)
			require.Error(t, err)
			require.ErrorContains(t, err, tt.wantErr)

			// Every registration failure is attributed to the constructor.
			require.ErrorContains(t, err, "cannot add resource")

			// A failed registration must leave the graph untouched.
			require.Equal(t, tt.wantCount, c.rootScope.dag.Count())
		})
	}
}

// TestResourceDetectsCycle covers the translation of *dag.CycleError into
// errCycleDetected.
func TestResourceDetectsCycle(t *testing.T) {
	t.Parallel()

	t.Run("three node cycle", func(t *testing.T) {
		t.Parallel()

		c := New()
		require.NoError(t, c.Resource(newCycleA))
		require.NoError(t, c.Resource(newCycleB))

		err := c.Resource(newCycleC)
		require.Error(t, err)

		var cycleErr errCycleDetected
		require.ErrorAs(t, err, &cycleErr)
		require.ErrorContains(t, err, "cycle detected")
		require.NotContains(t, err.Error(), "dag:",
			"the dag-go prefix must be stripped")

		// Add rolled the node back.
		require.Equal(t, 2, c.rootScope.dag.Count())
	})

	t.Run("self cycle", func(t *testing.T) {
		t.Parallel()

		c := New()

		err := c.Resource(newSelfCycle)
		require.Error(t, err)

		var cycleErr errCycleDetected
		require.ErrorAs(t, err, &cycleErr)
		require.ErrorContains(t, err, "cycle detected")

		require.Equal(t, 0, c.rootScope.dag.Count())
	})
}

// TestResourceRejectsInvalidConstructor covers the checks Scope.Resource runs
// before parsing the constructor.
func TestResourceRejectsInvalidConstructor(t *testing.T) {
	t.Parallel()

	// Both of these errors are raised before the constructor has a digreflect
	// location, so they carry neither the "cannot add resource" prefix nor a
	// call site.
	tests := []struct {
		name    string
		ctor    any
		wantErr string
	}{
		{
			name:    "nil constructor",
			ctor:    nil,
			wantErr: "dig: nil cannot provide resource",
		},
		{
			name:    "non-function constructor",
			ctor:    "chongqing",
			wantErr: "must be function, got chongqing (type string)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := New()
			err := c.Resource(tt.ctor)
			require.ErrorContains(t, err, tt.wantErr)
			require.Equal(t, 0, c.rootScope.dag.Count())
		})
	}
}

// TestResourceOptionValidation covers ResourceOptions.Validate. Every invalid
// option combination is rejected before the constructor is parsed.
func TestResourceOptionValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		opts    []Option[*ResourceOptions]
		wantErr string
	}{
		{
			name:    "name and group are mutually exclusive",
			opts:    []Option[*ResourceOptions]{Name("ro"), Group("rain")},
			wantErr: `cannot use name-tag and group-tag together: name:"ro" group:"rain"`,
		},
		{
			name:    "name cannot contain backquotes",
			opts:    []Option[*ResourceOptions]{Name("r`o")},
			wantErr: "names cannot contain backquotes",
		},
		{
			name:    "group cannot contain backquotes",
			opts:    []Option[*ResourceOptions]{Group("r`ain")},
			wantErr: "group names cannot contain backquotes",
		},
		{
			name:    "As(nil) is rejected",
			opts:    []Option[*ResourceOptions]{As(nil)},
			wantErr: "invalid dig.As(nil): argument must be a pointer to an interface",
		},
		{
			name:    "As of a non-pointer is rejected",
			opts:    []Option[*ResourceOptions]{As(typ1{})},
			wantErr: "invalid dig.As(dig.typ1): argument must be a pointer to an interface",
		},
		{
			name:    "As of a pointer to a non-interface is rejected",
			opts:    []Option[*ResourceOptions]{As(new(*typ1))},
			wantErr: "invalid dig.As(**dig.typ1): argument must be a pointer to an interface",
		},
		{
			name:    "As of error is rejected",
			opts:    []Option[*ResourceOptions]{As(new(error))},
			wantErr: "invalid dig.As(new(error)): cannot use error in As()",
		},
		{
			name:    "duplicate As targets are rejected",
			opts:    []Option[*ResourceOptions]{As(new(any)), As(new(any))},
			wantErr: "duplicate type in As: *interface {}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := New()
			err := c.Resource(func() *typ1 { return nil }, tt.opts...)
			require.ErrorContains(t, err, tt.wantErr)

			// Validate runs before the constructor is parsed, so nothing was
			// registered and the error is not attributed to a constructor.
			require.NotContains(t, err.Error(), "cannot add resource")
			require.Equal(t, 0, c.rootScope.dag.Count())
		})
	}
}

// TestResourceReceipt covers the receipt written by a successful registration.
func TestResourceReceipt(t *testing.T) {
	t.Parallel()

	t.Run("receipt describes the registered node", func(t *testing.T) {
		t.Parallel()

		c := New()
		require.NoError(t, c.Resource(func() *typ2 { return nil }))

		var info ResourceReceiptInfo
		require.NoError(t, c.Resource(newReceiptCtor, ResourceReceipt(&info)))

		require.NotEqual(t, DagID(0), info.DagID)
		require.Len(t, info.Params, 1)
		require.Equal(t, "*dig.typ2", info.Params[0].String())
		require.Len(t, info.Results, 1)
		require.Equal(t, "*dig.typ1", info.Results[0].String())
	})

	t.Run("failed registration leaves the receipt zero", func(t *testing.T) {
		t.Parallel()

		c := New()
		require.NoError(t, c.Resource(func() *typ1 { return nil }))

		var info ResourceReceiptInfo
		err := c.Resource(func() *typ1 { return nil }, ResourceReceipt(&info))
		require.Error(t, err)

		require.Equal(t, DagID(0), info.DagID)
		require.Empty(t, info.Params)
		require.Empty(t, info.Results)
	})
}
