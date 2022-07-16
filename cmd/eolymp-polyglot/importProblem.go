package main

import (
	"context"
	"fmt"
	"github.com/eolymp/contracts/go/eolymp/atlas"
	"github.com/eolymp/contracts/go/eolymp/wellknown"
	"github.com/eolymp/polyglot/cmd/eolymp-polyglot/types"
	"log"
)

func ImportProblem(path string, pid *string) error {

	var err error

	imp, err := types.CreatePolygonImporter(path)

	ctx := context.Background()

	statements := map[string]*atlas.Statement{}
	solutions := map[string]*atlas.Solution{}
	testsets := map[uint32]*atlas.Testset{}
	tests := map[string]*atlas.Test{}

	// create problem
	if *pid == "" {
		*pid, err = CreateProblem(ctx)
		if err != nil {
			log.Printf("Unable to create problem: %v", err)
			return err
		}
	} else {
		stout, err := atl.ListStatements(ctx, &atlas.ListStatementsInput{ProblemId: *pid})
		if err != nil {
			log.Printf("Unable to list problem statements in Atlas: %v", err)
			return err
		}

		log.Printf("Found %v existing statements", len(stout.GetItems()))

		for _, s := range stout.GetItems() {
			statements[s.GetLocale()] = s
		}

		eq := wellknown.ExpressionID{
			Is:    wellknown.ExpressionID_EQUAL,
			Value: *pid,
		}
		var filters []*wellknown.ExpressionID
		filters = append(filters, &eq)
		input := &atlas.ListSolutionsInput{Filters: &atlas.ListSolutionsInput_Filter{ProblemId: filters}}
		solout, err := atl.ListSolutions(ctx, input)
		if err != nil {
			log.Printf("Unable to list problem solutions in Atlas: %v", err)
			return err
		}

		log.Printf("Found %v existing solutions", len(solout.GetItems()))

		for _, s := range solout.GetItems() {
			solutions[s.GetLocale()] = s
		}

		tsout, err := atl.ListTestsets(ctx, &atlas.ListTestsetsInput{ProblemId: *pid})
		if err != nil {
			log.Printf("Unable to list problem testsets in Atlas: %v", err)
			return err
		}

		log.Printf("Found %v existing testsets", len(tsout.GetItems()))

		for _, ts := range tsout.GetItems() {
			testsets[ts.GetIndex()] = ts

			ttout, err := atl.ListTests(ctx, &atlas.ListTestsInput{TestsetId: ts.GetId()})
			if err != nil {
				log.Printf("Unable to list problem tests in Atlas: %v", err)
				return err
			}

			log.Printf("Found %v existing tests in testset %v", len(ttout.GetItems()), ts.Index)

			for _, tt := range ttout.GetItems() {
				tests[fmt.Sprint(ts.Index, "/", tt.Index)] = tt
			}
		}
	}

	oldTemplates, err := atl.ListCodeTemplates(ctx, &atlas.ListCodeTemplatesInput{ProblemId: *pid})

	for _, template := range oldTemplates.GetItems() {
		_, _ = atl.DeleteCodeTemplate(ctx, &atlas.DeleteCodeTemplateInput{TemplateId: template.Id})
	}

	templates, err := imp.GetTemplates(pid)

	for _, template := range templates {
		_, _ = atl.CreateCodeTemplate(ctx, &atlas.CreateCodeTemplateInput{ProblemId: *pid, Template: template})
		log.Printf("Added a template for %s", template.Runtime)
	}

	// set verifier
	verifier, err := imp.GetVerifier()
	if err != nil {
		log.Printf("Unable to create E-Olymp verifier: %v", err)
		return err
	}

	if _, err = atl.UpdateVerifier(ctx, &atlas.UpdateVerifierInput{ProblemId: *pid, Verifier: verifier}); err != nil {
		log.Printf("Unable to update problem verifier: %v", err)
		return err
	}

	log.Printf("Updated verifier")

	// set interactor

	if imp.HasInteractor() {
		interactor, err := imp.GetInteractor()
		if err != nil {
			log.Printf("Unable to create E-Olymp interactor: %v", err)
			return err
		}

		if _, err = atl.UpdateInteractor(ctx, &atlas.UpdateInteractorInput{ProblemId: *pid, Interactor: interactor}); err != nil {
			log.Printf("Unable to update problem interactor: %v", err)
			return err
		}

		log.Printf("Updated interactor")
	} else {
		log.Printf("No interactor found")
	}

	testsetList, err := imp.GetTestsets(kpr)
	if err != nil {
		// TODO
	} else if len(testsetList) > 0 {
		// create testsets

		for _, group := range testsetList {
			oldTestset, ok := testsets[group.Name]
			xts := group.Testset
			if ok {
				xts.Id = oldTestset.Id
			}

			delete(testsets, group.Name)

			if xts.Id != "" {
				_, err = UpdateTestset(ctx, &atlas.UpdateTestsetInput{TestsetId: xts.Id, Testset: xts})
				if err != nil {
					log.Printf("Unable to create testset: %v", err)
					return err
				}

				log.Printf("Updated testset %v", xts.Id)
			} else {
				out, err := CreateTestset(ctx, &atlas.CreateTestsetInput{ProblemId: *pid, Testset: xts})
				if err != nil {
					log.Printf("Unable to create testset: %v", err)
					return err
				}

				xts.Id = out.Id

				log.Printf("Created testset %v", xts.Id)
			}

			// upload tests

			for ti, xtt := range group.Tests {
				oldTest, ok := tests[fmt.Sprint(xts.Index, "/", int32(ti+1))]
				if ok {
					xtt.Id = oldTest.Id
				}
				delete(tests, fmt.Sprint(xts.Index, "/", int32(ti+1)))

				if xtt.Id == "" {
					out, err := CreateTest(ctx, &atlas.CreateTestInput{TestsetId: xts.Id, Test: xtt})
					if err != nil {
						log.Printf("Unable to create test: %v", err)
						return err
					}

					xtt.Id = out.Id

					log.Printf("Created test %v", xtt.Id)
				} else {
					if _, err := UpdateTest(ctx, &atlas.UpdateTestInput{TestId: xtt.Id, Test: xtt}); err != nil {
						log.Printf("Unable to update test: %v", err)
						return err
					}

					log.Printf("Updated test %v", xtt.Id)
				}
			}

		}
	}

	// remove unused objects
	for _, test := range tests {
		log.Printf("Deleting unused test %v", test.Id)
		if _, err := DeleteTest(ctx, &atlas.DeleteTestInput{TestId: test.Id}); err != nil {
			log.Printf("Unable to delete test: %v", err)
			return err
		}
	}

	for _, testset := range testsets {
		log.Printf("Deleting unused testset %v", testset.Id)
		if _, err := atl.DeleteTestset(ctx, &atlas.DeleteTestsetInput{TestsetId: testset.Id}); err != nil {
			log.Printf("Unable to delete testset: %v", err)
			return err
		}
	}

	newStatements := map[string]*atlas.Statement{}

	statementList, err := imp.GetStatements(ctx, tw, conf.Source)

	// get all statements
	for _, statement := range statementList {
		newStatements[statement.GetLocale()] = statement
	}

	for _, statement := range newStatements {

		log.Printf("Updating language %v", statement.Locale)

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
			out, err := CreateStatement(ctx, &atlas.CreateStatementInput{ProblemId: *pid, Statement: xs})
			if err != nil {
				log.Printf("Unable to create statement: %v", err)
				return err
			}

			xs.Id = out.Id

			log.Printf("Created statement %v", xs.Id)
		} else {
			_, err = UpdateStatement(ctx, &atlas.UpdateStatementInput{StatementId: xs.Id, Statement: xs})
			if err != nil {
				log.Printf("Unable to create statement: %v", err)
				return err
			}

			log.Printf("Updated statement %v", xs.Id)
		}
	}

	// remove unused objects
	for _, statement := range statements {
		log.Printf("Deleting unused statement %v", statement.Id)
		if _, err := atl.DeleteStatement(ctx, &atlas.DeleteStatementInput{StatementId: statement.Id}); err != nil {
			log.Printf("Unable to delete statement: %v", err)
			return err
		}
	}

	// get all solutions
	solutionList, err := imp.GetSolutions()

	if err != nil {
		// TODO
	} else {

		for _, solution := range solutionList {

			xs, ok := solutions[solution.GetLocale()]
			if !ok {
				xs = solution
			} else {
				xs.Locale = solution.Locale
				xs.Content = solution.Content
				xs.Format = solution.Format
			}
			delete(solutions, solution.GetLocale())

			if xs.Id == "" {
				out, err := atl.CreateSolution(ctx, &atlas.CreateSolutionInput{ProblemId: *pid, Solution: xs})
				if err != nil {
					log.Printf("Unable to create solution: %v", err)
					return err
				}

				xs.Id = out.SolutionId

				log.Printf("Created solution %v", xs.Id)
			} else {
				_, err = atl.UpdateSolution(ctx, &atlas.UpdateSolutionInput{SolutionId: xs.Id, Solution: xs})
				if err != nil {
					log.Printf("Unable to create solution: %v", err)
					return err
				}

				log.Printf("Updated solution %v", xs.Id)
			}
		}
	}
	// remove unused objects
	for _, solution := range solutions {
		log.Printf("Deleting unused solution %v", solution.Id)
		if _, err := atl.DeleteSolution(ctx, &atlas.DeleteSolutionInput{SolutionId: solution.Id}); err != nil {
			log.Printf("Unable to delete solution: %v", err)
			return err
		}
	}

	log.Printf("Finished")

	return nil
}
