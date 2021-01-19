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
	"github.com/eolymp/contracts/go/eolymp/keeper"
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

var client httpx.Client

func main() {
	pid := flag.String("id", "", "Problem ID")
	flag.Parse()

	client = httpx.NewClient(
		&http.Client{Timeout: 10 * time.Second},
		httpx.WithCredentials(oauth.PasswordCredentials(
			oauth.NewClient(env.StringDefault("EOLYMP_API_URL", "https://api.e-olymp.com")),
			env.String("EOLYMP_USERNAME"),
			env.String("EOLYMP_PASSWORD"),
		)),
	)

	atl := atlas.NewAtlas(client)

	path := flag.Arg(0)

	if path == "" {
		log.Println("Path argument is empty")
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

	ctx := context.Background()

	statements := map[string]*atlas.Statement{}
	testsets := map[uint32]*atlas.Testset{}
	tests := map[string]*atlas.Test{}

	// create problem
	if *pid == "" {
		pout, err := atl.CreateProblem(ctx, &atlas.CreateProblemInput{Problem: &atlas.Problem{}})
		if err != nil {
			log.Printf("Unable to create problem: %v", err)
			os.Exit(-1)
		}

		pid = &pout.ProblemId

		log.Printf("Problem created with ID %#v", *pid)
	} else {
		stout, err := atl.ListStatements(ctx, &atlas.ListStatementsInput{ProblemId: *pid})
		if err != nil {
			log.Printf("Unable to list problem statements in Atlas: %v", err)
			os.Exit(-1)
		}

		log.Printf("Found %v existing statements", len(stout.GetItems()))

		for _, s := range stout.GetItems() {
			statements[s.GetLocale()] = s
		}

		tsout, err := atl.ListTestsets(ctx, &atlas.ListTestsetsInput{ProblemId: *pid})
		if err != nil {
			log.Printf("Unable to list problem testsets in Atlas: %v", err)
			os.Exit(-1)
		}

		log.Printf("Found %v existing testsets", len(tsout.GetItems()))

		for _, ts := range tsout.GetItems() {
			testsets[ts.GetIndex()] = ts

			ttout, err := atl.ListTests(ctx, &atlas.ListTestsInput{TestsetId: ts.GetId()})
			if err != nil {
				log.Printf("Unable to list problem tests in Atlas: %v", err)
				os.Exit(-1)
			}

			log.Printf("Found %v existing tests in testset %v", len(ttout.GetItems()), ts.Index)

			for _, tt := range ttout.GetItems() {
				tests[fmt.Sprint(ts.Index, "/", tt.Index)] = tt
			}
		}
	}

	// set verifier
	verifier, err := MakeVerifier(path, spec)
	if err != nil {
		log.Printf("Unable to create E-Olymp verifier from specification in problem.xml: %v", err)
		os.Exit(-1)
	}

	if _, err = atl.UpdateVerifier(ctx, &atlas.UpdateVerifierInput{ProblemId: *pid, Verifier: verifier}); err != nil {
		log.Printf("Unable to update problem verifier: %v", err)
		os.Exit(-1)
	}

	log.Printf("Updated verifier")

	// create testsets
	for index, testset := range spec.Judging.Testsets {
		xt, ok := testsets[uint32(index+1)]
		if !ok {
			xt = &atlas.Testset{}
		}

		xt.Index = uint32(index + 1)
		xt.TimeLimit = uint32(testset.TimeLimit)
		xt.MemoryLimit = uint64(testset.MemoryLimit)
		xt.FileSizeLimit = 536870912
		xt.ScoringMode = atlas.Testset_EACH

		if xt.Id != "" {
			_, err = atl.UpdateTestset(ctx, &atlas.UpdateTestsetInput{TestsetId: xt.Id, Testset: xt})
			if err != nil {
				log.Printf("Unable to create testset: %v", err)
				os.Exit(-1)
			}

			log.Printf("Updated testset %v", xt.Id)
		} else {
			out, err := atl.CreateTestset(ctx, &atlas.CreateTestsetInput{ProblemId: *pid, Testset: xt})
			if err != nil {
				log.Printf("Unable to create testset: %v", err)
				os.Exit(-1)
			}

			xt.Id = out.Id

			log.Printf("Created testset %v", xt.Id)
		}

		// upload tests
		for ti, ts := range testset.Tests {
			xtt, ok := tests[fmt.Sprint(xt.Index, "/", int32(ti+1))]
			if !ok {
				xtt = &atlas.Test{}
			}

			log.Printf("Processing %v test %v in testset %v (example: %v)", ts.Method, ti, xt.Index, ts.Sample)

			input, err := MakeObject(filepath.Join(path, fmt.Sprintf(testset.InputPathPattern, ti+1)))
			if err != nil {
				log.Printf("Unable to upload test input data to E-Olymp: %v", err)
				os.Exit(-1)
			}

			answer, err := MakeObject(filepath.Join(path, fmt.Sprintf(testset.AnswerPathPattern, ti+1)))
			if err != nil {
				log.Printf("Unable to upload test answer data to E-Olymp: %v", err)
				os.Exit(-1)
			}

			xtt.Index = int32(ti + 1)
			xtt.Example = ts.Sample
			xtt.Score = int32(ts.Points)
			xtt.InputObjectId = input
			xtt.AnswerObjectId = answer

			if xtt.Id == "" {
				out, err := atl.CreateTest(ctx, &atlas.CreateTestInput{TestsetId: xt.Id, Test: xtt})
				if err != nil {
					log.Printf("Unable to create test: %v", err)
					os.Exit(-1)
				}

				xtt.Id = out.Id

				log.Printf("Created test %v", xtt.Id)
			} else {
				if _, err := atl.UpdateTest(ctx, &atlas.UpdateTestInput{TestId: xtt.Id, Test: xtt}); err != nil {
					log.Printf("Unable to update test: %v", err)
					os.Exit(-1)
				}

				log.Printf("Updated test %v", xtt.Id)
			}
		}
	}

	// get all statements
	for _, ss := range spec.Statements {
		if ss.Type != "application/x-tex" {
			continue
		}

		log.Printf("Processing statement in %#v", ss.Language)

		statement, err := MakeStatement(path, &ss)
		if err != nil {
			log.Printf("Unable to create E-Olymp statement from specification in problem.xml: %v", err)
			os.Exit(-1)
		}

		xs, ok := statements[statement.GetLocale()]
		if !ok {
			xs = statement
		} else {
			xs.Locale = statement.Locale
			xs.Title = statement.Title
			xs.Content = statement.Content
			xs.Format = statement.Format
			xs.Author = statement.Author
			xs.Source = statement.Source
		}

		if xs.Id == "" {
			out, err := atl.CreateStatement(ctx, &atlas.CreateStatementInput{ProblemId: *pid, Statement: xs})
			if err != nil {
				log.Printf("Unable to create statement: %v", err)
				os.Exit(-1)
			}

			xs.Id = out.Id

			log.Printf("Created statement %v", xs.Id)
		} else {
			_, err = atl.UpdateStatement(ctx, &atlas.UpdateStatementInput{StatementId: xs.Id, Statement: xs})
			if err != nil {
				log.Printf("Unable to create statement: %v", err)
				os.Exit(-1)
			}

			log.Printf("Updated statement %v", xs.Id)
		}
	}
}

func MakeObject(path string) (string, error) {
	kpr := keeper.NewKeeper(client)

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	out, err := kpr.CreateObject(context.Background(), &keeper.CreateObjectInput{Data: data})
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

			data, err := ioutil.ReadFile(filepath.Join(path, source.Path))
			if err != nil {
				return nil, err
			}

			return &executor.Verifier{
				Type:   executor.Verifier_PROGRAM,
				Source: string(data), // todo: actually read file
				Lang:   lang,
			}, nil
		}
	}

	return nil, errors.New("checker configuration is not supported")
}

func MakeStatement(path string, statement *SpecificationStatement) (*atlas.Statement, error) {
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
		parts = append(parts, fmt.Sprintf("\\InputFile\n\n%v", props.Input))
	}

	if props.Output != "" {
		parts = append(parts, fmt.Sprintf("\\OutputFile\n\n%v", props.Output))
	}

	if props.Notes != "" {
		parts = append(parts, fmt.Sprintf("\\Note\n\n%v", props.Notes))
	}

	if props.Scoring != "" {
		parts = append(parts, fmt.Sprintf("\\Scoring\n\n%v", props.Scoring))
	}

	return &atlas.Statement{
		Locale:  locale,
		Title:   props.Name,
		Content: strings.Join(parts, "\n\n"),
		Format:  atlas.Statement_TEX,
		Author:  props.AuthorName,
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
