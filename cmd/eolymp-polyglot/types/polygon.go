package types

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/eolymp/go-sdk/eolymp/atlas"
	"github.com/eolymp/go-sdk/eolymp/executor"
	"github.com/eolymp/go-sdk/eolymp/keeper"
	"github.com/eolymp/go-sdk/eolymp/typewriter"
	"golang.org/x/exp/slices"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type PolygonImporter struct {
	Importer
	spec *Specification
	path string
}

func CreatePolygonImporter(path string) (*PolygonImporter, error) {
	p := new(PolygonImporter)
	p.path = path

	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Printf("Import path %#v is invalid: %v", path, err)
		return nil, err
	}

	p.spec = &Specification{}

	specf, err := os.Open(filepath.Join(path, "problem.xml"))
	if err != nil {
		log.Printf("Unable to open problem.xml: %v", err)
		return nil, err
	}

	defer func() {
		_ = specf.Close()
	}()

	if err := xml.NewDecoder(specf).Decode(&p.spec); err != nil {
		log.Printf("Unable to parse problem.xml: %v", err)
		return nil, err
	}

	if len(p.spec.Judging.Testsets) > 1 {
		log.Printf("More than 1 testset defined in problem.xml, only first one will be imported")
	}

	return p, nil
}

var mapping = map[string][]string{
	"cpp:17-gnu10": {"c.gcc", "cpp.g++", "cpp.g++11", "cpp.g++14", "cpp.g++17", "cpp.ms", "cpp.msys2-mingw64-9-g++17"},
	"csharp":       {"csharp.mono"},
	"d":            {"d"},
	"go":           {"go"},
	"java":         {"java11", "java8"},
	"kotlin":       {"kotlin"},
	"fpc":          {"pas.dpr", "pas.fpc"},
	"php":          {"php.5"},
	"python":       {"python.2", "python.3"},
	"pypy":         {"python.pypy2", "python.pypy3"},
	"ruby":         {"ruby"},
	"rust":         {"rust"},
}

func (p PolygonImporter) GetVerifier() (*executor.Verifier, error) {
	switch p.spec.Checker.Name {
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

		for lang, types := range mapping {
			source, ok := SourceByType(p.spec.Checker.Sources, types...)
			if !ok {
				continue
			}

			log.Printf("Unknown checker name %#v, using source code", p.spec.Checker.Name)

			data, err := ioutil.ReadFile(filepath.Join(p.path, source.Path))
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

func (p PolygonImporter) HasInteractor() bool {
	return len(p.spec.Interactor.Sources) > 0
}

func (p PolygonImporter) GetInteractor() (*executor.Interactor, error) {

	for lang, types := range mapping {
		source, ok := SourceByType(p.spec.Interactor.Sources, types...)
		if !ok {
			continue
		}

		log.Printf("Unknown interactor name %#v, using source code", p.spec.Checker.Name)

		data, err := ioutil.ReadFile(filepath.Join(p.path, source.Path))
		if err != nil {
			return nil, err
		}

		return &executor.Interactor{
			Type:   executor.Interactor_PROGRAM,
			Source: string(data), // todo: actually read file
			Lang:   lang,
		}, nil
	}

	return nil, errors.New("interactor configuration is not supported")
}

func (p PolygonImporter) GetStatements(ctx context.Context, tw *typewriter.TypewriterService, source string) ([]*atlas.Statement, error) {
	var statements []*atlas.Statement
	for _, statement := range p.spec.Statements {
		if statement.Type != "application/x-tex" {
			continue
		}
		locale, err := MakeLocale(statement.Language)
		if err != nil {
			continue
		}

		propdata, err := ioutil.ReadFile(filepath.Join(p.path, filepath.Dir(statement.Path), "problem-properties.json"))
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

		content := strings.Join(parts, "\n\n")

		content, err = UpdateContentWithPictures(ctx, tw, content, p.path+"/statements/"+statement.Language+"/")
		if err != nil {
			return nil, err
		}

		statements = append(statements, &atlas.Statement{
			Locale:  locale,
			Title:   props.Name,
			Content: content,
			Format:  atlas.Statement_TEX,
			Author:  props.AuthorName,
			Source:  source,
		})
	}
	return statements, nil
}

func (p PolygonImporter) GetSolutions() ([]*atlas.Solution, error) {
	var solutions []*atlas.Solution
	for _, solution := range p.spec.Solutions {
		if solution.Type != "application/x-tex" {
			continue
		}
		locale, err := MakeLocale(solution.Language)
		if err != nil {
			return nil, err
		}

		propdata, err := ioutil.ReadFile(filepath.Join(p.path, filepath.Dir(solution.Path), "problem-properties.json"))
		if err != nil {
			return nil, err
		}

		props := PolygonProblemProperties{}

		if err := json.Unmarshal(propdata, &props); err != nil {
			return nil, fmt.Errorf("unable to unmrashal problem-properties.json: %w", err)
		}

		parts := []string{props.Solution}
		if props.Input != "" {
			parts = append(parts, fmt.Sprintf("\\InputFile\n\n%v", props.Input))
		}
		solutions = append(solutions, &atlas.Solution{
			Locale:  locale,
			Content: props.Solution,
			Format:  atlas.Solution_TEX,
		})
	}

	return solutions, nil
}

func (p PolygonImporter) GetTestsets(kpr *keeper.KeeperService) ([]*Group, error) {

	tags := p.getTags()

	blockMin := slices.Contains(tags, "block_min")

	var groups []*Group

	if len(p.spec.Judging.Testsets) > 0 {
		testset := p.spec.Judging.Testsets[0]
		for _, test := range p.spec.Judging.Testsets {
			if test.Name == "tests" {
				testset = test
			}
		}

		// read tests by group
		groupTests := map[uint32][]SpecificationTest{}
		testIndex := map[string]int{}
		for gi, test := range testset.Tests {
			groups := strings.Split(test.Group, "-")
			for _, group := range groups {
				intName, err := strconv.ParseUint(group, 10, 32)
				if err != nil {
					return nil, err
				}
				groupIndex := uint32(intName)
				groupTests[groupIndex] = append(groupTests[groupIndex], test)
				testIndex[fmt.Sprint(groupIndex, "/", len(groupTests[groupIndex]))] = gi
			}
		}

		groupList := testset.Groups
		if len(groupList) == 0 {
			groupList = []SpecificationGroup{
				{FeedbackPolicy: "icpc-expanded", Name: "0", Points: 100, PointsPolicy: "each-test"},
			}
		}

		for intName, groupTest := range groupTests {

			group := groupList[0]
			found := false
			for _, g := range groupList {
				if g.Name == string(intName) {
					group = g
					found = true
					break
				}
			}

			if !found {
				group = SpecificationGroup{
					FeedbackPolicy: groupList[0].FeedbackPolicy,
					Name:           string(intName),
					Points:         0,
					PointsPolicy:   groupList[0].PointsPolicy,
					Dependencies:   nil,
				}
			}

			newGroup := new(Group)
			log.Println(group.Name)

			groupIndex := uint32(intName)

			newGroup.Name = groupIndex
			xts := &atlas.Testset{}

			xts.Index = newGroup.Name
			xts.TimeLimit = uint32(testset.TimeLimit)
			xts.MemoryLimit = uint64(testset.MemoryLimit)
			xts.FileSizeLimit = 536870912

			xts.ScoringMode = atlas.ScoringMode_EACH
			if group.PointsPolicy == "complete-group" {
				xts.ScoringMode = atlas.ScoringMode_ALL
			}

			if blockMin && group.Name != "0" {
				xts.ScoringMode = atlas.ScoringMode_WORST
			}

			xts.FeedbackPolicy = atlas.FeedbackPolicy_COMPLETE
			if group.FeedbackPolicy == "icpc" || group.FeedbackPolicy == "points" {
				xts.FeedbackPolicy = atlas.FeedbackPolicy_ICPC
			} else if group.FeedbackPolicy == "icpc-expanded" {
				xts.FeedbackPolicy = atlas.FeedbackPolicy_ICPC_EXPANDED
			}

			xts.Dependencies = nil
			for _, d := range group.Dependencies {
				intName, err := strconv.ParseUint(d.Group, 10, 32)
				if err != nil {
					continue
				}
				groupIndex := uint32(intName)
				xts.Dependencies = append(xts.Dependencies, groupIndex)
			}

			newGroup.Testset = xts

			// upload tests

			for ti, ts := range groupTest {
				xtt := &atlas.Test{}

				// index in the test list from specification
				gi := testIndex[fmt.Sprint(xts.Index, "/", int32(ti+1))]

				log.Printf("Processing %v test %v (Global Index: %v, ID: %#v) in testset %v (example: %v)", ts.Method, ti, gi, xtt.Id, xts.Index, ts.Sample)

				input, err := MakeObject(filepath.Join(p.path, fmt.Sprintf(testset.InputPathPattern, gi+1)), kpr)
				if err != nil {
					log.Printf("Unable to upload test input data to E-Olymp: %v", err)
					return nil, err
				}

				answer, err := MakeObject(filepath.Join(p.path, fmt.Sprintf(testset.AnswerPathPattern, gi+1)), kpr)
				if err != nil {
					log.Printf("Unable to upload test answer data to E-Olymp: %v", err)
					return nil, err
				}

				xtt.Index = int32(ti + 1)
				xtt.Example = intName == 0
				xtt.Score = ts.Points
				xtt.InputObjectId = input
				xtt.AnswerObjectId = answer

				if xts.FeedbackPolicy == atlas.FeedbackPolicy_ICPC_EXPANDED {
					score := 100 / len(groupTests[groupIndex])
					if len(groupTests[groupIndex])-ti <= 100%len(groupTests[groupIndex]) {
						score++
					}
					xtt.Score = float32(score)
				}

				newGroup.Tests = append(newGroup.Tests, xtt)
			}
			groups = append(groups, newGroup)

		}

	}
	return groups, nil
}

func (p PolygonImporter) GetTemplates(pid *string, kpr *keeper.KeeperService) ([]*atlas.Template, error) {
	templateLanguages := map[string][]string{
		"files/template_cpp.cpp":   {"gpp", "cpp:17-gnu10"},
		"files/template_java.java": {"java"},
		"files/template_pas.pas":   {"fpc"},
		"files/template_py.py":     {"pypy", "python"},
	}

	var templates []*atlas.Template
	for _, file := range p.spec.Templates {
		name := file.Source.Path
		if list, ok := templateLanguages[name]; ok {
			for _, lang := range list {
				template := &atlas.Template{}
				template.ProblemId = *pid
				template.Runtime = lang
				source, err := ioutil.ReadFile(filepath.Join(p.path, file.Source.Path))
				if err != nil {
					return nil, err
				}
				template.Source = string(source)
				templates = append(templates, template)
			}
		}
	}

	if len(p.spec.Graders) > 0 {
		template := &atlas.Template{}
		template.ProblemId = *pid
		template.Runtime = "cpp:17-gnu10"
		var graders []*keeper.CreateObjectOutput
		for _, file := range p.spec.Graders {
			path := filepath.Join(p.path, file.Path)
			hasSolution := false
			for _, asset := range file.Assets {
				if asset.Name == "solution" {
					hasSolution = true
				}
			}
			if hasSolution {
				obj, err := MakeObjectGetFile(path, kpr)
				if err != nil {
					fmt.Println("Failed to upload grader")
					return nil, err
				}
				graders = append(graders, obj)
				splits := strings.Split(path, "/")
				fileName := splits[len(splits)-1]
				f := atlas.File{
					Path:      fileName,
					SourceErn: obj.BlobErn,
				}
				log.Println(fileName, "has been uploaded")
				template.Files = append(template.Files, &f)
				templates = append(templates, template)
			}
		}

	}

	return templates, nil
}

func (p PolygonImporter) getTags() []string {
	var tags []string
	for _, tag := range p.spec.Tags {
		tags = append(tags, tag.Value)
	}
	return tags
}

func (p PolygonImporter) GetAttachments(pid *string, ctx context.Context, tw *typewriter.TypewriterService) ([]*atlas.Attachment, error) {
	var attachments []*atlas.Attachment

	for _, material := range p.spec.Materials {
		if material.Publish == "with-statement" {
			data, err := ioutil.ReadFile(filepath.Join(p.path, material.Path))
			if err != nil {
				fmt.Println("Failed to upload material")
				return nil, err
			}

			splits := strings.Split(material.Path, "/")
			fileName := splits[len(splits)-1]

			asset, err := tw.UploadAsset(ctx, &typewriter.UploadAssetInput{Filename: fileName, Data: data})
			if err != nil {
				log.Println(err)
				return nil, err
			}

			attachment := atlas.Attachment{
				ProblemId: *pid,
				Name:      fileName,
				Link:      asset.Link,
			}
			log.Println(fileName, "has been uploaded")
			attachments = append(attachments, &attachment)
		}
	}
	return attachments, nil
}
