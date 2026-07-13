// Copyright (c) 2019 Uber Technologies, Inc.
// Copyright (c) 2026 k2 <skrik2@outlook.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package digclock

import (
	"testing"
	"time"
)

func testClock(t *testing.T, clock Clock, advance func(d time.Duration)) {
	now := clock.Now()
	if now.IsZero() {
		t.Error("expected non-zero time")
	}

	t.Run("Since", func(t *testing.T) {
		advance(1 * time.Millisecond)
		if clock.Since(now) == 0 {
			t.Error("time must have advanced")
		}
	})
}

func TestSystemClock(t *testing.T) {
	clock := System
	testClock(t, clock, func(d time.Duration) { time.Sleep(d) })
}

func TestMockClock(t *testing.T) {
	clock := NewMock()
	testClock(t, clock, clock.Add)
}

func TestMock_AddNegative(t *testing.T) {
	clock := NewMock()
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when adding negative duration")
		}
	}()
	clock.Add(-1)
}
