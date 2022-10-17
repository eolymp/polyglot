package main

import (
	"context"
	"encoding/json"
	"github.com/eolymp/go-sdk/eolymp/atlas"
	"io/ioutil"
	"log"
	"os"
	"time"
)

const RepeatNumber = 10
const TimeSleep = 10 * time.Second

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
