package main

import (
	"bytes"
	"context"
	"github.com/antchfx/xmlquery"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"time"
)

const RepeatNumberProblemUploads = 5
const TimeToSleep = 5 * time.Minute

func ImportContest(contestId string) error {
	data := GetData()
	problems := GetProblems(contestId)
	log.Println(problems)
	var problemList []map[string]interface{}
	ctx := context.Background()
	for _, problem := range problems {
		pid, err := CreateProblem(ctx)
		if err != nil {
			log.Println("Failed to create problem")
			return err
		}
		problemList = append(problemList, map[string]interface{}{"id": pid, "link": problem})
	}
	data[contestId] = problemList
	SaveData(data)
	log.Println(data)
	//UpdateContest(contestId)
	return nil
}

func UpdateContest(contestId string, firstProblem int) error {
	data := GetData()
	t := reflect.ValueOf(data[contestId])
	for i := firstProblem; i < t.Len(); i++ {
		m := t.Index(i).Elem()
		g := make(map[string]string)
		iter := m.MapRange()
		for iter.Next() {
			g[iter.Key().String()] = iter.Value().Elem().String()
		}
		pid := g["id"]
		log.Println(pid, g["link"])
		for j := 0; j < RepeatNumberProblemUploads; j++ {
			if err := DownloadAndImportProblem(g["link"], &pid); err != nil {
				log.Println(err)
				time.Sleep(TimeToSleep)
				if j+1 == RepeatNumberProblemUploads {
					log.Println("Failed to update problem", pid)
					return err
				}
			} else {
				break
			}
		}
	}
	return nil
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
