package types

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/eolymp/go-sdk/eolymp/atlas"
	"github.com/eolymp/go-sdk/eolymp/keeper"
	"github.com/eolymp/go-sdk/eolymp/typewriter"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const RepeatNumber = 10
const TimeSleep = 10 * time.Second

func UpdateContentWithPictures(ctx context.Context, tw *typewriter.TypewriterService, content, source string) (string, error) {
	exts := []string{".png", ".jpeg", ".jpg"}
	files := FindFilesWithExtension(source, exts)
	for _, file := range files {
		if strings.Contains(content, file) {
			data, err := ioutil.ReadFile(source + file)
			if err != nil {
				log.Println("Failed to read file " + file)
				return "", err
			}
			var output *typewriter.UploadAssetOutput
			for i := 0; i < RepeatNumber; i++ {
				output, err = tw.UploadAsset(ctx, &typewriter.UploadAssetInput{Filename: file, Data: data})
				if err == nil {
					break
				}
				log.Println("Error while uploading asset")
			}
			if err != nil {
				log.Println("Error while uploading asset")
				return "", err
			}
			content = strings.ReplaceAll(content, file, output.Link)
		}
	}
	return content, nil
}

func FindFilesWithExtension(path string, exts []string) []string {
	var files []string
	_ = filepath.Walk(path, func(path string, f os.FileInfo, _ error) error {
		for _, ext := range exts {
			r, err := regexp.Match(ext, []byte(f.Name()))
			if err == nil && r {
				files = append(files, f.Name())
			}
		}
		return nil
	})
	return files
}

func MakeLocale(lang string) (string, error) {
	switch lang {
	case "ukrainian", "russian", "english", "hungarian":
		return lang[:2], nil
	case "polish":
		return "pl", nil
	case "kazakh":
		return "kk", nil
	default:
		return lang, fmt.Errorf("unknown language %#v", lang)
	}
}

func MakeObject(path string, kpr *keeper.KeeperService) (key string, err error) {
	output, err := MakeObjectGetFile(path, kpr)
	if err != nil {
		return "", err
	}
	return output, err
}

func MakeObjectGetFile(path string, kpr *keeper.KeeperService) (key string, err error) {

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	return MakeObjectByData(data, kpr)
}

func MakeObjectByData(data []byte, kpr *keeper.KeeperService) (key string, err error) {
	return UploadObject(kpr, bytes.NewReader(data))
}

func UploadObject(kpr *keeper.KeeperService, reader io.Reader) (string, error) {

	data, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	h := sha1.New()
	h.Write(data)
	sha := hex.EncodeToString(h.Sum(nil))
	log.Println(sha)

	if val, ok := GetCacheValue(sha); ok {
		log.Println("Cached", val)
		return val, nil
	}

	size := len(data)

	maxSize := 5242880

	// single call API for a small object
	if size < maxSize {
		out, err := kpr.CreateObject(context.Background(), &keeper.CreateObjectInput{Data: data})
		if err != nil {
			return "", err
		}
		log.Println(out.Key)

		SetCacheValue(sha, out.Key)
		return out.Key, nil
	}

	// multipart upload for tests > 5MB
	upload, err := kpr.StartMultipartUpload(context.Background(), &keeper.StartMultipartUploadInput{})
	if err != nil {
		return "", err
	}

	var parts []*keeper.CompleteMultipartUploadInput_Part

	pos := 0
	for i := 1; ; i++ {
		if size <= pos {
			break
		}
		length := maxSize
		if size-pos < length {
			length = size - pos
		}

		log.Printf("Uploading part #%d, %d bytes", i, length)

		part, err := kpr.UploadPart(context.Background(), &keeper.UploadPartInput{
			ObjectId:   upload.GetObjectId(),
			UploadId:   upload.GetUploadId(),
			PartNumber: uint32(i),
			Data:       data[pos : pos+length],
		})

		if err != nil {
			return "", err
		}

		parts = append(parts, &keeper.CompleteMultipartUploadInput_Part{
			Number: uint32(i),
			Etag:   part.GetEtag(),
		})

		pos += maxSize
	}

	_, err = kpr.CompleteMultipartUpload(context.Background(), &keeper.CompleteMultipartUploadInput{
		ObjectId: upload.GetObjectId(),
		UploadId: upload.GetUploadId(),
		Parts:    parts,
	})

	if err != nil {
		return "", err
	}

	key, err := upload.GetObjectId(), nil
	SetCacheValue(sha, key)

	return key, err
}

func RemoveSpaces(data string) string {
	l := 0
	r := len(data)
	for l < r && (data[l] == ' ' || data[l] == '\n') {
		l++
	}
	for l < r && (data[r-1] == ' ' || data[r-1] == '\n') {
		r--
	}
	return data[l:r]
}

func GetTestsFromLocation(path string, kpr *keeper.KeeperService) ([]*atlas.Test, error) {
	testPaths, err := GetTestPathsFromLocation(path)
	if err != nil {
		return nil, err
	}

	testCounter := 0
	var tests []*atlas.Test

	for _, t := range testPaths {
		inputName, outputName := t.input, t.output
		log.Println(inputName, outputName)
		input, err := MakeObject(inputName, kpr)
		if err != nil {
			log.Printf("Unable to upload test input data to E-Olymp: %v", err)
			return nil, err
		}
		output, err := MakeObject(outputName, kpr)
		if err != nil {
			log.Printf("Unable to upload test output data to E-Olymp: %v", err)
			return nil, err
		}
		log.Printf("Uploaded test %d", testCounter+1)
		testCounter += 1
		test := &atlas.Test{}
		test.Index = int32(testCounter)
		test.InputObjectId = input
		test.AnswerObjectId = output
		tests = append(tests, test)
	}
	return tests, nil
}

type TestPath struct {
	input  string
	output string
}

func GetTestPathsFromLocation(path string) ([]TestPath, error) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}

	inputExtensions := map[string]bool{"": true, ".in": true, ".dat": true}
	outputExtensions := map[string]bool{".a": true, ".out": true, ".sol": true, ".ans": true}

	inputs := map[string]string{}
	outputs := map[string]string{}

	for _, file := range files {
		extension := filepath.Ext(file.Name())
		filename := file.Name()[:len(file.Name())-len(extension)]
		dest := filepath.Join(path, file.Name())
		if inputExtensions[extension] {
			inputs[filename] = dest
		} else if outputExtensions[extension] {
			outputs[filename] = dest
		} else {
			inputs[file.Name()] = dest
		}
	}

	keys := make([]string, 0)
	for k, _ := range inputs {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	var tests []TestPath

	for _, filename := range keys {
		inputName, ok1 := inputs[filename]
		outputName, ok2 := outputs[filename]
		if ok1 && ok2 {
			test := TestPath{input: inputName, output: outputName}
			tests = append(tests, test)
		}
	}
	return tests, nil
}

func GetTestsFromTexStatement(data string, kpr *keeper.KeeperService) (tests []*atlas.Test, err error) {
	split := strings.Split(data, "\\exmp{")
	for i, d := range split {
		if i == 0 {
			continue
		}
		tst := strings.Split(d, "}")
		inputData := RemoveSpaces(tst[0])
		outputData := RemoveSpaces(strings.Split(tst[1], "{")[1])
		input, err := MakeObjectByData([]byte(inputData), kpr)
		if err != nil {
			log.Printf("Unable to upload test input data to E-Olymp: %v", err)
			return nil, err
		}
		output, err := MakeObjectByData([]byte(outputData), kpr)
		if err != nil {
			log.Printf("Unable to upload test output data to E-Olymp: %v", err)
			return nil, err
		}
		log.Printf("Uploaded sample %d", i)
		test := &atlas.Test{}
		test.Index = int32(i)
		test.Example = true
		test.Score = 0
		test.InputObjectId = input
		test.AnswerObjectId = output
		tests = append(tests, test)
	}
	return tests, nil
}

func GetExamplesFromLocation(path string, kpr *keeper.KeeperService) (tests []*atlas.Test, err error) {
	tests, err = GetTestsFromLocation(path, kpr)
	if err != nil {
		return tests, err
	}
	for i := 0; i < len(tests); i++ {
		tests[i].Example = true
	}
	return tests, nil
}

func GetCacheValue(s string) (string, bool) {
	val, ok := GetCache()[s]
	return val, ok
}

func SetCacheValue(key string, value string) {
	cache := GetCache()
	log.Println("Set", key, value)
	cache[key] = value
	SaveCache(cache)
}

func SaveCache(data map[string]string) {
	json, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	ioutil.WriteFile("cache.json", json, 0644)
}

func GetCache() map[string]string {
	jsonFile, err := os.Open("cache.json")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			err = os.WriteFile("cache.json", []byte("{}"), 0644)
			if err != nil {
				log.Println("Can't create file")
				log.Fatal(err)
			}
		}
		jsonFile, err = os.Open("cache.json")
		if err != nil {
			log.Println("Can't open file")
			log.Fatal(err)
		}
	}
	defer jsonFile.Close()
	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		panic(err)
	}
	var result map[string]string
	json.Unmarshal(byteValue, &result)
	return result
}

func AddPointsToTests(g *Group) {
	for i := 0; i < len(g.Tests); i++ {
		score := 100 / len(g.Tests)
		if len(g.Tests)-i <= 100%len(g.Tests) {
			score += 1
		}
		g.Tests[i].Score = float32(score)
	}
}
