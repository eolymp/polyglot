package types

import (
	"context"
	"github.com/eolymp/go-sdk/eolymp/atlas"
	"github.com/eolymp/go-sdk/eolymp/executor"
	"github.com/eolymp/go-sdk/eolymp/keeper"
	"github.com/eolymp/go-sdk/eolymp/typewriter"
	"io/ioutil"
	"log"
	"path/filepath"
	"strconv"
	"strings"
)

type EjudgeImporter struct {
	Importer
	path          string
	mainStatement string
	context       context.Context
	ts            *typewriter.TypewriterService
	kpr           *keeper.KeeperService
}

const EjudgeDefaultLang = "gpp"

func CreateEjudgeImporter(path string, context context.Context, ts *typewriter.TypewriterService, kpr *keeper.KeeperService) (*EjudgeImporter, error) {
	importer := new(EjudgeImporter)
	importer.path = path
	importer.context = context
	importer.ts = ts
	importer.kpr = kpr
	files, err := ioutil.ReadDir(filepath.Join(path, "statement"))
	if err != nil {
		return nil, err
	}
	for _, statement := range files {
		if filepath.Ext(statement.Name()) == ".tex" {
			if strings.Contains(statement.Name(), "en") && !strings.Contains(statement.Name(), "tutorial") {
				importer.mainStatement = statement.Name()
			}
		}
	}
	return importer, nil
}

func (imp EjudgeImporter) GetVerifier() (*executor.Verifier, error) {
	names := [2]string{"check.cpp", "checker.cpp"}
	for _, name := range names {
		data, err := ioutil.ReadFile(filepath.Join(imp.path, name))
		if err == nil {
			return &executor.Verifier{
				Type:   executor.Verifier_PROGRAM,
				Source: string(data), // todo: actually read file
				Lang:   EjudgeDefaultLang,
			}, nil
		}
	}
	return &executor.Verifier{Type: executor.Verifier_TOKENS, Precision: 0, CaseSensitive: true}, nil
}

func (imp EjudgeImporter) HasInteractor() bool {
	names := [2]string{"inter.cpp", "interactor.cpp"}
	for _, name := range names {
		_, err := ioutil.ReadFile(filepath.Join(imp.path, name))
		if err == nil {
			return true
		}
	}
	return false
}

func (imp EjudgeImporter) GetInteractor() (*executor.Interactor, error) {
	names := [2]string{"inter.cpp", "interactor.cpp"}
	for _, name := range names {
		data, err := ioutil.ReadFile(filepath.Join(imp.path, name))
		if err == nil {
			return &executor.Interactor{
				Type:   executor.Interactor_PROGRAM,
				Source: string(data), // todo: actually read file
				Lang:   EjudgeDefaultLang,
			}, nil
		}
	}
	return nil, nil
}

func (imp EjudgeImporter) GetStatements(source string) ([]*atlas.Statement, error) {
	data, err := ioutil.ReadFile(filepath.Join(imp.path, "statement", imp.mainStatement))
	if err != nil {
		return nil, err
	}
	d := string(data)
	name := strings.Split(strings.Split(d, "{")[2], "}")[0]
	statement := d[strings.Index(d, "\n"):]
	statement = statement[0:strings.Index(statement, "\\Example")]
	var statements []*atlas.Statement
	statements = append(statements, &atlas.Statement{
		Locale:  "en",
		Title:   name,
		Content: statement,
		Format:  atlas.Statement_TEX,
		Author:  "",
		Source:  source,
	})
	return statements, nil
}

func (imp EjudgeImporter) GetSolutions() ([]*atlas.Solution, error) {
	return nil, nil
}

func (imp EjudgeImporter) GetTestsets() ([]*Group, error) {

	var groups []*Group

	stf, err := ioutil.ReadFile(filepath.Join(imp.path, "statement", imp.mainStatement))
	if err != nil {
		return nil, err
	}
	data := string(stf)
	split := strings.Split(data, "{")
	seconds, _ := strconv.Atoi(strings.Split(split[7], " ")[0])
	time := uint32(seconds * 1000)
	megabytes, _ := strconv.Atoi(strings.Split(split[8], " ")[0])
	memory := uint64(megabytes * 1024 * 1024)
	split = strings.Split(data, "\\exmp{")

	examples, err := GetTestsFromLocation(filepath.Join(imp.path, "statement"), imp.kpr)
	if err != nil {
		return nil, err
	}

	samples := new(Group)

	testset := &atlas.Testset{}
	testset.Index = 0
	testset.TimeLimit = time
	testset.MemoryLimit = memory
	testset.FileSizeLimit = 536870912
	testset.ScoringMode = atlas.ScoringMode_ALL
	testset.FeedbackPolicy = atlas.FeedbackPolicy_ICPC
	testset.Dependencies = nil

	samples.Testset = testset
	samples.Name = 0

	if len(examples) > 0 {
		for _, test := range examples {
			test.Example = true
			test.Score = 1
			samples.Tests = append(samples.Tests, test)
		}
		groups = append(groups, samples)
	} else if len(split) > 1 {

		for i, d := range split {
			if i == 0 {
				continue
			}
			tst := strings.Split(d, "}")
			inputData := RemoveSpaces(tst[0])
			outputData := RemoveSpaces(strings.Split(tst[1], "{")[1])
			input, err := MakeObjectByData([]byte(inputData), imp.kpr)
			if err != nil {
				log.Printf("Unable to upload test input data to E-Olymp: %v", err)
				return nil, err
			}
			output, err := MakeObjectByData([]byte(outputData), imp.kpr)
			if err != nil {
				log.Printf("Unable to upload test output data to E-Olymp: %v", err)
				return nil, err
			}
			log.Printf("Uploaded sample %d", i)
			test := &atlas.Test{}
			test.Index = int32(i)
			test.Example = true
			test.Score = 1
			test.InputObjectId = input.Key
			test.AnswerObjectId = output.Key
			samples.Tests = append(samples.Tests, test)
		}
		groups = append(groups, samples)

	}

	newGroup := new(Group)

	tests, err := GetTestsFromLocation(filepath.Join(imp.path, "tests"), imp.kpr)
	if err != nil {
		return nil, err
	}

	for _, test := range tests {
		test.Example = false
		test.Score = 1
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

	return groups, nil
}

func (imp EjudgeImporter) GetTemplates(pid *string) ([]*atlas.Template, error) {
	return nil, nil
}

func (imp EjudgeImporter) GetAttachments(*string) ([]*atlas.Attachment, error) {
	return nil, nil
}
