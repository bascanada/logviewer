package restapi

import (
    "testing"

    "github.com/berlingoqc/logviewer/pkg/ty"
    "github.com/stretchr/testify/assert"
)

func TestBuildSearchJobData_DefaultTimes_NoCustomSearch(t *testing.T) {
    body := buildSearchJobData("index=main", "", "", nil)

    // ensure earliest/latest were defaulted
    assert.Equal(t, "-24h@h", body["earliest_time"])
    assert.Equal(t, "now", body["latest_time"])

    // ensure search param is present and contains the prefixed "search "
    assert.Equal(t, "search index=main", body["search"])

    // ensure custom.search is not present
    _, ok := body["custom.search"]
    assert.False(t, ok)

    // ensure function does not mutate caller's map when nil passed
    m := ty.MS{"foo": "bar"}
    out := buildSearchJobData("index=main", "2020-01-01", "2020-01-02", m)
    assert.Equal(t, "2020-01-02", out["latest_time"])
    assert.Equal(t, "2020-01-01", out["earliest_time"])
}
