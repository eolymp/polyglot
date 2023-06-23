package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/eolymp/go-sdk/eolymp/atlas"
	"github.com/eolymp/go-sdk/eolymp/executor"
	"github.com/eolymp/polyglot/cmd/eolymp-polyglot/exporter"
	"github.com/eolymp/polyglot/cmd/eolymp-polyglot/types"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
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
	edi := atlas.NewEditorialServiceHttpClient("https://api.eolymp.com/spaces/"+conf.SpaceId+"/problems/"+pid, client)

	imp, err := types.CreateEolympImporter(ctx, pid, atl, edi)
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

	config.Interactor, err = downloadInteractor(imp, path)
	if err != nil {
		log.Println("Failed to download interactor")
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
	err = os.WriteFile(filepath.Join(path, "config.json"), jsonBytes, 0644)
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
		specStatement.Title = statement.Title
		specStatement.Locale = statement.GetLocale()
		if len(statement.Content.GetLatex()) > 0 {
			fileName := "statement-" + specStatement.Locale + ".tex"
			err = os.WriteFile(filepath.Join(path, fileName), []byte(statement.Content.GetLatex()), 0644)
			if err != nil {
				log.Println("Failed to save statement.tex file")
				return nil, err
			}
			specStatement.Source = fileName
		}
		if len(statement.GetDownloadLink()) > 0 {
			fileName := "statement-" + specStatement.Locale + ".pdf"
			err = downloadFile(filepath.Join(path, fileName), statement.GetDownloadLink())
			if err != nil {
				log.Println("Failed to download PDF statement")
				return nil, err
			}
			specStatement.PDF = fileName
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
		checkerFile := "checker.cpp"
		err = os.WriteFile(filepath.Join(path, checkerFile), []byte(verifier.Source), 0644)
		if err != nil {
			log.Println("Failed to save checker file")
			return specChecker, err
		}
		specChecker.Location = checkerFile
	}
	return specChecker, nil
}

func downloadInteractor(imp types.Importer, path string) (exporter.SpecificationInteractor, error) {
	var specInteractor exporter.SpecificationInteractor
	interactor, err := imp.GetInteractor()
	if err != nil {
		log.Println("Failed to download interactor")
		return specInteractor, err
	}
	log.Println(imp.GetInteractor())
	if interactor.Type == executor.Interactor_NONE {
		specInteractor.Location = ""
	} else {
		interactorFile := "interactor.cpp"
		err = os.WriteFile(filepath.Join(path, interactorFile), []byte(interactor.Source), 0644)
		if err != nil {
			log.Println("Failed to save checker file")
			return specInteractor, err
		}
		specInteractor.Location = interactorFile
	}
	return specInteractor, nil
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
			name := fmt.Sprint(group.Name) + "-" + fmt.Sprint(test.Index)
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

	return specGroups, nil
}

func saveDataToFile(path string, id string) error {
	return downloadFile(path, "https://blob.eolymp.com/objects/"+id)
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
