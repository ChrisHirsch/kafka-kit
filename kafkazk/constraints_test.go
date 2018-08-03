package kafkazk

import (
	"testing"
)

func TestBestCandidateByCount(t *testing.T) {
	localities := []string{"a", "b", "c"}
	bl := brokerList{}

	for i := 0; i < 4; i++ {
		b := &Broker{
			ID:       1000 + i,
			Locality: localities[i%3],
			Used:     i,
		}

		bl = append(bl, b)
	}

	c := newConstraints()
	// Removes ID 1000 as a candidate.
	c.id[1000] = true
	// Removes any brokers with locality
	// "b" as candidates.
	c.locality["b"] = true

	b, _ := bl.bestCandidate(c, "count", 1)
	// 1002 should be the first available.
	if b.ID != 1002 {
		t.Errorf("Expected candidate with ID 1002, got %d", b.ID)
	}

	b, _ = bl.bestCandidate(c, "count", 1)
	// 1003 should be next available.
	if b.ID != 1003 {
		t.Errorf("Expected candidate with ID 1003, got %d", b.ID)
	}

	_, err := bl.bestCandidate(c, "count", 1)
	if err == nil {
		t.Errorf("Expected exhausted candidate list")
	}
}

func TestBestCandidateByStorage(t *testing.T) {
	localities := []string{"a", "b", "c"}
	bl := brokerList{}

	for i := 0; i < 4; i++ {
		b := &Broker{
			ID:          1000 + i,
			Locality:    localities[i%3],
			Used:        i,
			StorageFree: float64(1000 * i),
		}

		bl = append(bl, b)
	}

	c := newConstraints()
	// Removes any brokers with locality
	// "b" as candidates.
	c.locality["c"] = true
	// Sets request size.
	c.requestSize = 1000.00

	b, _ := bl.bestCandidate(c, "storage", 1)

	// 1003 should be the first available.
	if b.ID != 1003 {
		t.Errorf("Expected candidate with ID 1003, got %d", b.ID)
	}

	b, _ = bl.bestCandidate(c, "storage", 1)
	// 1003 should be next available.
	if b.ID != 1001 {
		t.Errorf("Expected candidate with ID 1001, got %d", b.ID)
	}

	_, err := bl.bestCandidate(c, "storage", 1)
	if err == nil {
		t.Errorf("Expected exhausted candidate list")
	}
}

func TestConstraintsAdd(t *testing.T) {
	b1 := &Broker{ID: 1000, Locality: "a"}
	b2 := &Broker{ID: 1001, Locality: "b"}

	bl := brokerList{b1}
	c := mergeConstraints(bl)
	c.add(b2)

	bl = append(bl, b2)

	for _, b := range bl {
		if _, exists := c.id[b.ID]; !exists {
			t.Errorf("Expected ID %d to exist", b.ID)
		}
		if _, exists := c.locality[b.Locality]; !exists {
			t.Errorf("Expected ID %d to exist", b.ID)
		}
	}
}

func TestConstraintsPasses(t *testing.T) {
	c := newConstraints()
	c.locality["a"] = true
	c.id[1000] = true

	// Fail on ID and/or locality.
	b1 := &Broker{ID: 1000, Locality: "a"}
	// Fail on locality only.
	b2 := &Broker{ID: 1001, Locality: "a"}
	// Fail on ID only.
	b3 := &Broker{ID: 1000, Locality: "b"}
	// Pass.
	b4 := &Broker{ID: 1001, Locality: "c"}

	bl := brokerList{b1, b2, b3, b4}

	expected := []string{
		"ID 1000 already in replica set",
		"Locality a already used in replica set",
		"ID 1000 already in replica set",
	}

	for i, b := range bl {
		if e := c.passes(b); e != nil && e.Error() != expected[i] {
			t.Errorf("Expected broker %d return constraint check with error: %s, got: %s", b.ID, expected[i], e)
		}
	}
}

func TestMergeConstraints(t *testing.T) {
	localities := []string{"a", "b", "c"}
	bl := brokerList{}

	for i := 0; i < 5; i++ {
		b := &Broker{
			ID:       1000 + i,
			Locality: localities[i%3],
		}

		// Brokers marked for replacement
		// don't get merged in.
		if i == 4 {
			b.Replace = true
		}

		bl = append(bl, b)
	}

	c := mergeConstraints(bl)

	// Check expected.
	for i := 1000; i < 1004; i++ {
		if _, exists := c.id[i]; !exists {
			t.Errorf("Expected ID %d to exist", i)
		}
	}

	for _, l := range localities {
		if _, exists := c.locality[l]; !exists {
			t.Errorf("Expected locality %s to exist", l)
		}
	}

	// Check excluded.
	if _, exists := c.id[1004]; exists {
		t.Error("ID 1004 shouldn't exist in the constraints")
	}
}
