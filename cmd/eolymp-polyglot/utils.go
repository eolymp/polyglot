package main

import (
	"context"
	"encoding/json"
	"github.com/eolymp/contracts/go/eolymp/atlas"
	"io/ioutil"
	"log"
	"os"
	"time"
)

const RepeatNumber = 10
const TimeSleep = time.Minute

func CreateProblem(ctx context.Context) (string, error) {
	for i := 0; i < RepeatNumber; i++ {
		pout, err := atl.CreateProblem(ctx, &atlas.CreateProblemInput{Problem: &atlas.Problem{}})
		if err == nil {
			log.Printf("Problem created with ID %#v", pout.ProblemId)
			return pout.ProblemId, nil
		}
		log.Printf("Unable to create problem: %v", err)
		time.Sleep(TimeSleep)
	}
	return "", nil
}

func CreateTestset(ctx context.Context, input *atlas.CreateTestsetInput) (*atlas.CreateTestsetOutput, error) {
	for i := 0; i < RepeatNumber; i++ {
		out, err := atl.CreateTestset(ctx, input)
		if err == nil {
			return out, nil
		}
		log.Printf("Error while creating testset: %v", err)
		time.Sleep(TimeSleep)
	}
	return atl.CreateTestset(ctx, input)
}

func UpdateTestset(ctx context.Context, input *atlas.UpdateTestsetInput) (*atlas.UpdateTestsetOutput, error) {
	for i := 0; i < RepeatNumber; i++ {
		out, err := atl.UpdateTestset(ctx, input)
		if err == nil {
			return out, nil
		}
		log.Printf("Error while updating testset: %v", err)
		time.Sleep(TimeSleep)
	}
	return atl.UpdateTestset(ctx, input)
}

func CreateTest(ctx context.Context, input *atlas.CreateTestInput) (*atlas.CreateTestOutput, error) {
	for i := 0; i < RepeatNumber; i++ {
		out, err := atl.CreateTest(ctx, input)
		if err == nil {
			return out, nil
		}
		log.Printf("Error while creating test: %v", err)
		time.Sleep(TimeSleep)
	}
	return atl.CreateTest(ctx, input)
}

func UpdateTest(ctx context.Context, input *atlas.UpdateTestInput) (*atlas.UpdateTestOutput, error) {
	for i := 0; i < RepeatNumber; i++ {
		out, err := atl.UpdateTest(ctx, input)
		if err == nil {
			return out, nil
		}
		log.Printf("Error while updating test: %v", err)
		time.Sleep(TimeSleep)
	}
	return atl.UpdateTest(ctx, input)
}

func DeleteTest(ctx context.Context, input *atlas.DeleteTestInput) (*atlas.DeleteTestOutput, error) {
	for i := 0; i < RepeatNumber; i++ {
		out, err := atl.DeleteTest(ctx, input)
		if err == nil {
			return out, nil
		}
		log.Printf("Error while deleting test: %v", err)
		time.Sleep(TimeSleep)
	}
	return atl.DeleteTest(ctx, input)
}

func CreateStatement(ctx context.Context, input *atlas.CreateStatementInput) (*atlas.CreateStatementOutput, error) {
	for i := 0; i < RepeatNumber; i++ {
		out, err := atl.CreateStatement(ctx, input)
		if err == nil {
			return out, nil
		}
		log.Printf("Error while creating statement: %v", err)
		time.Sleep(TimeSleep)
	}
	return atl.CreateStatement(ctx, input)
}

func UpdateStatement(ctx context.Context, input *atlas.UpdateStatementInput) (*atlas.UpdateStatementOutput, error) {
	for i := 0; i < RepeatNumber; i++ {
		out, err := atl.UpdateStatement(ctx, input)
		if err == nil {
			return out, nil
		}
		log.Printf("Error while updating statement: %v", err)
		time.Sleep(TimeSleep)
	}
	return atl.UpdateStatement(ctx, input)
}

func SaveData(data map[string]interface{}) {
	json, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	ioutil.WriteFile("data.json", json, 0644)
}

func GetData() map[string]interface{} {
	// TODO create file if does not exist
	jsonFile, err := os.Open("data.json")
	if err != nil {
		panic(err)
	}
	defer jsonFile.Close()
	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		panic(err)
	}
	var result map[string]interface{}
	json.Unmarshal(byteValue, &result)
	return result
}
