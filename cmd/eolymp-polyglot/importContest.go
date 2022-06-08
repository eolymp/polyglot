package main

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/antchfx/xmlquery"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"reflect"
)

func ImportContest(contestId string) {
	data := GetData()
	problems := GetProblems(contestId)
	log.Println(problems)
	var problemList []map[string]interface{}
	ctx := context.Background()
	for _, problem := range problems {
		pid, err := CreateProblem(ctx)
		if err != nil {
			panic(err)
		}
		problemList = append(problemList, map[string]interface{}{"id": pid, "link": problem})
	}
	data[contestId] = problemList
	SaveData(data)
	log.Println(data)
	UpdateContest(contestId)
}

func UpdateContest(contestId string) {
	data := GetData()
	t := reflect.ValueOf(data[contestId])
	for i := 0; i < t.Len(); i++ {
		m := t.Index(i).Elem()
		g := make(map[string]string)
		iter := m.MapRange()
		for iter.Next() {
			g[iter.Key().String()] = iter.Value().Elem().String()
		}
		pid := g["id"]
		log.Println(pid, g["link"])

		if err := DownloadAndImportProblem(g["link"], &pid); err != nil {
			log.Println(err)
		}
	}
}

func SaveData(data map[string]interface{}) {
	json, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	ioutil.WriteFile("data.json", json, 0644)
}

func GetData() map[string]interface{} {
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

func GetProblems(contestId string) []string {
	response, err := http.PostForm("https://polygon.codeforces.com/c/"+contestId+"/contest.xml", url.Values{"login": {conf.Polygon.Login}, "password": {conf.Polygon.Password}, "type": {"windows"}})
	if err != nil {
		return nil
	}
	defer func() {
		_ = response.Body.Close()
	}()
	buf := new(bytes.Buffer)
	buf.ReadFrom(response.Body)
	doc, err := xmlquery.Parse(buf)
	if err != nil {
		panic(err)
	}
	var result []string
	for _, n := range xmlquery.Find(doc, "//contest/problems/problem/@url") {
		result = append(result, n.InnerText())
	}
	return result
}
