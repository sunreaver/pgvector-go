package pgvector

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"strconv"
	"strings"
)

// Vector is a wrapper for []float64 to implement sql.Scanner and driver.Valuer.
type Vector struct {
	vec []float64
}

// NewVector creates a new Vector from a slice of float64.
func NewVector(vec []float64) Vector {
	return Vector{vec: vec}
}

// Slice returns the underlying slice of float64.
func (v Vector) Slice() []float64 {
	return v.vec
}

// String returns a string representation of the vector.
func (v Vector) String() string {
	var buf strings.Builder
	buf.WriteString("[")

	for i := 0; i < len(v.vec); i++ {
		if i > 0 {
			buf.WriteString(",")
		}
		buf.WriteString(strconv.FormatFloat(float64(v.vec[i]), 'f', -1, 32))
	}

	buf.WriteString("]")
	return buf.String()
}

// Parse parses a string representation of a vector.
func (v *Vector) Parse(s string) error {
	v.vec = make([]float64, 0)
	sp := strings.Split(s[1:len(s)-1], ",")
	for i := 0; i < len(sp); i++ {
		n, err := strconv.ParseFloat(sp[i], 32)
		if err != nil {
			return err
		}
		v.vec = append(v.vec, n)
	}
	return nil
}

// statically assert that Vector implements sql.Scanner.
var _ sql.Scanner = (*Vector)(nil)

// Scan implements the sql.Scanner interface.
func (v *Vector) Scan(src interface{}) (err error) {
	switch src := src.(type) {
	case []byte:
		return v.Parse(string(src))
	case string:
		return v.Parse(src)
	default:
		return fmt.Errorf("unsupported data type: %T", src)
	}
}

// statically assert that Vector implements driver.Valuer.
var _ driver.Valuer = (*Vector)(nil)

// Value implements the driver.Valuer interface.
func (v Vector) Value() (driver.Value, error) {
	return v.String(), nil
}
