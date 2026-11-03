// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package platform_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tfior/doc-tracker/platform"
)

func day(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

// Scenario 11: the earliest and latest day a partial date covers.
func TestPartialDate_Bounds(t *testing.T) {
	cases := []struct{ date, earliest, latest string }{
		{"1890", "1890-01-01", "1890-12-31"},
		{"1890-03", "1890-03-01", "1890-03-31"},
		{"1890-04", "1890-04-01", "1890-04-30"},
		{"1890-02", "1890-02-01", "1890-02-28"},
		{"1892-02", "1892-02-01", "1892-02-29"}, // leap year
		{"1900-02", "1900-02-01", "1900-02-28"}, // century, not a leap year
		{"2000-02", "2000-02-01", "2000-02-29"}, // divisible by 400, a leap year
		{"1890-12", "1890-12-01", "1890-12-31"},
		{"1890-03-15", "1890-03-15", "1890-03-15"},
	}
	for _, tc := range cases {
		t.Run(tc.date, func(t *testing.T) {
			d, ok := platform.ParsePartialDate(tc.date)
			require.True(t, ok)
			assert.Equal(t, day(tc.earliest), d.Earliest())
			assert.Equal(t, day(tc.latest), d.Latest())
			assert.Equal(t, tc.date, d.String())
		})
	}
}

func TestPartialDate_Parse(t *testing.T) {
	for _, s := range []string{"1890", "1890-03", "1890-03-15", "1892-02-29", "0001"} {
		_, ok := platform.ParsePartialDate(s)
		assert.True(t, ok, s)
	}
	for _, s := range []string{
		"", "90", "1890-3", "1890-03-5", "15/03/1890", "1890-03-15T00:00", " 1890",
		"0000", "1890-00", "1890-13", "1890-03-00", "1890-02-29", "1890-04-31", "1890-03-32",
	} {
		_, ok := platform.ParsePartialDate(s)
		assert.False(t, ok, s)
	}
}

// Scenario 11: the earliest and latest day a qualified date allows. A missing
// bound is open: "before 1915" has no earliest day.
func TestQualifiedDate_Bounds(t *testing.T) {
	end := func(s string) *string { return &s }
	cases := []struct {
		name             string
		date             platform.QualifiedDate
		earliest, latest string // "" for open
	}{
		{"exact day", platform.QualifiedDate{Value: "1890-03-15", Qualifier: "exact"}, "1890-03-15", "1890-03-15"},
		{"exact month", platform.QualifiedDate{Value: "1890-03", Qualifier: "exact"}, "1890-03-01", "1890-03-31"},
		{"about is not widened", platform.QualifiedDate{Value: "1890", Qualifier: "about"}, "1890-01-01", "1890-12-31"},
		{"before", platform.QualifiedDate{Value: "1915", Qualifier: "before"}, "", "1914-12-31"},
		{"before a day", platform.QualifiedDate{Value: "1892-03-01", Qualifier: "before"}, "", "1892-02-29"},
		{"after", platform.QualifiedDate{Value: "1888-06", Qualifier: "after"}, "1888-07-01", ""},
		{"after year end", platform.QualifiedDate{Value: "1888-12-31", Qualifier: "after"}, "1889-01-01", ""},
		{"between", platform.QualifiedDate{Value: "1890", Qualifier: "between", End: end("1892")}, "1890-01-01", "1892-12-31"},
		{"between months", platform.QualifiedDate{Value: "1890-03-15", Qualifier: "between", End: end("1890-04")}, "1890-03-15", "1890-04-30"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			earliest, ok := tc.date.Earliest()
			if tc.earliest == "" {
				assert.False(t, ok, "earliest should be open")
			} else if assert.True(t, ok) {
				assert.Equal(t, day(tc.earliest), earliest)
			}
			latest, ok := tc.date.Latest()
			if tc.latest == "" {
				assert.False(t, ok, "latest should be open")
			} else if assert.True(t, ok) {
				assert.Equal(t, day(tc.latest), latest)
			}
		})
	}
}
