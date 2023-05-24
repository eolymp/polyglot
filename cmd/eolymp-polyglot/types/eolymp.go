package types

import (
	"context"
	"github.com/eolymp/go-sdk/eolymp/atlas"
	"github.com/eolymp/go-sdk/eolymp/executor"
)

type EolympImporter struct {
	Importer
	context    context.Context
	atlas      *atlas.AtlasService
	editorials *atlas.EditorialServiceService
	probId     string
}

func CreateEolympImporter(context context.Context, probId string, atlas *atlas.AtlasService, editorials *atlas.EditorialServiceService) (*EolympImporter, error) {
	importer := new(EolympImporter)
	importer.context = context
	importer.atlas = atlas
	importer.probId = probId
	importer.editorials = editorials
	return importer, nil
}

func (imp EolympImporter) GetVerifier() (*executor.Verifier, error) {
	out, err := imp.atlas.DescribeVerifier(imp.context, &atlas.DescribeVerifierInput{ProblemId: imp.probId})
	if err != nil {
		return nil, err
	}
	return out.Verifier, nil
}

func (imp EolympImporter) HasInteractor() bool {
	out, _ := imp.atlas.DescribeInteractor(imp.context, &atlas.DescribeInteractorInput{ProblemId: imp.probId})
	return len(out.Interactor.Source) > 0
}

func (imp EolympImporter) GetInteractor() (*executor.Interactor, error) {
	out, err := imp.atlas.DescribeInteractor(imp.context, &atlas.DescribeInteractorInput{ProblemId: imp.probId})
	if err != nil {
		return nil, err
	}
	return out.Interactor, nil
}

func (imp EolympImporter) GetStatements(string) ([]*atlas.Statement, error) {
	var statements []*atlas.Statement
	out, err := imp.atlas.ListStatements(imp.context, &atlas.ListStatementsInput{ProblemId: imp.probId})
	if err != nil {
		return nil, err
	}
	for _, statement := range out.GetItems() {
		statement.Id = ""
		statements = append(statements, statement)
	}
	return statements, nil
}

func (imp EolympImporter) GetSolutions() ([]*atlas.Editorial, error) {
	var solutions []*atlas.Editorial
	out, err := imp.editorials.ListEditorials(imp.context, &atlas.ListEditorialsInput{})
	if err != nil {
		return nil, err
	}
	for _, solution := range out.GetItems() {
		solution.Id = ""
		solutions = append(solutions, solution)
	}
	return solutions, nil
}

func (imp EolympImporter) GetTestsets() ([]*Group, error) {
	var groups []*Group
	out, err := imp.atlas.ListTestsets(imp.context, &atlas.ListTestsetsInput{ProblemId: imp.probId})
	if err != nil {
		return nil, err
	}
	for _, testset := range out.GetItems() {
		var group Group

		group.Name = testset.Index

		tests, err := imp.atlas.ListTests(imp.context, &atlas.ListTestsInput{TestsetId: testset.Id, ProblemId: imp.probId})
		if err != nil {
			return nil, err
		}
		for _, test := range tests.Items {
			test.Id = ""
			group.Tests = append(group.Tests, test)
		}

		testset.Id = ""
		group.Testset = testset

		groups = append(groups, &group)
	}
	return groups, nil
}

func (imp EolympImporter) GetTemplates(*string) ([]*atlas.Template, error) {
	var templates []*atlas.Template
	out, err := imp.atlas.ListCodeTemplates(imp.context, &atlas.ListCodeTemplatesInput{ProblemId: imp.probId})
	if err != nil {
		return nil, err
	}
	for _, template := range out.GetItems() {
		template.Id = ""
		templates = append(templates, template)
	}
	return templates, nil
}

func (imp EolympImporter) GetAttachments(*string) ([]*atlas.Attachment, error) {
	var attachments []*atlas.Attachment
	out, err := imp.atlas.ListAttachments(imp.context, &atlas.ListAttachmentsInput{ProblemId: imp.probId})
	if err != nil {
		return nil, err
	}
	for _, attachment := range out.GetItems() {
		attachment.Id = ""
		attachments = append(attachments, attachment)
	}
	return attachments, nil
}
