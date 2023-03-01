package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/eolymp/go-sdk/eolymp/executor"
	"github.com/eolymp/polyglot/cmd/eolymp-polyglot/exporter"
	"github.com/eolymp/polyglot/cmd/eolymp-polyglot/types"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

func Export(folder string, pid string) error {
	path := filepath.Join(folder, pid)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		err = os.RemoveAll(path)
		if err != nil {
			log.Println("Failed to delete folder")
			return err
		}
	}
	err := os.Mkdir(path, os.ModePerm)
	if err != nil {
		log.Println("Failed to create folder")
		return err
	}
	ctx := context.Background()
	imp, err := types.CreateEolympImporter(ctx, pid, atl)
	if err != nil {
		log.Println("Failed to create importer")
		return err
	}

	config := new(exporter.SpecificationConfig)

	config.Groups, err = downloadGroups(imp, path)
	if err != nil {
		log.Println("Failed to download groups")
		return err
	}

	config.Checker, err = downloadChecker(imp, path)
	if err != nil {
		log.Println("Failed to download checker")
		return err
	}

	config.Statements, err = downloadStatements(imp, path)
	if err != nil {
		log.Println("Failed to download statements")
		return err
	}

	jsonBytes, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		fmt.Printf("Error: %s", err.Error())
		return err
	}
	fmt.Println(string(jsonBytes))
	err = saveBytesToFile(filepath.Join(path, "config.json"), jsonBytes)
	if err != nil {
		log.Println("Failed to save config.json")
		return err
	}
	return nil
}

func downloadStatements(imp types.Importer, path string) ([]exporter.SpecificationStatement, error) {
	var specStatements []exporter.SpecificationStatement
	statements, err := imp.GetStatements("")
	if err != nil {
		log.Println("Failed to download statements")
		return nil, err
	}
	for _, statement := range statements {
		var specStatement exporter.SpecificationStatement
		specStatement.Format = statement.GetFormat().String()
		specStatement.Title = statement.Title
		specStatement.Locale = statement.GetLocale()
		if len(statement.GetContentLatex()) > 0 {
			err = saveStringToFile(filepath.Join(path, "statement.tex"), statement.GetContentLatex())
			if err != nil {
				log.Println("Failed to save statement.tex file")
				return nil, err
			}
			specStatement.Source = "statement.tex"
		}
		if len(statement.GetDownloadLink()) > 0 {
			err = downloadFile(filepath.Join(path, "statement.pdf"), statement.GetDownloadLink())
			if err != nil {
				log.Println("Failed to download PDF statement")
				return nil, err
			}
			specStatement.PDF = "statement.pdf"
		}
		specStatements = append(specStatements, specStatement)
	}
	return specStatements, nil
}

func downloadChecker(imp types.Importer, path string) (exporter.SpecificationChecker, error) {
	var specChecker exporter.SpecificationChecker
	verifier, err := imp.GetVerifier()
	if err != nil {
		log.Println("Failed to download verifier")
		return specChecker, err
	}
	log.Println(imp.GetVerifier())
	specChecker.Type = verifier.Type.String()
	if verifier.Type == executor.Verifier_TOKENS {
		specChecker.CaseSensitive = verifier.CaseSensitive
		specChecker.Precision = verifier.Precision
	} else {
		err = saveStringToFile(filepath.Join(path, "checker.cpp"), verifier.Source)
		if err != nil {
			log.Println("Failed to save checker.cpp")
			return specChecker, err
		}
		specChecker.Location = "checker.cpp"
	}
	return specChecker, nil
}

func downloadGroups(imp types.Importer, path string) ([]exporter.SpecificationGroup, error) {
	var specGroups []exporter.SpecificationGroup

	groups, err := imp.GetTestsets()
	if err != nil {
		log.Println("Failed to get tests")
		return nil, err
	}
	for _, group := range groups {
		g := new(exporter.SpecificationGroup)
		g.Index = group.Testset.Index
		g.TimeLimit = group.Testset.TimeLimit
		g.MemoryLimit = group.Testset.MemoryLimit
		g.ScoringMode = group.Testset.ScoringMode.String()
		g.FeedBackPolicy = group.Testset.FeedbackPolicy.String()
		g.Dependencies = group.Testset.Dependencies
		if g.Dependencies == nil {
			g.Dependencies = []uint32{}
		}
		for _, test := range group.Tests {
			g.Scores = append(g.Scores, test.Score)
		}
		specGroups = append(specGroups, *g)
	}
	if false {
		testDir := filepath.Join(path, "tests")
		if _, err := os.Stat(testDir); os.IsNotExist(err) {
			err = os.Mkdir(testDir, os.ModePerm)
			if err != nil {
				log.Println("Failed to create tests folder")
				return nil, err
			}
		}
		for _, group := range groups {
			log.Println(group)
			for _, test := range group.Tests {
				log.Println(test)
				name := strconv.Itoa(int(group.Name)) + "-" + strconv.Itoa(int(test.Index))
				err = saveDataToFile(filepath.Join(testDir, name+".in"), test.InputObjectId)
				if err != nil {
					log.Println("Failed to download input")
					return nil, err
				}
				err = saveDataToFile(filepath.Join(testDir, name+".out"), test.AnswerObjectId)
				if err != nil {
					log.Println("Failed to download output")
					return nil, err
				}
			}
		}
	}
	return specGroups, nil
}

func saveDataToFile(path string, id string) error {
	result, err := http.Get("https://blob.eolymp.com/objects/" + id)
	if err != nil {
		log.Println("Failed to download test")
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		log.Println("Failed to create test file")
		return err
	}
	defer file.Close()
	_, err = io.Copy(file, result.Body)
	if err != nil {
		log.Println("Failed to write data to file")
		return err
	}
	return nil
}

func saveStringToFile(path string, data string) error {
	file, err := os.Create(path)
	if err != nil {
		log.Println("Failed to create file")
		return err
	}
	defer file.Close()
	_, err = file.WriteString(data)
	if err != nil {
		log.Println("Failed to write data to file")
		return err
	}
	return nil
}

func saveBytesToFile(path string, data []byte) error {
	file, err := os.Create(path)
	if err != nil {
		log.Println("Failed to create file")
		return err
	}
	defer file.Close()
	_, err = file.Write(data)
	if err != nil {
		log.Println("Failed to write data to file")
		return err
	}
	return nil
}

func downloadFile(filepath string, url string) error {

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}