package kafkazk

import (
	"errors"
	"fmt"
	"sort"
)

var (
	errInvalidSelectionMethod = errors.New("Invalid selection method")
)

type ErrNoBrokers struct {
	Message string
}

func (e *ErrNoBrokers) Error() string {
	return e.Message
}

// Constraints holds a map of
// IDs and locality key-values.
type constraints struct {
	requestSize float64
	locality    map[string]bool
	id          map[int]bool
}

// newConstraints returns an empty *constraints.
func newConstraints() *constraints {
	return &constraints{
		locality: make(map[string]bool),
		id:       make(map[int]bool),
	}
}

// bestCandidate takes a *constraints, selection method
// and pass / iteration number (for use as a seed value
// for pseudo-random number generation) and returns
// the most suitable broker.
func (b brokerList) bestCandidate(c *constraints, by string, p int64) (*Broker, error) {
	// Sort type based on the
	// desired placement criteria.
	switch by {
	case "count":
		// XXX Should instantiate
		// a dedicated Rand for this.
		b.SortPseudoShuffle(p)
	case "storage":
		sort.Sort(brokersByStorage(b))
	default:
		return nil, errInvalidSelectionMethod
	}

	var candidate *Broker

	// Iterate over candidates.
	var err error
	for _, candidate = range b {
		// Candidate passes, return.
		if err = c.passes(candidate); err == nil {
			c.add(candidate)
			candidate.Used++

			return candidate, nil
		}
	}

	// List exhausted, no brokers passed.
	// Return last error.
	return nil, err
}

// add takes a *Broker and adds its
// attributes to the *constraints.
// The requestSize is also subtracted
// from the *Broker.StorageFree.
func (c *constraints) add(b *Broker) {
	b.StorageFree -= c.requestSize

	if b.Locality != "" {
		c.locality[b.Locality] = true
	}

	c.id[b.ID] = true
}

// passes takes a *Broker and returns
// an error if it does not pass constraints.
func (c *constraints) passes(b *Broker) error {
	switch {
	// Fail if the candidate would run
	// out of storage.
	// TODO this needs thresholds and
	// more intelligent controls.
	case b.StorageFree-c.requestSize < 0:
		return &ErrNoBrokers{
			Message: fmt.Sprintf("Locality %s exhausted of storage capacity", b.Locality),
		}
	// Fail if the candidate is in any of
	// the existing replica set localities.
	case c.locality[b.Locality]:
		return &ErrNoBrokers{
			Message: fmt.Sprintf("Locality %s exhausted of available brokers", b.Locality),
		}
	// Fail if the candidate is one of the
	// IDs already in the replica set.
	case c.id[b.ID]:
		return &ErrNoBrokers{
			Message: fmt.Sprintf("ID %d already in replica set", b.ID),
		}
	}

	return nil
}

// mergeConstraints takes a brokerlist and
// builds a *constraints by merging the
// attributes of all brokers from the supplied list.
func mergeConstraints(bl brokerList) *constraints {
	c := newConstraints()

	for _, b := range bl {
		// Don't merge in attributes
		// from nodes that will be removed.
		if b.Replace {
			continue
		}

		if b.Locality != "" {
			c.locality[b.Locality] = true
		}

		c.id[b.ID] = true
	}

	return c
}
