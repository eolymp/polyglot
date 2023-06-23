package main

import (
	"context"
	"fmt"
	"github.com/eolymp/go-sdk/eolymp/atlas"
	"github.com/eolymp/polyglot/cmd/eolymp-polyglot/types"
	"log"
)

func ImportProblem(path string, pid *string, skipTests bool, format string) error {

	var err error

	var imp types.Importer
	ctx := context.Background()
	edi := atlas.NewEditorialServiceHttpClient("https://api.eolymp.com/spaces/"+conf.SpaceId+"/problems/"+*pid, client)

	if format == "eolymp" {
		atl := atlas.NewAtlasHttpClient(SpaceIdToLink(conf.Eolymp.SpaceImport), client)
		imp, err = types.CreateEolympImporter(ctx, path, atl, edi)
	} else if format == "ejudge" {
		imp, err = types.CreateEjudgeImporter(path, ctx, tw, kpr)
	} else if format == "dots" {
		imp, err = types.CreateDotsImporter(path, ctx, tw, kpr)
	} else {
		imp, err = types.CreatePolygonImporter(path, ctx, tw, kpr)
	}

	if err != nil {
		return err
	}

	statements := map[string]*atlas.Statement{}
	editorials := map[string]*atlas.Editorial{}
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

		solout, err := edi.ListEditorials(ctx, &atlas.ListEditorialsInput{})
		if err != nil {
			log.Printf("Unable to list problem editorials in Atlas: %v", err)
			return err
		}

		log.Printf("Found %v existing editorials", len(solout.GetItems()))

		for _, s := range solout.GetItems() {
			editorials[s.GetLocale()] = s
		}

		tsout, err := atl.ListTestsets(ctx, &atlas.ListTestsetsInput{ProblemId: *pid})
		if err != nil {
			log.Printf("Unable to list problem testsets in Atlas: %v", err)
			return err
		}

		log.Printf("Found %v existing testsets", len(tsout.GetItems()))

		for _, ts := range tsout.GetItems() {
			testsets[ts.GetIndex()] = ts

			ttout, err := atl.ListTests(ctx, &atlas.ListTestsInput{TestsetId: ts.GetId(), ProblemId: *pid})
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
		_, err = atl.DeleteCodeTemplate(ctx, &atlas.DeleteCodeTemplateInput{TemplateId: template.Id, ProblemId: *pid})
		if err != nil {
			log.Printf("Unable to delete code template: %v", err)
			return err
		}
	}

	templates, err := imp.GetTemplates(pid)
	if err != nil {
		return err
	}

	for _, template := range templates {
		_, err = atl.CreateCodeTemplate(ctx, &atlas.CreateCodeTemplateInput{ProblemId: *pid, Template: template})
		if err != nil {
			log.Printf("Unable to create code template: %v", err)
			return err
		}
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

	if !skipTests {
		testsetList, err := imp.GetTestsets()
		if err != nil {
			log.Println(err)
			log.Println("Failed to get testsets")
			return err
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
					_, err = atl.UpdateTestset(ctx, &atlas.UpdateTestsetInput{TestsetId: xts.Id, ProblemId: *pid, Testset: xts})
					if err != nil {
						log.Printf("Unable to create testset: %v", err)
						return err
					}

					log.Printf("Updated testset %v", xts.Id)
				} else {
					out, err := atl.CreateTestset(ctx, &atlas.CreateTestsetInput{ProblemId: *pid, Testset: xts})
					if err != nil {
						log.Printf("Unable to create testset: %v", err)
						return err
					}

					xts.Id = out.Id

					log.Printf("Created testset %v", xts.Id)
				}

				// upload tests

				for _, xtt := range group.Tests {
					oldTest, ok := tests[fmt.Sprint(group.Name, "/", xtt.Index)]
					if ok {
						xtt.Id = oldTest.Id
					}
					delete(tests, fmt.Sprint(group.Name, "/", xtt.Index))

					if xtt.Id == "" {
						out, err := atl.CreateTest(ctx, &atlas.CreateTestInput{TestsetId: xts.Id, ProblemId: *pid, Test: xtt})
						if err != nil {
							log.Printf("Unable to create test: %v", err)
							return err
						}

						xtt.Id = out.TestId

						log.Printf("Created test %v", xtt.Id)
					} else {
						if _, err := atl.UpdateTest(ctx, &atlas.UpdateTestInput{TestId: xtt.Id, Test: xtt, TestsetId: xts.Id, ProblemId: *pid}); err != nil {
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
			if _, err := atl.DeleteTest(ctx, &atlas.DeleteTestInput{TestsetId: test.TestsetId, TestId: test.Id, ProblemId: *pid}); err != nil {
				log.Printf("Unable to delete test: %v", err)
				return err
			}
		}

		for _, testset := range testsets {
			log.Printf("Deleting unused testset %v", testset.Id)
			if _, err := atl.DeleteTestset(ctx, &atlas.DeleteTestsetInput{TestsetId: testset.Id, ProblemId: *pid}); err != nil {
				log.Printf("Unable to delete testset: %v", err)
				return err
			}
		}

	}

	newStatements := map[string]*atlas.Statement{}

	statementList, err := imp.GetStatements(conf.Source)
	if err != nil {
		log.Println(err)
		log.Println("Failed to get statements")
		return err
	}

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
			xs.Author = statement.Author
			xs.Source = statement.Source
		}

		delete(statements, statement.GetLocale())

		if xs.Id == "" {
			out, err := atl.CreateStatement(ctx, &atlas.CreateStatementInput{ProblemId: *pid, Statement: xs})
			if err != nil {
				log.Printf("Unable to create statement: %v", err)
				return err
			}

			xs.Id = out.StatementId

			log.Printf("Created statement %v", xs.Id)
		} else {
			_, err = atl.UpdateStatement(ctx, &atlas.UpdateStatementInput{StatementId: xs.Id, Statement: xs, ProblemId: *pid})
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
		if _, err := atl.DeleteStatement(ctx, &atlas.DeleteStatementInput{StatementId: statement.Id, ProblemId: *pid}); err != nil {
			log.Printf("Unable to delete statement: %v", err)
			return err
		}
	}
	oldAttachments, err := atl.ListAttachments(ctx, &atlas.ListAttachmentsInput{ProblemId: *pid})
	if err != nil {
		return err
	}
	for _, attachment := range oldAttachments.Items {
		_, err = atl.DeleteAttachment(ctx, &atlas.DeleteAttachmentInput{
			ProblemId:    *pid,
			AttachmentId: attachment.Id,
		})
		if err != nil {
			return err
		}
		log.Println(attachment.Name, "has been deleted")
	}

	attachments, err := imp.GetAttachments(pid)
	if err != nil {
		return err
	}

	for _, attachment := range attachments {
		_, err = atl.CreateAttachment(ctx, &atlas.CreateAttachmentInput{
			ProblemId:  *pid,
			Attachment: attachment,
		})
		if err != nil {
			return err
		}
		log.Println(attachment.Name, "has been uploaded")
	}

	log.Printf("Finished")

	return nil
}
