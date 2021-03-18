package main

type Specification struct {
	Names      []SpecificationName      `xml:"names>name"`
	Statements []SpecificationStatement `xml:"statements>statement"`
	Files      []SpecificationFile      `xml:"files>executables>executable"`
	Judging    SpecificationJudging     `xml:"judging"`
	Checker    SpecificationChecker     `xml:"assets>checker"`
	Interactor SpecificationInteractor  `xml:"assets>interactor"`
}

type SpecificationName struct {
	Language string `xml:"language,attr"`
	Value    string `xml:"value,attr"`
}

type SpecificationStatement struct {
	Charset  string `xml:"charset,attr"`
	Language string `xml:"language,attr"`
	MathJAX  bool   `xml:"mathjax,attr"`
	Path     string `xml:"path,attr"`
	Type     string `xml:"type,attr"`
}

type SpecificationFile struct {
	Source  SpecificationSource `xml:"source"`
	Binarie SpecificationBinary `xml:"binary"`
}

type SpecificationJudging struct {
	Testsets []SpecificationTestset `xml:"testset"`
}

type SpecificationTestset struct {
	Name              string               `xml:"name,attr"`
	TimeLimit         int                  `xml:"time-limit"`
	MemoryLimit       int                  `xml:"memory-limit"`
	TestCount         int                  `xml:"test-count"`
	InputPathPattern  string               `xml:"input-path-pattern"`
	AnswerPathPattern string               `xml:"answer-path-pattern"`
	Tests             []SpecificationTest  `xml:"tests>test"`
	Groups            []SpecificationGroup `xml:"groups>group"`
}

type SpecificationTest struct {
	Method  string  `xml:"method,attr"`
	Group   uint32  `xml:"group,attr"`
	Command string  `xml:"cmd,attr"`
	Sample  bool    `xml:"sample,attr"`
	Points  float32 `xml:"points,attr"`
}

type SpecificationGroup struct {
	FeedbackPolicy string                    `xml:"feedback-policy,attr"`
	Name           uint32                    `xml:"name,attr"`
	Points         float32                   `xml:"points,attr"`
	PointsPolicy   string                    `xml:"points-policy,attr"`
	Dependencies   []SpecificationDependency `xml:"dependencies>dependency"`
}

type SpecificationDependency struct {
	Group uint32 `xml:"group,attr"`
}

type SpecificationChecker struct {
	Name     string                `xml:"name,attr"`
	Type     string                `xml:"type,attr"`
	Sources  []SpecificationSource `xml:"source"`
	Binaries []SpecificationBinary `xml:"binary"`
}

type SpecificationInteractor struct {
	Name     string                `xml:"name,attr"`
	Sources  []SpecificationSource `xml:"source"`
	Binaries []SpecificationBinary `xml:"binary"`
}

type SpecificationBinary struct {
	Path string `xml:"path,attr"`
	Type string `xml:"type,attr"`
}

type SpecificationSource struct {
	Path string `xml:"path,attr"`
	Type string `xml:"type,attr"`
}

type PolygonProblemProperties struct {
	Language    string `json:"language"`
	Name        string `json:"name"`
	Legend      string `json:"legend"`
	Input       string `json:"input"`
	Interaction string `json:"interaction"`
	Output      string `json:"output"`
	Notes       string `json:"notes"`
	Scoring     string `json:"scoring"`
	AuthorLogin string `json:"authorLogin"`
	AuthorName  string `json:"authorName"`
}

func SourceByType(sources []SpecificationSource, types ...string) (*SpecificationSource, bool) {
	v := map[string]bool{}
	for _, t := range types {
		v[t] = true
	}

	for _, s := range sources {
		if v[s.Type] {
			return &s, true
		}
	}

	return nil, false
}
