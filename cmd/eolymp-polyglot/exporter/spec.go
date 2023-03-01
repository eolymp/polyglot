package exporter

type SpecificationConfig struct {
	Groups     []SpecificationGroup
	Checker    SpecificationChecker
	Statements []SpecificationStatement
}

type SpecificationStatement struct {
	Format string
	Locale string
	Title  string
	Source string
	PDF    string
}

type SpecificationChecker struct {
	Type          string
	Location      string
	Precision     int32
	CaseSensitive bool
}

type SpecificationGroup struct {
	Index          uint32
	TimeLimit      uint32
	MemoryLimit    uint64
	ScoringMode    string
	FeedBackPolicy string
	Dependencies   []uint32
	Scores         []float32
}
