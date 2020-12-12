package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"github.com/eolymp/contracts/go/eolymp/atlas"
	"github.com/eolymp/contracts/go/eolymp/executor"
	"github.com/eolymp/contracts/go/eolymp/judge"
	"github.com/eolymp/go-packages/env"
	"github.com/eolymp/go-packages/httpx"
	"github.com/eolymp/go-packages/oauth"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	index := flag.Int("index", 0, "Problem Index")
	score := flag.Int("score", 0, "Problem Score")
	flag.Parse()

	client := &Client{cli: httpx.NewClient(
		&http.Client{Timeout: 10 * time.Second},
		httpx.WithCredentials(oauth.PasswordCredentials(
			oauth.NewClient(env.StringDefault("EOLYMP_API_URL", "https://api.e-olymp.com")),
			env.String("EOLYMP_USERNAME"),
			env.String("EOLYMP_PASSWORD"),
		)),
	)}

	contest, path := flag.Arg(0), flag.Arg(1)

	if path == "" {
		log.Println("Path argument is empty")
		flag.Usage()
		os.Exit(-1)
	}

	if contest == "" {
		log.Println("Contest argument is empty")
		flag.Usage()
		os.Exit(-1)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Printf("Import path %#v is invalid: %v", path, err)
		os.Exit(-1)
	}

	spec := &Specification{}

	specf, err := os.Open(filepath.Join(path, "problem.xml"))
	if err != nil {
		log.Printf("Unable to open problem.xml: %v", err)
		os.Exit(-1)
	}

	defer func() {
		_ = specf.Close()
	}()

	if err := xml.NewDecoder(specf).Decode(&spec); err != nil {
		log.Printf("Unable to parse problem.xml: %v", err)
		os.Exit(-1)
	}

	// get testset
	if len(spec.Judging.Testsets) != 1 {
		log.Printf("Specification in problem.xml contains more than one testset, it's not supported yet")
		os.Exit(-1)
	}

	testset := spec.Judging.Testsets[0]

	// convert verifier
	verifier, err := MakeVerifier(path, spec)
	if err != nil {
		log.Printf("Unable to create E-Olymp verifier from specification in problem.xml: %v", err)
		os.Exit(-1)
	}

	// compose problem object for E-Olymp
	problem := &judge.Problem{
		Index:         int32(*index),
		Score:         int32(*score),
		TimeLimit:     int32(testset.TimeLimit),
		MemoryLimit:   int32(testset.MemoryLimit),
		FileSizeLimit: int32(testset.MemoryLimit),
		Verifier:      verifier,
	}

	// get all tests
	for ti, ts := range testset.Tests {
		log.Printf("Adding %v test %v (example: %v)", ts.Method, ti, ts.Sample)

		input, err := MakeObject(client, filepath.Join(path, fmt.Sprintf(testset.InputPathPattern, ti+1)))
		if err != nil {
			log.Printf("Unable to upload test input data to E-Olymp: %v", err)
			os.Exit(-1)
		}

		answer, err := MakeObject(client, filepath.Join(path, fmt.Sprintf(testset.AnswerPathPattern, ti+1)))
		if err != nil {
			log.Printf("Unable to upload test answer data to E-Olymp: %v", err)
			os.Exit(-1)
		}

		problem.Tests = append(problem.Tests, &judge.Problem_Test{
			Index:          int32(ti + 1),
			Example:        ts.Sample,
			InputObjectId:  input,
			AnswerObjectId: answer,
		})
	}

	// get all statements
	for _, ss := range spec.Statements {
		if ss.Type != "application/x-tex" {
			continue
		}

		log.Printf("Adding statement in %#v", ss.Language)

		statement, err := MakeStatement(path, &ss)
		if err != nil {
			log.Printf("Unable to create E-Olymp statement from specification in problem.xml: %v", err)
			os.Exit(-1)
		}

		problem.Statements = append(problem.Statements, statement)
	}

	out, err := client.CreateProblem(context.Background(), &judge.CreateProblemInput{
		ContestId: contest,
		Problem:   problem,
	})

	if err != nil {
		log.Printf("Unable to create E-Olymp problem: %v", err)
		os.Exit(-1)
	}

	log.Printf("Problem created with ID %#v", out.GetProblemId())
}

func MakeObject(client *Client, path string) (string, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	out, err := client.CreateObject(context.Background(), &atlas.CreateObjectInput{Data: data})
	if err != nil {
		return "", err
	}

	return out.Key, nil
}

func MakeVerifier(path string, spec *Specification) (*executor.Verifier, error) {
	switch spec.Checker.Name {
	case "std::rcmp4.cpp", // Single or more double, max any error 1E-4
		"std::ncmp.cpp": // Single or more int64, ignores whitespaces
		return &executor.Verifier{Type: executor.Verifier_TOKENS, Precision: 4, CaseSensitive: true}, nil
	case "std::rcmp6.cpp": // Single or more double, max any error 1E-6
		return &executor.Verifier{Type: executor.Verifier_TOKENS, Precision: 6, CaseSensitive: true}, nil
	case "std::rcmp9.cpp": // Single or more double, max any error 1E-9
		return &executor.Verifier{Type: executor.Verifier_TOKENS, Precision: 9, CaseSensitive: true}, nil
	case "std::wcmp.cpp": // Sequence of tokens
		return &executor.Verifier{Type: executor.Verifier_TOKENS, Precision: 5, CaseSensitive: true}, nil
	case "std::nyesno.cpp", // Zero or more yes/no, case insensetive
		"std::yesno.cpp": // Single yes or no, case insensetive
		return &executor.Verifier{Type: executor.Verifier_TOKENS, Precision: 5, CaseSensitive: false}, nil
	case "std::fcmp.cpp", // Lines, doesn't ignore whitespaces
		"std::hcmp.cpp", // Single huge integer
		"std::lcmp.cpp": // Lines, ignores whitespaces
		return &executor.Verifier{Type: executor.Verifier_LINES}, nil
	default:
		mapping := map[string][]string{
			"gpp":    {"c.gcc", "cpp.g++", "cpp.g++11", "cpp.g++14", "cpp.g++17", "cpp.ms"},
			"csharp": {"csharp.mono"},
			"d":      {"d"},
			"go":     {"go"},
			"java":   {"java11", "java8"},
			"kotlin": {"kotlin"},
			"fpc":    {"pas.dpr", "pas.fpc"},
			"php":    {"php.5"},
			"python": {"python.2", "python.3"},
			"pypy":   {"python.pypy2", "python.pypy3"},
			"ruby":   {"ruby"},
			"rust":   {"rust"},
		}

		for lang, types := range mapping {
			source, ok := SourceByType(spec.Checker.Sources, types...)
			if !ok {
				continue
			}

			log.Printf("Unknown checker name %#v, using source code", spec.Checker.Name)

			return &executor.Verifier{
				Type:   executor.Verifier_PROGRAM,
				Source: filepath.Join(path, source.Path), // todo: actually read file
				Lang:   lang,
			}, nil
		}
	}

	return nil, errors.New("checker configuration is not supported")
}

func MakeStatement(path string, statement *SpecificationStatement) (*judge.Problem_Statement, error) {
	locale, err := MakeStatementLocale(statement.Language)
	if err != nil {
		return nil, err
	}

	propdata, err := ioutil.ReadFile(filepath.Join(path, filepath.Dir(statement.Path), "problem-properties.json"))
	if err != nil {
		return nil, err
	}

	props := PolygonProblemProperties{}

	if err := json.Unmarshal(propdata, &props); err != nil {
		return nil, fmt.Errorf("unable to unmrashal problem-properties.json: %w", err)
	}

	parts := []string{props.Legend}
	if props.Input != "" {
		parts = append(parts, fmt.Sprintf("### Input\n\n%v", props.Input))
	}

	if props.Output != "" {
		parts = append(parts, fmt.Sprintf("### Output\n\n%v", props.Output))
	}

	if props.Notes != "" {
		parts = append(parts, fmt.Sprintf("### Notes\n\n%v", props.Notes))
	}

	return &judge.Problem_Statement{
		Locale:  locale,
		Title:   props.Name,
		Content: strings.Join(parts, "\n\n"),
		Format:  judge.Problem_Statement_TEX,
	}, nil
}

func MakeStatementLocale(lang string) (string, error) {
	switch lang {
	case "ukrainian", "russian", "english":
		return lang[:2], nil
	default:
		return lang, fmt.Errorf("unknown language %#v", lang)
	}
}
