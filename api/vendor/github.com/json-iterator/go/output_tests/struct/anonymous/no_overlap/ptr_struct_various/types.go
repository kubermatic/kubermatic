package test

// Embedded TEST ONLY
type Embedded struct {
	String string
	Int    int32
	Float  float64
	Struct struct {
		X string
	}
	Slice []string
	Map   map[string]string
}

type typeForTest struct {
	*Embedded
}
