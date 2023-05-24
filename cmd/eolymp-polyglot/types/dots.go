package types

import (
	"context"
	"encoding/json"
	"github.com/eolymp/go-sdk/eolymp/atlas"
	"github.com/eolymp/go-sdk/eolymp/executor"
	"github.com/eolymp/go-sdk/eolymp/keeper"
	"github.com/eolymp/go-sdk/eolymp/typewriter"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type DotsImporter struct {
	Importer
	path          string
	mainStatement string
	context       context.Context
	ts            *typewriter.TypewriterService
	kpr           *keeper.KeeperService
}

const DotsDefaultLang = "gpp"

func CreateDotsImporter(path string, context context.Context, ts *typewriter.TypewriterService, kpr *keeper.KeeperService) (*DotsImporter, error) {
	importer := new(DotsImporter)
	importer.path = path
	importer.context = context
	importer.ts = ts
	importer.kpr = kpr
	files, err := ioutil.ReadDir(filepath.Join(path, "files"))
	if err != nil {
		return nil, err
	}
	for _, statement := range files {
		if filepath.Ext(statement.Name()) == ".tex" {
			if strings.Contains(statement.Name(), "ua") && !strings.Contains(statement.Name(), "tutorial") {
				importer.mainStatement = statement.Name()
			}
		}
	}
	return importer, nil
}

func (imp DotsImporter) GetVerifier() (*executor.Verifier, error) {
	// todo
	return &executor.Verifier{Type: executor.Verifier_TOKENS, Precision: 0, CaseSensitive: true}, nil
}

func (imp DotsImporter) HasInteractor() bool {
	return false
}

func (imp DotsImporter) GetInteractor() (*executor.Interactor, error) {
	return nil, nil
}

func (imp DotsImporter) GetStatements(source string) ([]*atlas.Statement, error) {
	data, err := ioutil.ReadFile(filepath.Join(imp.path, "files", imp.mainStatement))
	if err != nil {
		return nil, err
	}
	d := string(data)
	name := strings.Split(d, "\n")[1][2:]
	statement := strings.Join(strings.Split(d, "\n")[2:], "\n")
	statement = statement[0:strings.Index(statement, "\\Example")]
	var statements []*atlas.Statement
	statements = append(statements, &atlas.Statement{
		Locale:     "uk",
		Title:      name,
		ContentRaw: statement,
		Format:     atlas.Statement_TEX,
		Author:     "",
		Source:     source,
	})
	return statements, nil
}

func (imp DotsImporter) GetSolutions() ([]*atlas.Editorial, error) {
	return nil, nil
}

func (imp DotsImporter) GetTestsets() ([]*Group, error) {
	var groups []*Group

	stf, err := ioutil.ReadFile(filepath.Join(imp.path, "files", imp.mainStatement))
	if err != nil {
		return nil, err
	}
	data := string(stf)
	split := strings.Split(data, "{")
	seconds, _ := strconv.ParseFloat(strings.Split(split[5], " ")[0], 32)
	time := uint32(seconds * 1000)
	megabytes, _ := strconv.ParseFloat(strings.Split(split[6], " ")[0], 32)
	memory := uint64(megabytes * 1024 * 1024)

	if jsonFile, err := os.Open(filepath.Join(imp.path, "files/problem.config")); err != nil {
		testset := &atlas.Testset{}
		testset.Index = 0
		testset.TimeLimit = time
		testset.MemoryLimit = memory
		testset.FileSizeLimit = 536870912
		testset.ScoringMode = atlas.ScoringMode_EACH
		testset.FeedbackPolicy = atlas.FeedbackPolicy_COMPLETE
		testset.Dependencies = nil

		testsFromLocation, err := GetExamplesFromLocation(filepath.Join(imp.path, "files"), imp.kpr)
		if err != nil {
			log.Println("Failed to get examples from tex location")
			return nil, err
		}

		testsFromTex, err := GetTestsFromTexStatement(data, imp.kpr)
		if err != nil {
			log.Println("Failed to get tests from tex statement")
			return nil, err
		}

		samples := new(Group)
		samples.Testset = testset
		samples.Name = 0

		samples.Tests = append(samples.Tests, testsFromLocation...)
		samples.Tests = append(samples.Tests, testsFromTex...)

		if len(samples.Tests) > 0 {
			groups = append(groups, samples)
		}

		newGroup := new(Group)

		tests, err := GetTestsFromLocation(imp.path, imp.kpr)
		if err != nil {
			return nil, err
		}

		for i, test := range tests {
			test.Example = false
			test.Score = float32(100 / len(tests))
			if len(tests)-i <= 100%len(tests) {
				test.Score++
			}
			newGroup.Tests = append(newGroup.Tests, test)
		}

		testset = &atlas.Testset{}
		testset.Index = 1
		testset.TimeLimit = time
		testset.MemoryLimit = memory
		testset.FileSizeLimit = 536870912
		testset.ScoringMode = atlas.ScoringMode_EACH
		testset.FeedbackPolicy = atlas.FeedbackPolicy_COMPLETE
		testset.Dependencies = nil

		newGroup.Testset = testset
		newGroup.Name = 1

		groups = append(groups, newGroup)
	} else {
		type ConfigGroup struct {
			Id          uint32
			FromTest    int32 `json:"from_test"`
			ToTest      int32 `json:"to_test"`
			Score       int
			AfterGroups []int `json:"after_groups"`
		}
		type ConfigSpec struct {
			Groups []ConfigGroup
		}
		byteValue, _ := ioutil.ReadAll(jsonFile)
		var config ConfigSpec
		json.Unmarshal(byteValue, &config)
		tests, err := GetTestsFromLocation(imp.path, imp.kpr)
		if err != nil {
			return nil, err
		}

		for _, g := range config.Groups {
			group := new(Group)
			testset := &atlas.Testset{}
			testset.Index = g.Id
			testset.CpuLimit = time
			testset.TimeLimit = time
			testset.MemoryLimit = memory
			testset.FileSizeLimit = 536870912
			if g.Id == 0 {
				testset.ScoringMode = atlas.ScoringMode_EACH
				testset.FeedbackPolicy = atlas.FeedbackPolicy_COMPLETE
			} else {
				testset.ScoringMode = atlas.ScoringMode_ALL
				testset.FeedbackPolicy = atlas.FeedbackPolicy_ICPC
			}

			for _, test := range tests {
				if g.FromTest <= test.Index && test.Index <= g.ToTest {
					group.Tests = append(group.Tests, test)
				}
			}

			group.Tests[0].Score = float32(g.Score)

			if g.Id == 0 {
				for i := 0; i < len(group.Tests); i++ {
					group.Tests[i].Example = true
				}
			}

			testset.Dependencies = nil

			for _, d := range g.AfterGroups {
				testset.Dependencies = append(testset.Dependencies, uint32(d))
			}

			group.Name = g.Id
			group.Testset = testset
			groups = append(groups, group)

		}

	}

	return groups, nil
}

func (imp DotsImporter) GetTemplates(pid *string) ([]*atlas.Template, error) {
	return nil, nil
}

func (imp DotsImporter) GetAttachments(*string) ([]*atlas.Attachment, error) {
	return nil, nil
}
