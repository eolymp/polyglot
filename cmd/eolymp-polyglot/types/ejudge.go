package types

import (
	"context"
	"errors"
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
	config        map[string]string
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
		importer.mainStatement = ""
	} else {
		for _, statement := range files {
			if filepath.Ext(statement.Name()) == ".tex" {
				if strings.Contains(statement.Name(), "en") && !strings.Contains(statement.Name(), "tutorial") {
					importer.mainStatement = statement.Name()
				}
			}
		}
	}
	importer.config, err = CreateConfig(filepath.Join(path, "../../conf/serve.cfg"), path[strings.LastIndex(path, "/")+1:])
	if err != nil {
		log.Println("Failed to get config")
		return nil, err
	}
	return importer, nil
}

func CreateConfig(path string, letter string) (map[string]string, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		log.Println("Can't read file")
		return nil, err
	}
	data := string(bytes)
	list := strings.Split(data, "\n\n")
	var problems []map[string]string
	for _, obj := range list {
		rows := strings.Split(obj, "\n")
		if rows[0] == "[problem]" {
			m := map[string]string{}
			for _, row := range rows {
				s := strings.Split(row, " = ")
				if len(s) == 1 {
					m[s[0]] = ""
				} else {
					if s[1][0] == '"' && s[1][len(s[1])-1] == '"' {
						m[s[0]] = s[1][1 : len(s[1])-1]
					} else {
						m[s[0]] = s[1]
					}
				}
			}
			problems = append(problems, m)
		}
	}
	var problem map[string]string
	for _, prob := range problems {
		if prob["short_name"] == letter {
			problem = prob
		}
	}
	super, hasSuper := problem["super"]
	if hasSuper {
		for _, prob := range problems {
			if prob["short_name"] == super {
				for k, v := range prob {
					_, ok := problem[k]
					if !ok {
						problem[k] = v
					}
				}
			}
		}
	}
	return problem, nil
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
	var statement, name string
	data, err := ioutil.ReadFile(filepath.Join(imp.path, "statement", imp.mainStatement))
	if err == nil {
		d := string(data)
		name = strings.Split(strings.Split(d, "{")[2], "}")[0]
		statement = d[strings.Index(d, "\n"):]
		statement = statement[0:strings.Index(statement, "\\Example")]
	} else {
		statement = ""
		name = imp.config["long_name"]
	}

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

func (imp EjudgeImporter) GetSolutions() ([]*atlas.Editorial, error) {
	return nil, nil
}

func (imp EjudgeImporter) GetTestsets() ([]*Group, error) {

	var groups []*Group

	var time uint32
	var memory uint64

	samples := new(Group)

	testset := &atlas.Testset{}
	testset.Index = 0
	testset.FileSizeLimit = 536870912
	testset.ScoringMode = atlas.ScoringMode_ALL
	testset.FeedbackPolicy = atlas.FeedbackPolicy_COMPLETE
	testset.Dependencies = nil

	samples.Testset = testset
	samples.Name = 0

	valuerCmd, hasValuer := imp.config["valuer_cmd"]

	stf, err := ioutil.ReadFile(filepath.Join(imp.path, "statement", imp.mainStatement))
	if err != nil {
		tl, _ := strconv.Atoi(imp.config["time_limit_millis"])
		if tl < 1000 {
			tl = 1000
		}
		time = uint32(tl)
		memory = 536870912 // TODO
		testset.TimeLimit = time
		testset.MemoryLimit = memory
		scores, ok := imp.config["test_score_list"]
		if ok && !hasValuer {
			sampleTests := 0
			for _, item := range strings.Split(scores, " ") {
				if item == "0" {
					sampleTests++
				} else {
					break
				}
			}
			tests, err := GetTestsFromLocation(filepath.Join(imp.path, "tests"), imp.kpr)
			if err != nil {
				return nil, err
			}

			for ind, test := range tests {
				if ind == sampleTests {
					break
				}
				test.Example = true
				test.Score = 0
				samples.Tests = append(samples.Tests, test)
			}
		}
	} else {
		data := string(stf)
		split := strings.Split(data, "{")
		seconds, _ := strconv.Atoi(strings.Split(split[7], " ")[0])
		time = uint32(seconds * 1000)
		megabytes, _ := strconv.Atoi(strings.Split(split[8], " ")[0])
		memory = uint64(megabytes * 1024 * 1024)
		testset.TimeLimit = time
		testset.MemoryLimit = memory
		split = strings.Split(data, "\\exmp{")
		examples, err := GetTestsFromLocation(filepath.Join(imp.path, "statement"), imp.kpr)
		if err != nil {
			return nil, err
		}
		if len(examples) > 0 {
			for _, test := range examples {
				test.Example = true
				test.Score = 1
				samples.Tests = append(samples.Tests, test)
			}
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
				test.InputObjectId = input
				test.AnswerObjectId = output
				samples.Tests = append(samples.Tests, test)
			}

		}
	}
	groups = append(groups, samples)

	tests, err := GetTestsFromLocation(filepath.Join(imp.path, "tests"), imp.kpr)
	if err != nil {
		return nil, err
	}

	if hasValuer && valuerCmd == "gvaluer" {
		valuer, err := ReadGvaluerConfig(filepath.Join(imp.path, "valuer.cfg"))
		if err != nil {
			return nil, err
		}
		for name, g := range valuer {
			index, err := strconv.Atoi(name)
			if err != nil {
				return nil, err
			}
			newGroup := new(Group)
			if index == 0 {
				newGroup = samples
			} else {
				newGroup.Name = uint32(index)

				testset = &atlas.Testset{}
				testset.Index = uint32(index)
				testset.TimeLimit = time
				testset.MemoryLimit = memory
				testset.FileSizeLimit = 536870912
				testset.ScoringMode = atlas.ScoringMode_ALL
				testset.FeedbackPolicy = atlas.FeedbackPolicy_ICPC
				testset.Dependencies = nil

				newGroup.Testset = testset
				newGroup.Name = uint32(index)
			}

			testNumber, hasTests := g["tests"]
			if !hasTests {
				return nil, errors.New("gvaluer config does not have tests")
			}
			leftIndex, rightIndex := 0, 0
			dashIndex := strings.Index(testNumber, "-")
			if dashIndex == -1 {
				number, err := strconv.Atoi(testNumber)
				if err != nil {
					return nil, err
				}
				leftIndex, rightIndex = number, number
			} else {
				leftIndex, err = strconv.Atoi(testNumber[:dashIndex])
				if err != nil {
					return nil, err
				}
				rightIndex, err = strconv.Atoi(testNumber[dashIndex+1:])
				if err != nil {
					return nil, err
				}
			}
			for _, test := range tests {
				if leftIndex <= int(test.Index) && int(test.Index) <= rightIndex {
					newGroup.Tests = append(newGroup.Tests, test)
				}
			}
			if index == 0 {
				for i := 0; i < len(newGroup.Tests); i++ {
					newGroup.Tests[i].Example = true
				}
			}
			score, hasScore := g["score"]
			if hasScore {
				score, err := strconv.Atoi(score)
				if err != nil {
					return nil, err
				}
				newGroup.Tests[0].Score = float32(score)
			}
			dependencies, hasDependencies := g["requires"]
			if hasDependencies {
				for _, item := range strings.Split(dependencies, ",") {
					number, err := strconv.Atoi(strings.ReplaceAll(item, " ", ""))
					if err != nil {
						return nil, err
					}
					testset.Dependencies = append(testset.Dependencies, uint32(number))
				}
			}
			if index != 0 {
				groups = append(groups, newGroup)
			} else {
				groups[0] = newGroup
			}
		}
	} else {

		newGroup := new(Group)

		score_list, scoresSet := imp.config["test_score_list"]
		scores := strings.Split(score_list, " ")

		for ind, test := range tests {
			test.Example = false
			test.Score = 1
			if scoresSet {
				score, err := strconv.Atoi(scores[ind])
				if err != nil {
					return nil, err
				}
				test.Score = float32(score)
				if test.Score == 0 {
					continue
				}
			}
			newGroup.Tests = append(newGroup.Tests, test)
		}
		if !scoresSet {
			AddPointsToTests(newGroup)
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

	}
	log.Println(groups)
	return groups, nil
}

func ReadGvaluerConfig(path string) (map[string]map[string]string, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		log.Println("Can't read file")
		return nil, err
	}
	data := string(bytes)
	m := map[string]map[string]string{}
	splits := strings.Split(data, "}")
	for i := 0; i+1 < len(splits); i++ {
		data := splits[i]
		openIndex := strings.Index(data, "{")
		if openIndex == -1 {
			return nil, errors.New("can't parse gvaluer config (open index)")
		}
		groupIndex := strings.Index(data[:openIndex], "group")
		if groupIndex == -1 {
			return nil, errors.New("can't parse gvaluer config (group index)")
		}
		name := strings.ReplaceAll(data[groupIndex+5:openIndex], " ", "")
		log.Println(name)
		group := map[string]string{}
		items := strings.Split(data[openIndex+1:], ";")
		for j := 0; j+1 < len(items); j++ {
			item := strings.TrimSpace(items[j])
			spaceIndex := strings.Index(item, " ")
			if spaceIndex == -1 {
				return nil, errors.New("can't parse gvaluer config (spaceIndex index)")
			}
			key := item[:spaceIndex]
			value := item[spaceIndex+1:]
			group[key] = value
		}
		m[name] = group
	}
	return m, nil
}

func (imp EjudgeImporter) GetTemplates(pid *string) ([]*atlas.Template, error) {
	return nil, nil
}

func (imp EjudgeImporter) GetAttachments(*string) ([]*atlas.Attachment, error) {
	return nil, nil
}
