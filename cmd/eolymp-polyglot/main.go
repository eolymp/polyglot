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
		&http.Client{Timeout: 30 * time.Second},
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

	if len(spec.Judging.Testsets) > 1 {
		log.Printf("More than 1 testset defined in problem.xml, only first one will be imported")
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

	templateLanguages := map[string][]string{
		"files/template_cpp.cpp":		{"gpp"},
		"files/template_java.java":		{"java"},
		"files/template_pas.pas":		{"fpc"},
		"files/template_py.py":			{"pypy", "python"},
	}

	for _, file := range spec.Files {
		name := file.Source.Path
		if list, ok := templateLanguages[name]; ok {
			for _, lang := range list {
				template := &atlas.Template{}
				template.ProblemId = *pid
				template.Runtime = lang
				source, err := ioutil.ReadFile(filepath.Join(path, file.Source.Path))
				if err != nil {
					log.Printf("Unable to list problem tests in Atlas: %v", err)
					os.Exit(-1)
				}
				template.Source = string(source)
				atl.CreateCodeTemplate(ctx, &atlas.CreateCodeTemplateInput{ProblemId: *pid, Template: template})
				log.Printf("Added a template for %s", lang)
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

	// set interactor

	if len(spec.Interactor.Sources) != 0 {
		interactor, err := MakeInteractor(path, spec)
		if err != nil {
			log.Printf("Unable to create E-Olymp interactor from specification in problem.xml: %v", err)
			os.Exit(-1)
		}

		if _, err = atl.UpdateInteractor(ctx, &atlas.UpdateInteractorInput{ProblemId: *pid, Interactor: interactor}); err != nil {
			log.Printf("Unable to update problem interactor: %v", err)
			os.Exit(-1)
		}

		log.Printf("Updated interactor")
	} else {
		log.Printf("No interactor found")
	}

	// create testsets
	if len(spec.Judging.Testsets) > 0 {
		testset := spec.Judging.Testsets[0]

		// read tests by group
		groupTests := map[uint32][]SpecificationTest{}
		testIndex := map[string]int{}
		for gi, test := range testset.Tests {
			groupTests[test.Group] = append(groupTests[test.Group], test)
			testIndex[fmt.Sprint(test.Group, "/", len(groupTests[test.Group]))] = gi
		}

		groups := testset.Groups
		if len(groups) == 0 {
			groups = []SpecificationGroup{
				{FeedbackPolicy: "complete", Name: 0, Points: 0, PointsPolicy: "each-test"},
			}
		}

		for _, group := range groups {
			xts, ok := testsets[group.Name]
			if !ok {
				xts = &atlas.Testset{}
			}

			delete(testsets, group.Name)

			xts.Index = group.Name
			xts.TimeLimit = uint32(testset.TimeLimit)
			xts.MemoryLimit = uint64(testset.MemoryLimit)
			xts.FileSizeLimit = 536870912

			xts.ScoringMode = atlas.Testset_EACH
			if group.PointsPolicy == "complete-group" {
				xts.ScoringMode = atlas.Testset_ALL
			}

			xts.FeedbackPolicy = atlas.Testset_COMPLETE
			if group.FeedbackPolicy == "icpc" {
				xts.FeedbackPolicy = atlas.Testset_ICPC
			}

			xts.Dependencies = nil
			for _, d := range group.Dependencies {
				xts.Dependencies = append(xts.Dependencies, d.Group)
			}

			if xts.Id != "" {
				_, err = atl.UpdateTestset(ctx, &atlas.UpdateTestsetInput{TestsetId: xts.Id, Testset: xts})
				if err != nil {
					log.Printf("Unable to create testset: %v", err)
					os.Exit(-1)
				}

				log.Printf("Updated testset %v", xts.Id)
			} else {
				out, err := atl.CreateTestset(ctx, &atlas.CreateTestsetInput{ProblemId: *pid, Testset: xts})
				if err != nil {
					log.Printf("Unable to create testset: %v", err)
					os.Exit(-1)
				}

				xts.Id = out.Id

				log.Printf("Created testset %v", xts.Id)
			}

			// upload tests
			for ti, ts := range groupTests[group.Name] {
				xtt, ok := tests[fmt.Sprint(xts.Index, "/", int32(ti+1))]
				if !ok {
					xtt = &atlas.Test{}
				}

				delete(tests, fmt.Sprint(xts.Index, "/", int32(ti+1)))

				// index in the test list from specification
				gi := testIndex[fmt.Sprint(xts.Index, "/", int32(ti+1))]

				log.Printf("Processing %v test %v (Global Index: %v, ID: %#v) in testset %v (example: %v)", ts.Method, ti, gi, xtt.Id, xts.Index, ts.Sample)

				input, err := MakeObject(filepath.Join(path, fmt.Sprintf(testset.InputPathPattern, gi+1)))
				if err != nil {
					log.Printf("Unable to upload test input data to E-Olymp: %v", err)
					os.Exit(-1)
				}

				answer, err := MakeObject(filepath.Join(path, fmt.Sprintf(testset.AnswerPathPattern, gi+1)))
				if err != nil {
					log.Printf("Unable to upload test answer data to E-Olymp: %v", err)
					os.Exit(-1)
				}

				xtt.Index = int32(ti + 1)
				xtt.Example = ts.Sample
				xtt.Score = ts.Points
				xtt.InputObjectId = input
				xtt.AnswerObjectId = answer

				if xtt.Id == "" {
					out, err := atl.CreateTest(ctx, &atlas.CreateTestInput{TestsetId: xts.Id, Test: xtt})
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
	}

	// remove unused objects
	for _, test := range tests {
		log.Printf("Deleting unused test %v", test.Id)
		if _, err := atl.DeleteTest(ctx, &atlas.DeleteTestInput{TestId: test.Id}); err != nil {
			log.Printf("Unable to delete test: %v", err)
			os.Exit(-1)
		}
	}

	for _, testset := range testsets {
		log.Printf("Deleting unused testset %v", testset.Id)
		if _, err := atl.DeleteTestset(ctx, &atlas.DeleteTestsetInput{TestsetId: testset.Id}); err != nil {
			log.Printf("Unable to delete testset: %v", err)
			os.Exit(-1)
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

		delete(statements, statement.GetLocale())

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

	// remove unused objects
	for _, statement := range statements {
		log.Printf("Deleting unused statement %v", statement.Id)
		if _, err := atl.DeleteStatement(ctx, &atlas.DeleteStatementInput{StatementId: statement.Id}); err != nil {
			log.Printf("Unable to delete statement: %v", err)
			os.Exit(-1)
		}
	}
}

func MakeObject(path string) (key string, err error) {
	kpr := keeper.NewKeeper(client)

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	var out *keeper.CreateObjectOutput
	for i := 0; i < 10; i++ {
		out, err = kpr.CreateObject(context.Background(), &keeper.CreateObjectInput{Data: data})
		if err == nil {
			return out.Key, nil
		}

		log.Printf("Error while uploading file: %v", err)
	}

	return "", err
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

func MakeInteractor(path string, spec *Specification) (*executor.Interactor, error) {

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
		source, ok := SourceByType(spec.Interactor.Sources, types...)
		if !ok {
			continue
		}

		log.Printf("Unknown interactor name %#v, using source code", spec.Checker.Name)

		data, err := ioutil.ReadFile(filepath.Join(path, source.Path))
		if err != nil {
			return nil, err
		}

		return &executor.Interactor{
			Source: string(data), // todo: actually read file
			Lang:   lang,
		}, nil
	}

	return nil, errors.New("interactor configuration is not supported")
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

	if props.Interaction != "" {
		parts = append(parts, fmt.Sprintf("\\Interaction\n\n%v", props.Interaction))
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
